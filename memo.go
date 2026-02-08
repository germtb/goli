package goli

import (
	"sync"
	"sync/atomic"

	"github.com/germtb/gox"
)

// memoGeneration tracks render cycles for cache invalidation
var memoGeneration atomic.Int64

type memoEntry[P any] struct {
	props      P
	result     gox.VNode
	generation int64
}

// Keyed is the interface that memoized component props must implement.
// It provides the cache key for identifying component instances across renders.
// K must be comparable (int, string, etc.)
type Keyed[K comparable] interface {
	GetKey() K
}

// ShallowEquals returns a == b. Use this with Memo for comparable prop types.
//
// Usage:
//
//	var Cell = goli.Memo(renderCell, goli.ShallowEquals[CellProps])
func ShallowEquals[P comparable](a, b P) bool {
	return a == b
}

// Memo creates a memoized component that skips re-rendering when props haven't changed.
//
// Props must implement the Keyed[K] interface to provide a cache key.
// K is the key type (typically int or string), inferred from GetKey().
//
// Parameters:
//   - render: the component function to memoize
//   - equal: equality function to compare props (use goli.ShallowEquals for comparable types)
//
// Usage:
//
//	type CellProps struct {
//	    Key   int  // use int for zero allocation!
//	    Index int
//	}
//
//	func (p CellProps) GetKey() int { return p.Key }
//
//	var Cell = goli.Memo(
//	    func(props CellProps, children ...gox.VNode) gox.VNode {
//	        return <text>{props.Value}</text>
//	    },
//	    goli.ShallowEquals[CellProps],
//	)
func Memo[K comparable, P Keyed[K]](
	render func(P, ...gox.VNode) gox.VNode,
	equal func(a, b P) bool,
) func(P, ...gox.VNode) gox.VNode {
	var cache sync.Map

	return func(props P, children ...gox.VNode) gox.VNode {
		gen := memoGeneration.Load()
		key := props.GetKey()

		if entry, ok := cache.Load(key); ok {
			e := entry.(*memoEntry[P])
			if e.generation >= gen-1 && equal(e.props, props) {
				e.generation = gen
				return e.result
			}
		}

		result := render(props, children...)

		cache.Store(key, &memoEntry[P]{
			props:      props,
			result:     result,
			generation: gen,
		})

		return result
	}
}

// BeginRender increments the generation counter. Call at start of each render.
func BeginRender() {
	memoGeneration.Add(1)
}

// MemoStats returns cache statistics (for debugging/benchmarking).
type MemoStats struct {
	Generation int64
}

// GetMemoStats returns current memo statistics.
func GetMemoStats() MemoStats {
	return MemoStats{
		Generation: memoGeneration.Load(),
	}
}
