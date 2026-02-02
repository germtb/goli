package goli

import "sync"

// CleanupFunc is a function called to clean up an effect.
type CleanupFunc func()

// DisposeFunc is a function that disposes an effect.
type DisposeFunc func()

// CreateEffect creates a reactive effect that runs when its dependencies change.
// Returns a dispose function to stop the effect.
//
// The effect function can optionally return a cleanup function that runs before
// each re-execution and when the effect is disposed.
//
// Example:
//
//	count, setCount := CreateSignal(0)
//
//	dispose := CreateEffect(func() CleanupFunc {
//	    fmt.Println("Count is:", count())
//	    return func() { fmt.Println("Cleaning up") }
//	})
func CreateEffect(fn func() CleanupFunc) DisposeFunc {
	var cleanup CleanupFunc
	var disposed bool
	var mu sync.Mutex

	comp := &computation{
		subscriptions: make([]subscriber, 0),
	}

	comp.execute = func() {
		mu.Lock()
		if disposed {
			mu.Unlock()
			return
		}

		// Cleanup previous run
		if cleanup != nil {
			cleanupFn := cleanup
			cleanup = nil
			mu.Unlock()
			cleanupFn()
			mu.Lock()
		}

		// Unsubscribe from old signals before re-tracking (fixes memory leak)
		comp.mu.Lock()
		for _, sub := range comp.subscriptions {
			sub.unsubscribe(comp)
		}
		comp.subscriptions = comp.subscriptions[:0]
		comp.mu.Unlock()

		mu.Unlock()

		// Run with tracking
		prevComputation := Global.getCurrentComputation()
		Global.setCurrentComputation(comp)

		newCleanup := fn()

		Global.setCurrentComputation(prevComputation)

		mu.Lock()
		cleanup = newCleanup
		mu.Unlock()
	}

	// Initial run
	comp.execute()

	// Dispose function
	dispose := func() {
		mu.Lock()
		if disposed {
			mu.Unlock()
			return
		}
		disposed = true
		cleanupFn := cleanup
		cleanup = nil

		// Unsubscribe from all signals
		comp.mu.Lock()
		for _, sub := range comp.subscriptions {
			sub.unsubscribe(comp)
		}
		comp.subscriptions = nil
		comp.mu.Unlock()

		mu.Unlock()

		if cleanupFn != nil {
			cleanupFn()
		}
	}

	// Register with current owner for automatic cleanup
	owner := Global.getCurrentOwner()
	if owner != nil {
		Global.mu.Lock()
		owner.disposables = append(owner.disposables, dispose)
		Global.mu.Unlock()
	}

	return dispose
}

// CreateEffectSimple creates an effect without cleanup.
func CreateEffectSimple(fn func()) DisposeFunc {
	return CreateEffect(func() CleanupFunc {
		fn()
		return nil
	})
}

// CreateMemo creates a memoized computation.
// Only re-computes when dependencies change.
//
// Example:
//
//	count, _ := CreateSignal(5)
//	doubled := CreateMemo(func() int {
//	    return count() * 2
//	})
//	fmt.Println(doubled()) // 10
func CreateMemo[T any](fn func() T) Accessor[T] {
	value, setValue := CreateSignal[T](*new(T))

	CreateEffect(func() CleanupFunc {
		setValue(fn())
		return nil
	})

	return value
}
