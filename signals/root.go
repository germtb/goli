package signals

import "sync"

// Owner tracks disposables for cleanup.
type Owner struct {
	disposables []func()
}

var (
	currentOwner *Owner
	ownerMu      sync.Mutex
)

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

	ownerMu.Lock()
	prevOwner := currentOwner
	currentOwner = owner
	ownerMu.Unlock()

	defer func() {
		ownerMu.Lock()
		currentOwner = prevOwner
		ownerMu.Unlock()
	}()

	dispose := func() {
		ownerMu.Lock()
		disposables := owner.disposables
		owner.disposables = nil
		ownerMu.Unlock()

		for _, d := range disposables {
			d()
		}
	}

	return fn(dispose)
}

// OnCleanup registers a cleanup function to run when the current owner is disposed.
func OnCleanup(fn func()) {
	ownerMu.Lock()
	defer ownerMu.Unlock()

	if currentOwner != nil {
		currentOwner.disposables = append(currentOwner.disposables, fn)
	}
}

// GetOwner returns the current owner, if any.
func GetOwner() *Owner {
	ownerMu.Lock()
	defer ownerMu.Unlock()
	return currentOwner
}

// RunWithOwner runs a function with a specific owner.
func RunWithOwner[T any](owner *Owner, fn func() T) T {
	ownerMu.Lock()
	prevOwner := currentOwner
	currentOwner = owner
	ownerMu.Unlock()

	defer func() {
		ownerMu.Lock()
		currentOwner = prevOwner
		ownerMu.Unlock()
	}()

	return fn()
}
