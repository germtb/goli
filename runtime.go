// Package goli provides the reactive TUI framework runtime.
package goli

import "sync"

// computation tracks a reactive computation (effect or memo).
type computation struct {
	execute       func()
	subscriptions []subscriber // Signals this computation is subscribed to
	mu            sync.Mutex
}

// subscriber interface allows signals to be unsubscribed from.
type subscriber interface {
	unsubscribe(comp *computation)
}

// Owner tracks disposables for cleanup.
type Owner struct {
	disposables []func()
}

// Runtime holds all global mutable state for the goli framework.
// This enables easy state clearing for tests via Reset().
type Runtime struct {
	mu sync.Mutex

	// Reactive context (moved from signals package)
	currentComputation *computation
	currentOwner       *Owner
	batchDepth         int
	pendingComputations map[*computation]struct{}

	// Focus management (moved from focus.go)
	focusManager *FocusManager
}

// Global is the package-level runtime instance.
var Global *Runtime

func init() {
	Global = NewRuntime()
}

// NewRuntime creates a new Runtime with initialized state.
func NewRuntime() *Runtime {
	rt := &Runtime{
		pendingComputations: make(map[*computation]struct{}),
	}
	// focusManager will be lazily initialized when first accessed
	return rt
}

// Reset clears and reinitializes the global runtime.
// Call this at the start of tests for clean isolation.
func Reset() {
	Global = NewRuntime()
}

// FocusManager returns the focus manager, creating it if needed.
func (rt *Runtime) FocusManager() *FocusManager {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	if rt.focusManager == nil {
		current, setCurrent := createSignalInternal[Focusable](rt, nil)
		rt.focusManager = &FocusManager{
			currentFocused:    current,
			setCurrentFocused: setCurrent,
			registered:        make([]Focusable, 0),
		}
	}
	return rt.focusManager
}

// getCurrentComputation returns the current computation being tracked.
func (rt *Runtime) getCurrentComputation() *computation {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.currentComputation
}

// setCurrentComputation sets the current computation.
func (rt *Runtime) setCurrentComputation(comp *computation) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.currentComputation = comp
}

// getCurrentOwner returns the current owner.
func (rt *Runtime) getCurrentOwner() *Owner {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.currentOwner
}

// setCurrentOwner sets the current owner.
func (rt *Runtime) setCurrentOwner(owner *Owner) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.currentOwner = owner
}

// getBatchDepth returns the current batch depth.
func (rt *Runtime) getBatchDepth() int {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.batchDepth
}

// incrementBatchDepth increments the batch depth.
func (rt *Runtime) incrementBatchDepth() {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.batchDepth++
}

// decrementBatchDepth decrements the batch depth and returns true if we should flush.
func (rt *Runtime) decrementBatchDepth() bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.batchDepth--
	return rt.batchDepth == 0
}

// addPendingComputation adds a computation to the pending set.
func (rt *Runtime) addPendingComputation(comp *computation) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.pendingComputations[comp] = struct{}{}
}

// flushPending runs all pending computations and clears the set.
func (rt *Runtime) flushPending() {
	rt.mu.Lock()
	toRun := make([]*computation, 0, len(rt.pendingComputations))
	for comp := range rt.pendingComputations {
		toRun = append(toRun, comp)
	}
	rt.pendingComputations = make(map[*computation]struct{})
	rt.mu.Unlock()

	for _, comp := range toRun {
		comp.execute()
	}
}
