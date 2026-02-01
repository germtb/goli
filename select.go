// Package goli provides a select primitive for list selection.
package goli

import (
	"sync"

	"github.com/germtb/goli/signals"
)

// SelectOptions configures select creation.
type SelectOptions[T comparable] struct {
	// InitialValue is the starting selection (used to set initial index when option with this value is registered).
	InitialValue T
	// OnChange is called when selection changes.
	OnChange func(value T)
	// OnKeypress is a custom key handler (called before default handling).
	OnKeypress func(key string) bool
	// DisableFocus disables focus management registration (default: false, meaning focusable by default).
	DisableFocus bool
}

// Select represents a list selection component.
// The select tracks the selected index; option values come from <option> children.
type Select[T comparable] struct {
	mu sync.RWMutex

	selectedIndex signals.Accessor[int]
	setIndex      signals.Setter[int]
	focused       signals.Accessor[bool]
	setFocused    signals.Setter[bool]

	// optionValues is populated during layout from <option> children - not a signal
	optionValues    map[int]T
	optionCount     int
	initialValue    T
	hasInitialValue bool
	initialApplied  bool
	initialIndex    int // The index to use when initial value is found

	onChange       func(value T)
	onKeypress     func(key string) bool
	shouldRegister bool
	registered     bool
}

// NewSelect creates a new select primitive.
func NewSelect[T comparable](opts SelectOptions[T]) *Select[T] {
	selectedIndex, setIndex := signals.CreateSignal(0)
	focused, setFocused := signals.CreateSignal(false)

	shouldRegister := true
	if opts.DisableFocus {
		shouldRegister = false
	}

	var zero T
	hasInitial := opts.InitialValue != zero

	s := &Select[T]{
		selectedIndex:   selectedIndex,
		setIndex:        setIndex,
		focused:         focused,
		setFocused:      setFocused,
		optionValues:    make(map[int]T),
		initialValue:    opts.InitialValue,
		hasInitialValue: hasInitial,
		onChange:        opts.OnChange,
		onKeypress:      opts.OnKeypress,
		shouldRegister:  shouldRegister,
	}

	if shouldRegister {
		Register(s)
		s.registered = true
	}

	return s
}

// Value returns the currently selected value.
func (s *Select[T]) Value() T {
	idx := s.SelectedIndex()
	s.mu.RLock()
	defer s.mu.RUnlock()
	if val, ok := s.optionValues[idx]; ok {
		return val
	}
	// If no options registered yet but we have an initial value, return it
	if s.hasInitialValue {
		return s.initialValue
	}
	var zero T
	return zero
}

// SelectedIndex returns the currently selected index.
func (s *Select[T]) SelectedIndex() int {
	idx := s.selectedIndex()
	// If the signal is at 0 and we have an initial value, use the initial index
	s.mu.RLock()
	if idx == 0 && s.initialApplied && s.initialIndex > 0 {
		idx = s.initialIndex
	}
	s.mu.RUnlock()
	return idx
}

// IsSelectedIndex returns true if the given index is selected.
func (s *Select[T]) IsSelectedIndex(index int) bool {
	return s.SelectedIndex() == index
}

// Focused returns whether this select is focused.
func (s *Select[T]) Focused() bool {
	return s.focused()
}

// RegisterOption registers an option value at an index (called during layout).
// This does NOT trigger re-renders.
func (s *Select[T]) RegisterOption(index int, value T) {
	s.mu.Lock()
	s.optionValues[index] = value

	// Apply initial value if this matches - store the index for later use
	if !s.initialApplied && s.hasInitialValue && value == s.initialValue {
		s.initialApplied = true
		s.initialIndex = index
	}
	s.mu.Unlock()
}

// RegisterOptionAny registers an option value at an index (type-unsafe version for intrinsic use).
func (s *Select[T]) RegisterOptionAny(index int, value any) {
	if v, ok := value.(T); ok {
		s.RegisterOption(index, v)
	}
}

// SetOptionCount sets the option count (called during layout).
// This does NOT trigger re-renders.
func (s *Select[T]) SetOptionCount(count int) {
	s.mu.Lock()
	s.optionCount = count
	s.mu.Unlock()
}

// ClearOptions clears registered options (called during layout).
func (s *Select[T]) ClearOptions() {
	s.mu.Lock()
	s.optionValues = make(map[int]T)
	s.mu.Unlock()
}

// SetIndex sets selection by index.
func (s *Select[T]) SetIndex(index int) {
	if index < 0 {
		index = 0
	}
	s.setIndex(index)
}

// Next selects the next option.
func (s *Select[T]) Next() {
	current := s.selectedIndex()
	s.setIndex(current + 1)
}

// Prev selects the previous option.
func (s *Select[T]) Prev() {
	current := s.selectedIndex()
	if current > 0 {
		s.setIndex(current - 1)
	}
}

// Focus gives focus to this select.
func (s *Select[T]) Focus() {
	RequestFocus(s)
}

// Blur removes focus from this select.
func (s *Select[T]) Blur() {
	RequestBlur(s)
}

// SetFocused sets the focused state (called by focus manager).
func (s *Select[T]) SetFocused(f bool) {
	s.setFocused(f)
}

// Dispose unregisters from the focus manager.
func (s *Select[T]) Dispose() {
	if s.registered {
		Unregister(s)
		s.registered = false
	}
}

// HandleKey processes a key press.
func (s *Select[T]) HandleKey(key string) bool {
	if !s.focused() {
		return false
	}

	// Custom handler first
	if s.onKeypress != nil {
		if s.onKeypress(key) {
			return true
		}
	}

	// Default navigation
	switch key {
	case Up, CtrlP, CtrlK, "k":
		s.Prev()
		return true
	case Down, CtrlN, CtrlJ, "j":
		s.Next()
		return true
	case Home, HomeAlt, "g":
		s.SetIndex(0)
		return true
	case Enter, Space:
		return true
	}

	return false
}
