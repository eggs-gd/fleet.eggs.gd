package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/manager"
	"github.com/eggs-gd/fleet.eggs.gd/internal/providerconfig"
	"github.com/eggs-gd/fleet.eggs.gd/internal/settings"
)

type Config struct {
	Addr               string
	CoreRoot           string
	DataRootSource     string
	RuntimeRoot        string
	BackofficeDir      string
	DryRun             bool
	SessionTimeout     time.Duration
	SessionTimeoutFlag bool
	Version            string
	LaunchToken        string
	StartedAt          time.Time
	DataRootCreated    bool
}

func Serve(cfg Config) error {
	if cfg.Addr == "" {
		cfg.Addr = "127.0.0.1:8787"
	}
	if strings.TrimSpace(cfg.LaunchToken) == "" {
		token, err := loadLaunchToken(cfg.RuntimeRoot)
		if err != nil {
			return fmt.Errorf("launch token: %w", err)
		}
		cfg.LaunchToken = token
	}
	if cfg.SessionTimeout <= 0 {
		cfg.SessionTimeout = 10 * time.Minute
	}
	if cfg.StartedAt.IsZero() {
		cfg.StartedAt = time.Now()
	}

	taskProviderSettings, err := providerconfig.Load(cfg.CoreRoot)
	if err != nil {
		return fmt.Errorf("load task provider config: %w", err)
	}

	coreRuntime := compose(cfg, taskProviderSettings)
	if err := coreRuntime.Bootstrap(); err != nil {
		return err
	}

	overlay, err := settings.LoadOverlay(cfg.CoreRoot)
	if err != nil {
		return fmt.Errorf("load %s: %w", settings.OverlayFileName, err)
	}
	settings.ApplyOverlay(overlay)

	// Contour 1 (Manager) and Dashboard both write through TaskService.
	// Dashboard additionally polls RuntimeService — it is outside the three
	// contours and must not subscribe to Go channels (CORE-114).
	managerService := manager.NewService(coreRuntime.TaskService(), boardReader{app: coreRuntime})
	dashboard := coreRuntime.Surface()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"ok":        true,
			"core_root": cfg.CoreRoot,
			"time":      time.Now().Format(time.RFC3339),
		})
	})
	registerDashboardRoutes(mux, dashboard)
	settingsRT := newSettingsRuntime(cfg, coreRuntime)
	registerSettingsRoutes(mux, settingsRT)
	registerManagerRoutes(mux, managerService, cfg)
	registerManagerProvisionRoutes(mux, cfg)
	if bound := managerBindingStatus(cfg.CoreRoot); bound.Bound {
		_, _ = settings.EnsureManagerMCP(cfg.CoreRoot, bound.Agent, cfg.Addr, cfg.LaunchToken)
	}
	registerManagerMCPRoute(mux, managerService)
	registerAppConfigRoutes(mux, cfg)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serveBackoffice(w, r, cfg.BackofficeDir, cfg.LaunchToken)
	})

	listener, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go settingsRT.scanner.Watch(ctx, cfg.CoreRoot, 2*time.Second)
	go coreRuntime.Run(ctx)

	launchMode := "dry-run"
	if !cfg.DryRun {
		launchMode = "live"
	}
	fmt.Printf("Core backoffice: http://%s/\n", cfg.Addr)
	fmt.Printf("Core root: %s\n", cfg.CoreRoot)
	fmt.Printf("Backoffice dir: %s\n", cfg.BackofficeDir)
	fmt.Printf("Launch mode: %s\n", launchMode)
	fmt.Printf("Session timeout: %s\n", cfg.SessionTimeout)
	return http.Serve(listener, guardLocal(cfg.Addr, cfg.LaunchToken, mux))
}

func registerManagerRoutes(mux *http.ServeMux, service *manager.Service, cfg Config) {
	coreRoot := cfg.CoreRoot
	runtimeRoot := cfg.RuntimeRoot
	mux.HandleFunc("/api/manager/schema", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		schema, err := manager.SchemaObject()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, schema)
	})
	mux.HandleFunc("/api/manager/vocabulary", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, service.Vocabulary())
	})
	mux.HandleFunc("/api/manager/binding", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, managerBindingStatus(coreRoot))
	})
	mux.HandleFunc("/api/manager/threads", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		agent := strings.TrimSpace(r.URL.Query().Get("agent"))
		if agent == "" {
			http.Error(w, "agent query parameter is required", http.StatusBadRequest)
			return
		}
		threads, err := listManagerThreads(r.Context(), runtimeRoot, agent)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		if cwd := strings.TrimSpace(r.URL.Query().Get("cwd")); cwd != "" {
			threads = filterThreadsByCwd(threads, cwd)
		}
		writeJSON(w, threads)
	})
	mux.HandleFunc("/api/manager/text", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req manager.TextRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeManagerResponse(w, service.SubmitText(r.Context(), req))
	})
	mux.HandleFunc("/api/manager/command", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req manager.CommandRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeManagerResponse(w, service.SubmitCommand(r.Context(), req))
	})
	mux.HandleFunc("/api/manager/audio", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseMultipartForm(16 << 20); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		file, header, err := r.FormFile("audio")
		if err != nil {
			http.Error(w, "audio file field is required", http.StatusBadRequest)
			return
		}
		defer file.Close()
		payload, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		contentType := header.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		source := r.FormValue("source")
		writeManagerResponse(w, service.SubmitAudio(r.Context(), payload, contentType, source))
	})
}

func writeManagerResponse(w http.ResponseWriter, response manager.Response) {
	status := http.StatusOK
	if !response.OK {
		status = http.StatusBadRequest
		if response.Failure != nil {
			switch response.Failure.Code {
			case manager.FailureSTTNotConfigured, manager.FailureLLMNotConfigured:
				status = http.StatusServiceUnavailable
			case manager.FailureNotFound:
				status = http.StatusNotFound
			case manager.FailureProviderError:
				status = http.StatusBadGateway
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

// boardReader adapts the live dashboard board projection for Manager vocabulary.
type boardReader struct {
	app *App
}

func (b boardReader) Board() manager.BoardView {
	if b.app == nil {
		return manager.BoardView{}
	}
	state := b.app.State()
	return manager.BoardView{
		Workspaces: state.Workspaces,
		Projects:   state.Projects,
		Registry:   state.Registry,
	}
}
