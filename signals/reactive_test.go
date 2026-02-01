package signals

import (
	"testing"
)

func TestCreateSignal_ReturnsAccessorAndSetter(t *testing.T) {
	count, setCount := CreateSignal(0)

	if count == nil {
		t.Error("accessor should not be nil")
	}
	if setCount == nil {
		t.Error("setter should not be nil")
	}
}

func TestCreateSignal_AccessorReturnsCurrentValue(t *testing.T) {
	count, _ := CreateSignal(42)
	if count() != 42 {
		t.Errorf("expected 42, got %d", count())
	}
}

func TestCreateSignal_SetterUpdatesValue(t *testing.T) {
	count, setCount := CreateSignal(0)
	setCount(5)
	if count() != 5 {
		t.Errorf("expected 5, got %d", count())
	}
}

func TestCreateSignal_SetterAcceptsUpdateFunction(t *testing.T) {
	count, setCount := CreateSignal(10)
	// Use SetWith to update based on previous value
	SetWith(setCount, func(prev int) int { return prev + 5 }, count)
	if count() != 15 {
		t.Errorf("expected 15, got %d", count())
	}
}

func TestCreateSignal_DoesNotTriggerForSameValue(t *testing.T) {
	// Use CreateSignalWithEquals for equality-based deduplication
	count, setCount := CreateSignalWithEquals(5, func(a, b int) bool { return a == b })
	effectRuns := 0

	CreateRoot(func(dispose DisposeFunc) func() {
		CreateEffect(func() CleanupFunc {
			_ = count()
			effectRuns++
			return nil
		})
		return dispose
	})

	if effectRuns != 1 {
		t.Errorf("expected 1 effect run, got %d", effectRuns)
	}

	setCount(5) // Same value
	if effectRuns != 1 {
		t.Errorf("expected still 1 effect run, got %d", effectRuns)
	}
}

func TestCreateSignal_WorksWithObjects(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}
	state, setState := CreateSignal(Person{Name: "Alice", Age: 30})

	if state().Name != "Alice" {
		t.Errorf("expected Alice, got %s", state().Name)
	}

	setState(Person{Name: "Bob", Age: 25})
	if state().Name != "Bob" {
		t.Errorf("expected Bob, got %s", state().Name)
	}
}

func TestCreateSignal_WorksWithSlices(t *testing.T) {
	items, setItems := CreateSignal([]int{1, 2, 3})

	got := items()
	if len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Errorf("expected [1,2,3], got %v", got)
	}

	// Use SetWith to update based on previous value
	SetWith(setItems, func(arr []int) []int {
		return append(arr, 4)
	}, items)
	got = items()
	if len(got) != 4 || got[3] != 4 {
		t.Errorf("expected [1,2,3,4], got %v", got)
	}
}

func TestCreateEffect_RunsImmediately(t *testing.T) {
	ran := false
	CreateRoot(func(dispose DisposeFunc) func() {
		CreateEffect(func() CleanupFunc {
			ran = true
			return nil
		})
		return dispose
	})
	if !ran {
		t.Error("effect should run immediately")
	}
}

func TestCreateEffect_RerunsOnDependencyChange(t *testing.T) {
	count, setCount := CreateSignal(0)
	var values []int

	CreateRoot(func(dispose DisposeFunc) func() {
		CreateEffect(func() CleanupFunc {
			values = append(values, count())
			return nil
		})
		return dispose
	})

	setCount(1)
	setCount(2)

	expected := []int{0, 1, 2}
	if len(values) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, values)
	}
	for i, v := range expected {
		if values[i] != v {
			t.Errorf("at index %d, expected %d, got %d", i, v, values[i])
		}
	}
}

func TestCreateEffect_TracksMultipleSignals(t *testing.T) {
	a, setA := CreateSignal(1)
	b, setB := CreateSignal(2)
	var sums []int

	CreateRoot(func(dispose DisposeFunc) func() {
		CreateEffect(func() CleanupFunc {
			sums = append(sums, a()+b())
			return nil
		})
		return dispose
	})

	if len(sums) != 1 || sums[0] != 3 {
		t.Errorf("expected [3], got %v", sums)
	}

	setA(10)
	if len(sums) != 2 || sums[1] != 12 {
		t.Errorf("expected [3, 12], got %v", sums)
	}

	setB(20)
	if len(sums) != 3 || sums[2] != 30 {
		t.Errorf("expected [3, 12, 30], got %v", sums)
	}
}

