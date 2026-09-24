package markdown

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

var taskEditLocks sync.Map

// Hooks lets a caller keep a dashboard projection, derived index files, and
// the event log in sync with Markdown mutations without the provider
// depending on corechain directly.
type Hooks struct {
	AfterMutate  func(task tasklifecycle.Task, eventType string, message string, details map[string]any) error
	AfterObserve func(ObservedTaskChange) error
	OnEvent      func(task tasklifecycle.Task, eventType string, message string)
}

// Provider is the Markdown/file-backed task persistence adapter.
// It implements taskflow.TaskProvider and taskprovider.Provider.
type Provider struct {
	root  string
	hooks Hooks
}

// New creates a Markdown Provider rooted at root. Any Hooks field may be nil.
func New(root string, hooks Hooks) *Provider {
	return &Provider{root: root, hooks: hooks}
}

// PatchTaskFile is a convenience one-shot patch with no hooks.
func PatchTaskFile(root string, patch tasklifecycle.TaskPatch) (tasklifecycle.Task, error) {
	return New(root, Hooks{}).Mutate(patch)
}

// ClaimTaskForLaunch is a convenience one-shot claim with no hooks.
func ClaimTaskForLaunch(root string, path string) (tasklifecycle.Task, error) {
	return New(root, Hooks{}).Claim(path)
}

func (p *Provider) Type() string { return "markdown" }

func (p *Provider) Load(path string) (tasklifecycle.Task, error) {
	taskPath, err := safeTaskPath(p.root, path)
	if err != nil {
		return tasklifecycle.Task{}, err
	}
	return LoadTaskFile(p.root, taskPath)
}

// Reload reloads one task card after a filesystem change. Prefer
// ObserveFileChange when the caller also needs a normalized TaskEvent
// for the execution chain (CORE-103).
func (p *Provider) Reload(path string) (tasklifecycle.Task, error) {
	return p.Load(path)
}

