package goli

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
	Global.incrementBatchDepth()

	defer func() {
		if Global.decrementBatchDepth() {
			Global.flushPending()
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
	prevComputation := Global.getCurrentComputation()
	Global.setCurrentComputation(nil)

	defer func() {
		Global.setCurrentComputation(prevComputation)
	}()

	return fn()
}

// IsTracking returns true if we're currently inside a reactive tracking context.
func IsTracking() bool {
	return Global.getCurrentComputation() != nil
}
