package goli

// CreateRoot creates a reactive root. All reactive primitives created inside
// will be cleaned up when the root is disposed.
//
// Example:
//
//	result := CreateRoot(func(dispose DisposeFunc) string {
//	    count, setCount := CreateSignal(0)
//	    CreateEffect(func() CleanupFunc {
//	        fmt.Println("Count:", count())
//	        return nil
//	    })
//	    setCount(1)
//	    return "done"
//	})
func CreateRoot[T any](fn func(dispose DisposeFunc) T) T {
	owner := &Owner{disposables: make([]func(), 0)}

	prevOwner := Global.getCurrentOwner()
	Global.setCurrentOwner(owner)

	defer func() {
		Global.setCurrentOwner(prevOwner)
	}()

	dispose := func() {
		Global.mu.Lock()
		disposables := owner.disposables
		owner.disposables = nil
		Global.mu.Unlock()

		for _, d := range disposables {
			d()
		}
	}

	return fn(dispose)
}

// OnCleanup registers a cleanup function to run when the current owner is disposed.
func OnCleanup(fn func()) {
	Global.mu.Lock()
	defer Global.mu.Unlock()

	owner := Global.currentOwner
	if owner != nil {
		owner.disposables = append(owner.disposables, fn)
	}
}

// GetOwner returns the current owner, if any.
func GetOwner() *Owner {
	return Global.getCurrentOwner()
}

// RunWithOwner runs a function with a specific owner.
func RunWithOwner[T any](owner *Owner, fn func() T) T {
	prevOwner := Global.getCurrentOwner()
	Global.setCurrentOwner(owner)

	defer func() {
		Global.setCurrentOwner(prevOwner)
	}()

	return fn()
}
