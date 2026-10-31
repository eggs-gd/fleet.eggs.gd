package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/appconfig"
	"github.com/eggs-gd/fleet.eggs.gd/internal/projectscan"
	"github.com/eggs-gd/fleet.eggs.gd/internal/server"
	"github.com/eggs-gd/fleet.eggs.gd/internal/settings"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskprovider/markdown"
	"github.com/eggs-gd/fleet.eggs.gd/internal/webui"
)

// version is set by the build (-ldflags "-X main.version=...").
var version = "0.0.1"

func main() {
	log.SetFlags(0)

	if len(os.Args) < 2 {
		usage(os.Stderr)
		os.Exit(2)
	}

	switch os.Args[1] {
	case "help", "-h", "--help", "-help":
		usage(os.Stdout)
	case "rebuild-index":
		rebuildIndex(os.Args[2:])
	case "scan":
		scanProjects(os.Args[2:])
	case "serve":
		serve(os.Args[2:])
	case "version":
		fmt.Println(version)
	default:
		log.Printf("unknown command: %s", os.Args[1])
		usage(os.Stderr)
		os.Exit(2)
	}
}

func scanProjects(args []string) {
	flags := flag.NewFlagSet("scan", flag.ExitOnError)
	root := flags.String("root", "", "Data root (default: saved app config, else ~/.fleet/workspace)")
	var projects stringList
	flags.Var(&projects, "projects", "absolute directory to scan; repeat for several roots")
	flags.Parse(args) //nolint:errcheck // ExitOnError exits on a bad flag itself

	coreRoot, _, err := appconfig.ResolveDataRoot(*root)
	if err != nil {
		log.Fatal(err)
	}
	if len(projects) == 0 {
		log.Fatal("scan root is required: pass --projects")
	}
	result, err := projectscan.Scan(projectscan.ScanOptions{
		ScanRoots:   projects,
		RegistryDir: filepath.Join(coreRoot, "_registry"),
		WorkDir:     filepath.Join(coreRoot, "Work"),
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Discovered %d repositories under %s\n", result.Repositories, strings.Join(result.Roots, ", "))
	fmt.Printf("Found %d deletion review candidates\n", result.DeletionCandidates)
	fmt.Printf("Wrote _registry to %s\n", result.RegistryDir)
}

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }

func (s *stringList) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("scan root is empty")
	}
	*s = append(*s, value)
	return nil
}

func rebuildIndex(args []string) {
	flags := flag.NewFlagSet("rebuild-index", flag.ExitOnError)
	root := flags.String("root", "", "Data root (default: saved app config, else ~/.fleet_data)")
	flags.Parse(args) //nolint:errcheck // ExitOnError exits on a bad flag itself

	coreRoot, _, err := appconfig.ResolveDataRoot(*root)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := appconfig.EnsureDataRoot(coreRoot); err != nil {
		log.Fatal(err)
	}
	if err := markdown.RebuildWorkIndex(coreRoot); err != nil {
		log.Fatal(err)
	}
}

func serve(args []string) {
	flags := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := flags.String("addr", "127.0.0.1:8787", "HTTP listen address")
	root := flags.String("root", "", "Data root (default: saved app config, else ~/.fleet/workspace)")
	backofficeDir := flags.String("backoffice-dir", "", "serve the dashboard from this directory instead of the one built into the binary (for development)")
	live := flags.Bool("live", false, "start AI agents for ready tasks. They edit files without asking each time")
	dryRun := flags.Bool("dry-run", false, "only plan agent launches (the default)")
	sessionTimeout := flags.Duration("session-timeout", 10*time.Minute, "idle attention threshold for controllable agent sessions; resets on activity and is not a total-session cap")
	flags.Parse(args) //nolint:errcheck // ExitOnError exits on a bad flag itself
	sessionTimeoutFlag := false
	launchFlag := ""
	flags.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "session-timeout":
			sessionTimeoutFlag = true
		case "live", "dry-run":
			launchFlag = "--" + f.Name
		}
	})

	coreRoot, rootSource, err := appconfig.ResolveDataRoot(*root)
	if err != nil {
		log.Fatal(err)
	}
	created, err := appconfig.EnsureDataRoot(coreRoot)
	if err != nil {
		log.Fatal(err)
	}
	if created {
		log.Printf("bootstrapped new Data root at %s (source: %s)", coreRoot, rootSource)
	}

	runtimeRoot, err := appconfig.EnsureRuntimeRoot()
	if err != nil {
		log.Fatal(err)
	}

	dashboard, dashboardSource, err := resolveDashboard(*backofficeDir)
	if err != nil {
		log.Fatal(err)
	}

	overlay, err := settings.LoadOverlay(coreRoot)
	if err != nil {
		log.Fatal(err)
	}
	timeout, err := settings.ResolveServeTimeout(sessionTimeoutFlag, *sessionTimeout, overlay)
	if err != nil {
		log.Fatal(err)
	}
	planOnly, err := settings.ResolveLaunchMode(*live, *dryRun, overlay)
	if err != nil {
		log.Fatal(err)
	}

	cfg := server.Config{
		Addr:               *addr,
		CoreRoot:           coreRoot,
		DataRootSource:     rootSource,
		RuntimeRoot:        runtimeRoot,
		Backoffice:         dashboard,
		BackofficeSource:   dashboardSource,
		DryRun:             planOnly,
		LaunchFlag:         launchFlag,
		SessionTimeout:     timeout,
		SessionTimeoutFlag: sessionTimeoutFlag,
		Version:            version,
		DataRootCreated:    created,
	}
	if err := server.Serve(cfg); err != nil {
		log.Fatal(err)
	}
}

// resolveDashboard picks the dashboard to serve: a directory named with
// --backoffice-dir (for development), otherwise the one built into the binary.
// A relative directory resolves against the working directory, like any other
// path argument.
func resolveDashboard(dir string) (fs.FS, string, error) {
	if dir != "" {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return nil, "", err
		}
		return os.DirFS(abs), abs, nil
	}
	if fsys, ok := webui.Embedded(); ok {
		return fsys, "built into this binary", nil
	}
	return nil, "", errors.New("this binary was built without the dashboard: run `make build`, or pass --backoffice-dir <dir with index.html>")
}

const usageText = `usage:
  fleet serve [--addr 127.0.0.1:8787] [--root <data-root>] [--live | --dry-run] [--session-timeout 10m]
  fleet scan [--root <data-root>] --projects <scan-root> [--projects <scan-root>...]
  fleet rebuild-index [--root <data-root>]
  fleet version

serve only plans agent launches unless you pass --live or turn live mode on in
Settings. Live mode starts agents that edit files without asking each time.
--root is optional: without it, Fleet uses the Data root saved in
~/.fleet/app.json, defaulting to and creating ~/.fleet/workspace on first run.
`

func usage(w io.Writer) {
	_, _ = io.WriteString(w, usageText) // a failed usage write has nowhere to be reported
}