func TestCreateEffect_RunsCleanupBeforeRerun(t *testing.T) {
	count, setCount := CreateSignal(0)
	cleanups := 0

	CreateRoot(func(dispose DisposeFunc) func() {
		CreateEffect(func() CleanupFunc {
			_ = count()
			return func() {
				cleanups++
			}
		})
		return dispose
	})

	if cleanups != 0 {
		t.Errorf("expected 0 cleanups, got %d", cleanups)
	}
	setCount(1)
	if cleanups != 1 {
		t.Errorf("expected 1 cleanup, got %d", cleanups)
	}
	setCount(2)
	if cleanups != 2 {
		t.Errorf("expected 2 cleanups, got %d", cleanups)
	}
}

func TestCreateEffect_CleanupRunsOnDispose(t *testing.T) {
	count, _ := CreateSignal(0)
	cleanups := 0

	CreateRoot(func(dispose DisposeFunc) func() {
		CreateEffect(func() CleanupFunc {
			_ = count()
			return func() {
				cleanups++
			}
		})

		if cleanups != 0 {
			t.Errorf("expected 0 cleanups before dispose, got %d", cleanups)
		}
		dispose()
		if cleanups != 1 {
			t.Errorf("expected 1 cleanup after dispose, got %d", cleanups)
		}
		return dispose
	})
}

func TestCreateMemo_ComputesDerivedValue(t *testing.T) {
	count, _ := CreateSignal(5)
	computeCount := 0

	doubled := CreateMemo(func() int {
		computeCount++
		return count() * 2
	})

	if doubled() != 10 {
		t.Errorf("expected 10, got %d", doubled())
	}
	if computeCount != 1 {
		t.Errorf("expected 1 computation, got %d", computeCount)
	}
}

func TestCreateMemo_UpdatesOnDependencyChange(t *testing.T) {
	count, setCount := CreateSignal(5)
	doubled := CreateMemo(func() int {
		return count() * 2
	})

	if doubled() != 10 {
		t.Errorf("expected 10, got %d", doubled())
	}
	setCount(10)
	if doubled() != 20 {
		t.Errorf("expected 20, got %d", doubled())
	}
}

func TestCreateMemo_ChainsMemos(t *testing.T) {
	count, setCount := CreateSignal(2)
	doubled := CreateMemo(func() int {
		return count() * 2
	})
	quadrupled := CreateMemo(func() int {
		return doubled() * 2
	})

	if quadrupled() != 8 {
		t.Errorf("expected 8, got %d", quadrupled())
	}
	setCount(5)
	if quadrupled() != 20 {
		t.Errorf("expected 20, got %d", quadrupled())
	}
}

func TestCreateRoot_ReturnsResult(t *testing.T) {
	result := CreateRoot(func(dispose DisposeFunc) int {
		return 42
	})
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
}

func TestCreateRoot_DisposeCleansUpEffects(t *testing.T) {
	count, setCount := CreateSignal(0)
	var values []int

	CreateRoot(func(dispose DisposeFunc) func() {
		CreateEffect(func() CleanupFunc {
			values = append(values, count())
			return nil
		})

		setCount(1)
		if len(values) != 2 {
			t.Errorf("expected [0,1], got %v", values)
		}

		dispose()
		setCount(2)
		// No new value should be added after dispose
		if len(values) != 2 {
			t.Errorf("expected still [0,1] after dispose, got %v", values)
		}
		return dispose
	})
}

func TestOnCleanup_RunsOnDispose(t *testing.T) {
	cleaned := false

	CreateRoot(func(dispose DisposeFunc) func() {
		OnCleanup(func() {
			cleaned = true
		})

		if cleaned {
			t.Error("should not be cleaned before dispose")
		}
		dispose()
		if !cleaned {
			t.Error("should be cleaned after dispose")
		}
		return dispose
	})
}

func TestBatch_BatchesMultipleUpdates(t *testing.T) {
	a, setA := CreateSignal(0)
	b, setB := CreateSignal(0)
	effectRuns := 0

	CreateRoot(func(dispose DisposeFunc) func() {
		CreateEffect(func() CleanupFunc {
			_ = a()
			_ = b()
			effectRuns++
			return nil
		})
		return dispose
	})

	if effectRuns != 1 {
		t.Errorf("expected 1 initial run, got %d", effectRuns)
	}

	BatchVoid(func() {
		setA(1)
		setB(2)
	})

	if effectRuns != 2 {
		t.Errorf("expected 2 runs (initial + 1 batch), got %d", effectRuns)
	}
}

