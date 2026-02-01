package signals

// Batch batches multiple signal updates into a single update cycle.
// All effects are deferred until the batch completes.
//
// Example:
//
//	count, setCount := CreateSignal(0)
//	name, setName := CreateSignal("")
//
//	Batch(func() {
//	    setCount(1)
//	    setName("test")
//	    // Effects run only once after both updates
//	})
func Batch[T any](fn func() T) T {
	batchMu.Lock()
	batchDepth++
	batchMu.Unlock()

	defer func() {
		batchMu.Lock()
		batchDepth--
		shouldFlush := batchDepth == 0
		batchMu.Unlock()

		if shouldFlush {
			flushPending()
		}
	}()

	return fn()
}

// BatchVoid is a convenience wrapper for Batch when there's no return value.
func BatchVoid(fn func()) {
	Batch(func() struct{} {
		fn()
		return struct{}{}
	})
}

func flushPending() {
	batchMu.Lock()
	toRun := make([]*computation, 0, len(pendingComputations))
	for comp := range pendingComputations {
		toRun = append(toRun, comp)
	}
	pendingComputations = make(map[*computation]struct{})
	batchMu.Unlock()

	for _, comp := range toRun {
		comp.execute()
	}
}

// Untrack reads signals without tracking them as dependencies.
//
// Example:
//
//	count, _ := CreateSignal(0)
//	other, _ := CreateSignal(0)
//
//	CreateEffect(func() CleanupFunc {
//	    // This effect only depends on 'count', not 'other'
//	    fmt.Println(count(), Untrack(func() int { return other() }))
//	    return nil
//	})
func Untrack[T any](fn func() T) T {
	currentComputationMu.Lock()
	prevComputation := currentComputation
	currentComputation = nil
	currentComputationMu.Unlock()

	defer func() {
		currentComputationMu.Lock()
		currentComputation = prevComputation
		currentComputationMu.Unlock()
	}()

	return fn()
}

// IsTracking returns true if we're currently inside a reactive tracking context.
func IsTracking() bool {
	currentComputationMu.Lock()
	defer currentComputationMu.Unlock()
	return currentComputation != nil
}
