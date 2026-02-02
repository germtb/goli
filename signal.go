// Package goli provides fine-grained reactive primitives.
//
// Key principles:
// - Components run ONCE (setup phase)
// - Signals created inside components are local to that instance
// - Fine-grained reactivity: only re-run what depends on changed signals
// - No rules of hooks - signals are just values
package goli

import "sync"

// Accessor is a function that reads a signal value.
type Accessor[T any] func() T

// Setter is a function that updates a signal value.
type Setter[T any] func(T)

// SetterFunc updates based on previous value.
type SetterFunc[T any] func(prev T) T

// signalValue is the internal signal implementation.
type signalValue[T any] struct {
	value       T
	subscribers map[*computation]struct{}
	mu          sync.RWMutex
}

// unsubscribe removes a computation from this signal's subscribers.
// Implements the subscriber interface.
func (s *signalValue[T]) unsubscribe(comp *computation) {
	s.mu.Lock()
	delete(s.subscribers, comp)
	s.mu.Unlock()
}

// CreateSignal creates a reactive signal.
//
// Example:
//
//	count, setCount := CreateSignal(0)
//	fmt.Println(count()) // 0
//	setCount(1)
//	fmt.Println(count()) // 1
func CreateSignal[T any](initialValue T) (Accessor[T], Setter[T]) {
	return createSignalInternal(Global, initialValue)
}

// createSignalInternal creates a signal using the given runtime.
// This is used internally to avoid circular initialization.
func createSignalInternal[T any](rt *Runtime, initialValue T) (Accessor[T], Setter[T]) {
	s := &signalValue[T]{
		value:       initialValue,
		subscribers: make(map[*computation]struct{}),
	}

	read := func() T {
		s.mu.RLock()
		val := s.value
		s.mu.RUnlock()

		// Track this signal as a dependency of current computation
		comp := rt.getCurrentComputation()
		if comp != nil {
			s.mu.Lock()
			s.subscribers[comp] = struct{}{}
			s.mu.Unlock()

			// Store subscription for cleanup (fixes memory leak)
			comp.mu.Lock()
			comp.subscriptions = append(comp.subscriptions, s)
			comp.mu.Unlock()
		}

		return val
	}

	write := func(newValue T) {
		s.mu.Lock()
		s.value = newValue

		// Get subscribers to notify
		subs := make([]*computation, 0, len(s.subscribers))
		for comp := range s.subscribers {
			subs = append(subs, comp)
		}
		s.mu.Unlock()

		// Notify subscribers
		inBatch := rt.getBatchDepth() > 0

		if inBatch {
			for _, comp := range subs {
				rt.addPendingComputation(comp)
			}
		} else {
			for _, comp := range subs {
				comp.execute()
			}
		}
	}

	return read, write
}

// CreateSignalWithEquals creates a signal with a custom equality function.
// If the new value equals the old value according to the equality function,
// subscribers are not notified.
func CreateSignalWithEquals[T any](initialValue T, equals func(a, b T) bool) (Accessor[T], Setter[T]) {
	s := &signalValue[T]{
		value:       initialValue,
		subscribers: make(map[*computation]struct{}),
	}

	read := func() T {
		s.mu.RLock()
		val := s.value
		s.mu.RUnlock()

		comp := Global.getCurrentComputation()
		if comp != nil {
			s.mu.Lock()
			s.subscribers[comp] = struct{}{}
			s.mu.Unlock()

			// Store subscription for cleanup (fixes memory leak)
			comp.mu.Lock()
			comp.subscriptions = append(comp.subscriptions, s)
			comp.mu.Unlock()
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

		inBatch := Global.getBatchDepth() > 0

		if inBatch {
			for _, comp := range subs {
				Global.addPendingComputation(comp)
			}
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