func TestBatch_HandlesNestedBatches(t *testing.T) {
	count, setCount := CreateSignal(0)
	effectRuns := 0

	CreateRoot(func(dispose DisposeFunc) func() {
		CreateEffect(func() CleanupFunc {
			_ = count()
			effectRuns++
			return nil
		})
		return dispose
	})

	BatchVoid(func() {
		setCount(1)
		BatchVoid(func() {
			setCount(2)
		})
		setCount(3)
	})

	if effectRuns != 2 {
		t.Errorf("expected 2 runs (initial + 1 batch), got %d", effectRuns)
	}
	if count() != 3 {
		t.Errorf("expected count=3, got %d", count())
	}
}

func TestBatch_ReturnsResult(t *testing.T) {
	result := Batch(func() int {
		return 42
	})
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
}

func TestUntrack_PreventsTracking(t *testing.T) {
	count, setCount := CreateSignal(0)
	effectRuns := 0

	CreateRoot(func(dispose DisposeFunc) func() {
		CreateEffect(func() CleanupFunc {
			Untrack(func() int { return count() })
			effectRuns++
			return nil
		})
		return dispose
	})

	if effectRuns != 1 {
		t.Errorf("expected 1 run, got %d", effectRuns)
	}
	setCount(1)
	if effectRuns != 1 {
		t.Errorf("expected still 1 run (untracked), got %d", effectRuns)
	}
}

func TestUntrack_ReturnsValue(t *testing.T) {
	count, _ := CreateSignal(42)
	value := Untrack(func() int { return count() })
	if value != 42 {
		t.Errorf("expected 42, got %d", value)
	}
}

func TestSolidPatterns_ComponentLocalState(t *testing.T) {
	type Counter struct {
		Count     Accessor[int]
		Increment func()
		Decrement func()
	}

	createCounter := func(initial int) Counter {
		count, setCount := CreateSignal(initial)
		return Counter{
			Count:     count,
			Increment: func() { SetWith(setCount, func(c int) int { return c + 1 }, count) },
			Decrement: func() { SetWith(setCount, func(c int) int { return c - 1 }, count) },
		}
	}

	counter1 := createCounter(0)
	counter2 := createCounter(100)

	counter1.Increment()
	counter1.Increment()
	counter2.Decrement()

	if counter1.Count() != 2 {
		t.Errorf("expected counter1 = 2, got %d", counter1.Count())
	}
	if counter2.Count() != 99 {
		t.Errorf("expected counter2 = 99, got %d", counter2.Count())
	}
}

func TestSolidPatterns_DerivedStateWithMemo(t *testing.T) {
	firstName, setFirstName := CreateSignal("John")
	lastName, setLastName := CreateSignal("Doe")

	fullName := CreateMemo(func() string {
		return firstName() + " " + lastName()
	})

	if fullName() != "John Doe" {
		t.Errorf("expected 'John Doe', got %q", fullName())
	}

	setFirstName("Jane")
	if fullName() != "Jane Doe" {
		t.Errorf("expected 'Jane Doe', got %q", fullName())
	}

	setLastName("Smith")
	if fullName() != "Jane Smith" {
		t.Errorf("expected 'Jane Smith', got %q", fullName())
	}
}

func TestSolidPatterns_ConditionalEffects(t *testing.T) {
	show, setShow := CreateSignal(true)
	count, setCount := CreateSignal(0)
	var values []int

	CreateRoot(func(dispose DisposeFunc) func() {
		CreateEffect(func() CleanupFunc {
			if show() {
				values = append(values, count())
			}
			return nil
		})
		return dispose
	})

	if len(values) != 1 || values[0] != 0 {
		t.Errorf("expected [0], got %v", values)
	}

	setCount(1)
	if len(values) != 2 || values[1] != 1 {
		t.Errorf("expected [0, 1], got %v", values)
	}

	setShow(false)
	setCount(2)
	// Effect still tracks show(), so it runs but doesn't push
	if len(values) != 2 {
		t.Errorf("expected still [0, 1], got %v", values)
	}
}