func (p *Provider) List() ([]tasklifecycle.Task, error) {
	matches, err := filepath.Glob(filepath.Join(p.root, "Work", "*", "tasks", "*.md"))
	if err != nil {
		return nil, err
	}

	tasks := make([]tasklifecycle.Task, 0, len(matches))
	for _, path := range matches {
		if filepath.Base(path) == "README.md" {
			continue
		}
		task, err := LoadTaskFile(p.root, path)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	sort.Slice(tasks, func(i, j int) bool {
		return tasklifecycle.TaskPickupLess(tasks[i], tasks[j])
	})
	return tasks, nil
}

// CreateFile writes a brand-new task card verbatim at create.Path.
// Prefer CreateFromRequest / taskflow Create for normal application creates.
func (p *Provider) CreateFile(create tasklifecycle.TaskCreate) (tasklifecycle.Task, error) {
	taskPath, err := safeTaskPath(p.root, create.Path)
	if err != nil {
		return tasklifecycle.Task{}, err
	}
	lock := lockTaskFile(taskPath)
	defer lock.Unlock()

	if _, err := os.Stat(taskPath); err == nil {
		return tasklifecycle.Task{}, fmt.Errorf("%w: task already exists", tasklifecycle.ErrTaskConflict)
	} else if !errors.Is(err, os.ErrNotExist) {
		return tasklifecycle.Task{}, err
	}

	if err := writeFileAtomic(taskPath, []byte(create.Content), 0o644); err != nil {
		return tasklifecycle.Task{}, err
	}
	return p.finalizeMutation(taskPath, "task_created", "Task file created through MarkdownProvider", nil)
}

func (p *Provider) Mutate(patch tasklifecycle.TaskPatch) (tasklifecycle.Task, error) {
	taskPath, err := safeTaskPath(p.root, patch.Path)
	if err != nil {
		return tasklifecycle.Task{}, err
	}
	lock := lockTaskFile(taskPath)
	defer lock.Unlock()

	return p.mutateLocked(taskPath, patch)
}

func (p *Provider) Transition(path string, status string) (tasklifecycle.Task, error) {
	return p.Mutate(tasklifecycle.TaskPatch{
		Path:   path,
		Status: status,
	})
}

func (p *Provider) Claim(path string) (tasklifecycle.Task, error) {
	taskPath, err := safeTaskPath(p.root, path)
	if err != nil {
		return tasklifecycle.Task{}, err
	}
	lock := lockTaskFile(taskPath)
	defer lock.Unlock()

	task, data, err := p.loadCurrentLocked(taskPath)
	if err != nil {
		return tasklifecycle.Task{}, err
	}
	pickupStatus := task.Status
	if !tasklifecycle.LaunchablePickupStatus(task.Status) {
		if p.hooks.OnEvent != nil {
			p.hooks.OnEvent(task, "task_claim_rejected", "Task claim rejected because status is not a pickup status")
		}
		return tasklifecycle.Task{}, fmt.Errorf("%w: status is %q", tasklifecycle.ErrTaskAlreadyClaimed, task.Status)
	}

	claimed, err := p.mutateLoadedLocked(taskPath, data, task, tasklifecycle.TaskPatch{
		Path:   task.Path,
		Status: "doing",
	})
	if err != nil {
		return tasklifecycle.Task{}, err
	}
	return p.finalizeLoadedMutation(claimed, "task_claimed", fmt.Sprintf("Task claimed for launch from %s", pickupStatus), map[string]any{
		"pickup_status": pickupStatus,
	})
}

func (p *Provider) mutateLocked(taskPath string, patch tasklifecycle.TaskPatch) (tasklifecycle.Task, error) {
	task, data, err := p.loadCurrentLocked(taskPath)
	if err != nil {
		return tasklifecycle.Task{}, err
	}
	mutated, err := p.mutateLoadedLocked(taskPath, data, task, patch)
	if err != nil {
		return tasklifecycle.Task{}, err
	}
	return p.finalizeLoadedMutation(mutated, "task_patched", "Task file patched through MarkdownProvider", map[string]any{
		"stage":         "status-update",
		"source":        "markdown_provider",
		"index_rebuilt": true,
		"changes":       tasklifecycle.TaskPatchChangeSummary(task, mutated, patch),
		"previous": map[string]any{
			"status":   task.Status,
			"assignee": task.Assignee,
			"priority": task.Priority,
			"project":  firstNonEmpty(task.ProjectID, task.Project),
			"path":     task.RelativePath,
		},
		"current": map[string]any{
			"status":   mutated.Status,
			"assignee": mutated.Assignee,
			"priority": mutated.Priority,
			"project":  firstNonEmpty(mutated.ProjectID, mutated.Project),
			"path":     mutated.RelativePath,
		},
		"patch": map[string]any{
			"status":     patch.Status,
			"assignee":   patch.Assignee,
			"priority":   patch.Priority,
			"project":    patch.Project,
			"repository": patch.Repository,
			"comment":    strings.TrimSpace(patch.Comment) != "",
			"body":       patch.Body != nil,
		},
	})
}

func (p *Provider) loadCurrentLocked(taskPath string) (tasklifecycle.Task, []byte, error) {
	data, err := os.ReadFile(taskPath)
	if err != nil {
		return tasklifecycle.Task{}, nil, err
	}
	task, err := LoadTaskFile(p.root, taskPath)
	if err != nil {
		return tasklifecycle.Task{}, nil, err
	}
	return task, data, nil
}

func (p *Provider) mutateLoadedLocked(taskPath string, data []byte, current tasklifecycle.Task, patch tasklifecycle.TaskPatch) (tasklifecycle.Task, error) {
	if err := validateExpectedTaskVersion(current, data, patch); err != nil {
		return tasklifecycle.Task{}, err
	}
	frontmatter, body, err := splitMarkdown(string(data))
	if err != nil {
		return tasklifecycle.Task{}, err
	}

	if patch.Status != "" {
		if !tasklifecycle.IsKnownTaskStatus(patch.Status) {
			return tasklifecycle.Task{}, fmt.Errorf("unknown task status %q", patch.Status)
		}
		if !patch.AllowStatusOverride && !tasklifecycle.AllowedTaskStatusTransition(current.Status, patch.Status) {
			return tasklifecycle.Task{}, fmt.Errorf("%w: %s -> %s", tasklifecycle.ErrTaskTransitionRejected, current.Status, patch.Status)
		}
		frontmatter, err = setFrontmatterValue(frontmatter, "status", patch.Status)
		if err != nil {
			return tasklifecycle.Task{}, err
		}
	}
	if patch.Assignee != "" {
		frontmatter, err = setFrontmatterValue(frontmatter, "assignee", patch.Assignee)
		if err != nil {
			return tasklifecycle.Task{}, err
		}
	}
	if patch.Priority != nil {
		if *patch.Priority < 1 || *patch.Priority > 5 {
			return tasklifecycle.Task{}, fmt.Errorf("unknown task priority %d", *patch.Priority)
		}
		frontmatter, err = setFrontmatterValue(frontmatter, "priority", fmt.Sprintf("%d", *patch.Priority))
		if err != nil {
			return tasklifecycle.Task{}, err
		}
	}
	if patch.DependsOn != nil {
		frontmatter, err = setFrontmatterDependsOn(frontmatter, tasklifecycle.NormalizeDependsOn(*patch.DependsOn))
		if err != nil {
			return tasklifecycle.Task{}, err
		}
	}

	writePath := taskPath
	var destinationUnlock func()
	if projectPatch := strings.TrimSpace(patch.Project); projectPatch != "" {
		if strings.Contains(projectPatch, "..") || strings.ContainsAny(projectPatch, `\:`) {
			return tasklifecycle.Task{}, fmt.Errorf("project id %q is invalid", projectPatch)
		}
		workProjectID := workProjectDirID(projectPatch)
		if err := validateProjectForCreate(p.root, workProjectID); err != nil {
			return tasklifecycle.Task{}, err
		}
		frontmatter, err = setFrontmatterValue(frontmatter, "project", workProjectID)
		if err != nil {
			return tasklifecycle.Task{}, err
		}
		repositories := []string{}
		if repo := strings.TrimSpace(patch.Repository); repo != "" {
			repositories = []string{repo}
		}
		frontmatter, err = setFrontmatterRepositories(frontmatter, repositories)
		if err != nil {
			return tasklifecycle.Task{}, err
		}

		currentWorkProjectID := filepath.Base(filepath.Dir(filepath.Dir(taskPath)))
		if workProjectID != currentWorkProjectID {
			destPath, destErr := uniqueTaskDestinationPath(p.root, workProjectID, filepath.Base(taskPath))
			if destErr != nil {
				return tasklifecycle.Task{}, destErr
			}
			if filepath.Clean(destPath) != filepath.Clean(taskPath) {
				lock := lockTaskFile(destPath)
				destinationUnlock = lock.Unlock
				writePath = destPath
			}
		}
	}
	if destinationUnlock != nil {
		defer destinationUnlock()
	}

	frontmatter, err = setFrontmatterValue(frontmatter, "updated_at", time.Now().Format(time.RFC3339))
	if err != nil {
		return tasklifecycle.Task{}, err
	}

	if patch.Body != nil {
		body = strings.TrimSpace(*patch.Body) + "\n"
	}
	if strings.TrimSpace(patch.Comment) != "" {
		body = appendReviewComment(body, patch.CommentAuthor, patch.Comment)
	}

	if err := writeFileAtomic(writePath, []byte("---\n"+frontmatter+"---\n\n"+body), 0o644); err != nil {
		return tasklifecycle.Task{}, err
	}
	if writePath != taskPath {
		if err := os.Remove(taskPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return tasklifecycle.Task{}, fmt.Errorf("remove old task path after project move: %w", err)
		}
	}

	return LoadTaskFile(p.root, writePath)
}

func uniqueTaskDestinationPath(root string, workProjectID string, fileName string) (string, error) {
	tasksDir := filepath.Join(root, "Work", workProjectID, "tasks")
	candidate := filepath.Join(tasksDir, fileName)
	if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
		return candidate, nil
	} else if err != nil {
		return "", err
	}

	ext := filepath.Ext(fileName)
	base := strings.TrimSuffix(fileName, ext)
	for i := 2; ; i++ {
		candidate = filepath.Join(tasksDir, fmt.Sprintf("%s-%d%s", base, i, ext))
		_, err := os.Stat(candidate)
		if errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
		if i > 1000 {
			return "", fmt.Errorf("could not allocate unique task destination for %s", fileName)
		}
	}
}

