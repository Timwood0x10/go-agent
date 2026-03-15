package engine

import (
	"context"
	"fmt"
	"log/slog"
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

// FileWatcher watches files for changes using fsnotify.
type FileWatcher struct {
	watcher      *fsnotify.Watcher
	workflows    map[string]*Workflow
	loader       WorkflowLoader
	callbacks    []callbackWithID
	callbackID   uint64
	mu           sync.RWMutex
	pollInterval time.Duration // Fallback polling interval (only used if fsnotify fails)
}

// NewFileWatcher creates a new FileWatcher.
func NewFileWatcher(loader WorkflowLoader, workflows map[string]*Workflow) *FileWatcher {
	// Try to create fsnotify watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Warn("FileWatcher: fsnotify not available, falling back to polling", "error", err)
	} else {
		slog.Info("FileWatcher: using fsnotify for real-time file monitoring")
	}

	return &FileWatcher{
		watcher:      watcher,
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

	// If we have fsnotify watcher, use event-driven approach
	if w.watcher != nil {
		// Add directory to watch
		if err := w.watcher.Add(dir); err != nil {
			return fmt.Errorf("watch directory: %w", err)
		}

		// Watch subdirectories for workflow files
		w.watchDirectory(ctx, dir)

		go w.fsnotifyLoop(ctx, dir)
	} else {
		// Fallback to polling
		go w.watchLoop(ctx, dir)
	}

	return nil
}

// watchDirectory recursively adds directories to fsnotify watch.
func (w *FileWatcher) watchDirectory(ctx context.Context, dir string) {
	if w.watcher == nil {
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			path := filepath.Join(dir, entry.Name())
			if err := w.watcher.Add(path); err != nil {
				continue
			}
			// Recursively watch subdirectories
			w.watchDirectory(ctx, path)
		}
	}
}

// fsnotifyLoop watches for file change events.
func (w *FileWatcher) fsnotifyLoop(ctx context.Context, dir string) {
	defer func() {
		if w.watcher != nil {
			w.watcher.Close()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			// Only handle write and create events
			if event.Op&fsnotify.Write == 0 && event.Op&fsnotify.Create == 0 {
				continue
			}
			// Check if it's a workflow file
			ext := filepath.Ext(event.Name)
			if ext != ".json" && ext != ".yaml" && ext != ".yml" {
				continue
			}
			// Reload on file change
			if err := w.scanAndLoad(ctx, dir); err != nil {
				continue
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			slog.Error("FileWatcher error", "error", err)
		}
	}
}

// watchLoop periodically checks for file changes (fallback when fsnotify unavailable).
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
	loader     WorkflowLoader
	workflows  map[string]*Workflow
	callbackID uint64
	callbacks  map[string]ReloadCallback // Use map for O(1) lookup
	mu         sync.RWMutex
	watcher    *FileWatcher
}

// NewWorkflowReloader creates a new WorkflowReloader.
func NewWorkflowReloader(loader WorkflowLoader) *WorkflowReloader {
	return &WorkflowReloader{
		loader:    loader,
		workflows: make(map[string]*Workflow),
		callbacks: make(map[string]ReloadCallback),
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

// RegisterCallback registers a callback for reload events and returns the callback ID.
func (r *WorkflowReloader) RegisterCallback(callback ReloadCallback) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.callbackID++
	id := fmt.Sprintf("callback-%d", r.callbackID)
	r.callbacks[id] = callback
	return id
}

// UnregisterCallback removes a callback by ID.
func (r *WorkflowReloader) UnregisterCallback(callbackID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.callbacks, callbackID)
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
