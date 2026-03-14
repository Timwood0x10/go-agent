package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// ReloadCallback is called when workflows are reloaded.
type ReloadCallback func(workflows map[string]*Workflow)

// callbackWithID wraps a ReloadCallback with an ID.
type callbackWithID struct {
	id string
	fn ReloadCallback
}

// FileWatcher watches files for changes.
type FileWatcher struct {
	watcher      *fsnotify.Watcher
	workflows    map[string]*Workflow
	loader       WorkflowLoader
	callbacks    []callbackWithID
	callbackID   uint64
	mu           sync.RWMutex
	pollInterval time.Duration
}

// NewFileWatcher creates a new FileWatcher.
func NewFileWatcher(loader WorkflowLoader, workflows map[string]*Workflow) *FileWatcher {
	return &FileWatcher{
		loader:       loader,
		workflows:    workflows,
		callbacks:    make([]callbackWithID, 0),
		pollInterval: 5 * time.Second,
	}
}

// Watch starts watching workflow files for changes.
func (w *FileWatcher) Watch(ctx context.Context, dir string) error {
	if err := w.scanAndLoad(ctx, dir); err != nil {
		return err
	}

	go w.watchLoop(ctx, dir)

	return nil
}

// watchLoop periodically checks for file changes.
func (w *FileWatcher) watchLoop(ctx context.Context, dir string) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.scanAndLoad(ctx, dir); err != nil {
				continue
			}
		}
	}
}

// scanAndLoad scans and loads workflows from directory.
func (w *FileWatcher) scanAndLoad(ctx context.Context, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read directory: %w", err)
	}

	newWorkflows := make(map[string]*Workflow)
	modified := false

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".json" && ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		stat, err := os.Stat(path)
		if err != nil {
			continue
		}

		workflow, err := w.loader.Load(ctx, path)
		if err != nil {
			continue
		}

		newWorkflows[workflow.ID] = workflow

		w.mu.RLock()
		oldWorkflow, exists := w.workflows[workflow.ID]
		w.mu.RUnlock()

		if !exists || stat.ModTime().After(oldWorkflow.UpdatedAt) {
			modified = true
		}
	}

	if modified {
		w.mu.Lock()
		w.workflows = newWorkflows
		w.mu.Unlock()

		w.notifyCallbacks()
	}

	return nil
}

// notifyCallbacks notifies all registered callbacks.
func (w *FileWatcher) notifyCallbacks() {
	w.mu.RLock()
	workflows := w.workflows
	callbacks := w.callbacks
	w.mu.RUnlock()

	for _, cb := range callbacks {
		cb.fn(workflows)
	}
}

// RegisterCallback registers a callback for reload events and returns the callback ID.
func (w *FileWatcher) RegisterCallback(callback ReloadCallback) string {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.callbackID++
	id := fmt.Sprintf("callback-%d", w.callbackID)
	w.callbacks = append(w.callbacks, callbackWithID{
		id: id,
		fn: callback,
	})
	return id
}

// UnregisterCallback removes a callback by ID.
func (w *FileWatcher) UnregisterCallback(callbackID string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for i, cb := range w.callbacks {
		if cb.id == callbackID {
			w.callbacks = append(w.callbacks[:i], w.callbacks[i+1:]...)
			return
		}
	}
}

// WorkflowReloader manages workflow hot reloading.
type WorkflowReloader struct {
	loader    WorkflowLoader
	workflows map[string]*Workflow
	callbacks []ReloadCallback
	mu        sync.RWMutex
	watcher   *FileWatcher
}

// NewWorkflowReloader creates a new WorkflowReloader.
func NewWorkflowReloader(loader WorkflowLoader) *WorkflowReloader {
	return &WorkflowReloader{
		loader:    loader,
		workflows: make(map[string]*Workflow),
		callbacks: make([]ReloadCallback, 0),
	}
}

// Load workflows from a directory.
func (r *WorkflowReloader) Load(ctx context.Context, dir string) error {
	loader, ok := r.loader.(*FileLoader)
	if !ok {
		return ErrInvalidLoader
	}

	dirLoader := NewDirectoryLoader(loader)
	workflows, err := dirLoader.LoadAll(ctx, dir)
	if err != nil {
		return fmt.Errorf("load workflows: %w", err)
	}

	r.mu.Lock()
	r.workflows = workflows
	r.mu.Unlock()

	return nil
}

// StartWatching starts watching for file changes.
func (r *WorkflowReloader) StartWatching(ctx context.Context, dir string) error {
	loader, ok := r.loader.(*FileLoader)
	if !ok {
		return ErrInvalidLoader
	}

	dirLoader := NewDirectoryLoader(loader)
	workflows, err := dirLoader.LoadAll(ctx, dir)
	if err != nil {
		return fmt.Errorf("load workflows: %w", err)
	}

	r.mu.Lock()
	r.workflows = workflows
	r.mu.Unlock()

	watcher := NewFileWatcher(r.loader, r.workflows)
	watcher.RegisterCallback(r.onReload)

	if err := watcher.Watch(ctx, dir); err != nil {
		return fmt.Errorf("start watcher: %w", err)
	}

	r.watcher = watcher

	return nil
}

// onReload handles workflow reload events.
func (r *WorkflowReloader) onReload(workflows map[string]*Workflow) {
	r.mu.Lock()
	r.workflows = workflows
	r.mu.Unlock()

	r.notifyCallbacks()
}

// notifyCallbacks notifies all registered callbacks.
func (r *WorkflowReloader) notifyCallbacks() {
	r.mu.RLock()
	workflows := r.workflows
	callbacks := r.callbacks
	r.mu.RUnlock()

	for _, callback := range callbacks {
		callback(workflows)
	}
}

// RegisterCallback registers a callback for reload events.
func (r *WorkflowReloader) RegisterCallback(callback ReloadCallback) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.callbacks = append(r.callbacks, callback)
}

// GetWorkflow returns a workflow by ID.
func (r *WorkflowReloader) GetWorkflow(id string) (*Workflow, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	workflow, exists := r.workflows[id]
	return workflow, exists
}

// ListWorkflows returns all loaded workflows.
func (r *WorkflowReloader) ListWorkflows() []*Workflow {
	r.mu.RLock()
	defer r.mu.RUnlock()

	workflows := make([]*Workflow, 0, len(r.workflows))
	for _, wf := range r.workflows {
		workflows = append(workflows, wf)
	}

	return workflows
}

// StopWatching stops watching for file changes.
func (r *WorkflowReloader) StopWatching() {
	if r.watcher != nil {
		r.watcher = nil
	}
}