func (p *Provider) finalizeMutation(taskPath string, eventType string, message string, details map[string]any) (tasklifecycle.Task, error) {
	task, err := LoadTaskFile(p.root, taskPath)
	if err != nil {
		return tasklifecycle.Task{}, err
	}
	return p.finalizeLoadedMutation(task, eventType, message, details)
}

func (p *Provider) finalizeLoadedMutation(task tasklifecycle.Task, eventType string, message string, details map[string]any) (tasklifecycle.Task, error) {
	if p.hooks.AfterMutate != nil {
		if err := p.hooks.AfterMutate(task, eventType, message, details); err != nil {
			return tasklifecycle.Task{}, err
		}
	}
	return task, nil
}

func validateExpectedTaskVersion(current tasklifecycle.Task, data []byte, patch tasklifecycle.TaskPatch) error {
	if patch.ExpectedUpdatedAt != "" && patch.ExpectedUpdatedAt != current.UpdatedAt {
		return fmt.Errorf("%w: updated_at is %q, expected %q", tasklifecycle.ErrTaskConflict, current.UpdatedAt, patch.ExpectedUpdatedAt)
	}
	if patch.ExpectedHash != "" {
		actual := taskContentHash(data)
		if patch.ExpectedHash != actual {
			return fmt.Errorf("%w: hash is %q, expected %q", tasklifecycle.ErrTaskConflict, actual, patch.ExpectedHash)
		}
	}
	return nil
}

func taskContentHash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tempPath := file.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()

	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Chmod(perm); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func lockTaskFile(path string) *sync.Mutex {
	clean := filepath.Clean(path)
	value, _ := taskEditLocks.LoadOrStore(clean, &sync.Mutex{})
	lock := value.(*sync.Mutex)
	lock.Lock()
	return lock
}

func safeTaskPath(root string, path string) (string, error) {
	if path == "" {
		return "", errors.New("task path is required")
	}

	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}

	taskPath := path
	if !filepath.IsAbs(taskPath) {
		taskPath = filepath.Join(root, taskPath)
	}
	taskPath, err = filepath.Abs(taskPath)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(root, taskPath)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return "", errors.New("task path is outside Core root")
	}
	if !IsTaskFile(taskPath) {
		return "", errors.New("path is not a Core task markdown file")
	}
	return taskPath, nil
}
