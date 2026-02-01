// Package signals provides fine-grained reactive primitives.
//
// Key principles:
// - Components run ONCE (setup phase)
// - Signals created inside components are local to that instance
// - Fine-grained reactivity: only re-run what depends on changed signals
// - No rules of hooks - signals are just values
package signals

import "sync"

// Accessor is a function that reads a signal value.
type Accessor[T any] func() T

// Setter is a function that updates a signal value.
type Setter[T any] func(T)

// SetterFunc updates based on previous value.
type SetterFunc[T any] func(prev T) T

// computation tracks a reactive computation (effect or memo).
type computation struct {
	execute      func()
	dependencies map[*signal[any]]struct{}
	mu           sync.Mutex
}

// signal is the internal signal implementation.
type signal[T any] struct {
	value       T
	subscribers map[*computation]struct{}
	mu          sync.RWMutex
}

// Global reactive context
var (
	currentComputation   *computation
	currentComputationMu sync.Mutex

	batchDepth          int
	batchMu             sync.Mutex
	pendingComputations = make(map[*computation]struct{})
)

// CreateSignal creates a reactive signal.
//
// Example:
//
//	count, setCount := CreateSignal(0)
//	fmt.Println(count()) // 0
//	setCount(1)
//	fmt.Println(count()) // 1
func CreateSignal[T any](initialValue T) (Accessor[T], Setter[T]) {
	s := &signal[T]{
		value:       initialValue,
		subscribers: make(map[*computation]struct{}),
	}

	read := func() T {
		s.mu.RLock()
		val := s.value
		s.mu.RUnlock()

		// Track this signal as a dependency of current computation
		currentComputationMu.Lock()
		comp := currentComputation
		currentComputationMu.Unlock()

		if comp != nil {
			s.mu.Lock()
			s.subscribers[comp] = struct{}{}
			s.mu.Unlock()

			comp.mu.Lock()
			// Store as any to work with typed signals
			comp.dependencies[(*signal[any])(nil)] = struct{}{}
			comp.mu.Unlock()
		}

		return val
	}

	write := func(newValue T) {
		s.mu.Lock()
		// Simple equality check for comparable types
		s.value = newValue

		// Get subscribers to notify
		subs := make([]*computation, 0, len(s.subscribers))
		for comp := range s.subscribers {
			subs = append(subs, comp)
		}
		s.mu.Unlock()

		// Notify subscribers
		batchMu.Lock()
		inBatch := batchDepth > 0
		batchMu.Unlock()

		if inBatch {
			batchMu.Lock()
			for _, comp := range subs {
				pendingComputations[comp] = struct{}{}
			}
			batchMu.Unlock()
		} else {
			for _, comp := range subs {
				comp.execute()
			}
		}
	}

	return read, write
}

// CreateSignalWithEquals creates a signal with a custom equality function.
func CreateSignalWithEquals[T any](initialValue T, equals func(a, b T) bool) (Accessor[T], Setter[T]) {
	s := &signal[T]{
		value:       initialValue,
		subscribers: make(map[*computation]struct{}),
	}

	read := func() T {
		s.mu.RLock()
		val := s.value
		s.mu.RUnlock()

		currentComputationMu.Lock()
		comp := currentComputation
		currentComputationMu.Unlock()

		if comp != nil {
			s.mu.Lock()
			s.subscribers[comp] = struct{}{}
			s.mu.Unlock()
		}

		return val
	}

	write := func(newValue T) {
		s.mu.Lock()
		if equals(s.value, newValue) {
			s.mu.Unlock()
			return
		}
		s.value = newValue

		subs := make([]*computation, 0, len(s.subscribers))
		for comp := range s.subscribers {
			subs = append(subs, comp)
		}
		s.mu.Unlock()

		batchMu.Lock()
		inBatch := batchDepth > 0
		batchMu.Unlock()

		if inBatch {
			batchMu.Lock()
			for _, comp := range subs {
				pendingComputations[comp] = struct{}{}
			}
			batchMu.Unlock()
		} else {
			for _, comp := range subs {
				comp.execute()
			}
		}
	}

	return read, write
}

// SetWith updates a signal using a function that receives the previous value.
func SetWith[T any](setter Setter[T], fn SetterFunc[T], getter Accessor[T]) {
	setter(fn(getter()))
}
