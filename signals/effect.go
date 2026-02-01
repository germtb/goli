package signals

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
		dependencies: make(map[*signal[any]]struct{}),
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

		// Clear old dependencies (simplified - full impl would unsubscribe)
		comp.dependencies = make(map[*signal[any]]struct{})
		mu.Unlock()

		// Run with tracking
		currentComputationMu.Lock()
		prevComputation := currentComputation
		currentComputation = comp
		currentComputationMu.Unlock()

		newCleanup := fn()

		currentComputationMu.Lock()
		currentComputation = prevComputation
		currentComputationMu.Unlock()

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
		mu.Unlock()

		if cleanupFn != nil {
			cleanupFn()
		}
	}

	// Register with current owner for automatic cleanup
	ownerMu.Lock()
	if currentOwner != nil {
		currentOwner.disposables = append(currentOwner.disposables, dispose)
	}
	ownerMu.Unlock()

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
