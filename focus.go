package goli

import (
	"sync"

	"github.com/germtb/goli/signals"
)

// Focusable is the interface for any focusable element (input, button, etc).
type Focusable interface {
	Focused() bool
	Focus()
	Blur()
	Dispose()
	HandleKey(key string) bool
	SetFocused(focused bool)
}

// FocusManager manages focus state for terminal UI components.
type FocusManager struct {
	mu                  sync.RWMutex
	currentFocused      signals.Accessor[Focusable]
	setCurrentFocused   signals.Setter[Focusable]
	registered          []Focusable
	globalKeyHandler func(key string) bool
}

// Global focus manager instance
var manager *FocusManager
var managerOnce sync.Once

// Manager returns the global focus manager.
func Manager() *FocusManager {
	managerOnce.Do(func() {
		current, setCurrent := signals.CreateSignal[Focusable](nil)
		manager = &FocusManager{
			currentFocused:    current,
			setCurrentFocused: setCurrent,
			registered:        make([]Focusable, 0),
		}
	})
	return manager
}

// Register adds a focusable to the manager.
func (m *FocusManager) Register(f Focusable) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.registered = append(m.registered, f)
}

// Unregister removes a focusable from the manager.
func (m *FocusManager) Unregister(f Focusable) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, registered := range m.registered {
		if registered == f {
			m.registered = append(m.registered[:i], m.registered[i+1:]...)
			break
		}
	}

	// If this was focused, clear focus
	if m.currentFocused() == f {
		m.setCurrentFocused(nil)
	}
}

// RequestFocus focuses a specific focusable.
func (m *FocusManager) RequestFocus(f Focusable) {
	current := m.currentFocused()
	if current == f {
		return
	}

	signals.BatchVoid(func() {
		if current != nil {
			current.SetFocused(false)
		}
		f.SetFocused(true)
		m.setCurrentFocused(f)
	})
}

// RequestBlur blurs a specific focusable.
func (m *FocusManager) RequestBlur(f Focusable) {
	if m.currentFocused() == f {
		signals.BatchVoid(func() {
			f.SetFocused(false)
			m.setCurrentFocused(nil)
		})
	}
}

// Current returns the currently focused element.
func (m *FocusManager) Current() Focusable {
	return m.currentFocused()
}

// Next focuses the next element in registration order.
func (m *FocusManager) Next() {
	m.mu.RLock()
	focusables := make([]Focusable, len(m.registered))
	copy(focusables, m.registered)
	m.mu.RUnlock()

	if len(focusables) == 0 {
		return
	}

	current := m.currentFocused()
	if current == nil {
		focusables[0].Focus()
		return
	}

	currentIndex := -1
	for i, f := range focusables {
		if f == current {
			currentIndex = i
			break
		}
	}

	nextIndex := (currentIndex + 1) % len(focusables)
	focusables[nextIndex].Focus()
}

// Prev focuses the previous element in registration order.
func (m *FocusManager) Prev() {
	m.mu.RLock()
	focusables := make([]Focusable, len(m.registered))
	copy(focusables, m.registered)
	m.mu.RUnlock()

	if len(focusables) == 0 {
		return
	}

	current := m.currentFocused()
	if current == nil {
		focusables[len(focusables)-1].Focus()
		return
	}

	currentIndex := -1
	for i, f := range focusables {
		if f == current {
			currentIndex = i
			break
		}
	}

	prevIndex := (currentIndex - 1 + len(focusables)) % len(focusables)
	focusables[prevIndex].Focus()
}

// HandleKey routes a keypress to the focused element.
// Handles Tab/Shift+Tab for focus navigation.
// Returns true if the key was consumed.
func (m *FocusManager) HandleKey(key string) bool {
	// Handle focus navigation
	if key == Tab {
		m.Next()
		return true
	}
	if key == ShiftTab {
		m.Prev()
		return true
	}

	// Route to focused element
	current := m.currentFocused()
	if current != nil && current.HandleKey(key) {
		return true
	}

	// Try unhandled handler
	m.mu.RLock()
	handler := m.globalKeyHandler
	m.mu.RUnlock()

	if handler != nil {
		return handler(key)
	}

	return false
}

// SetGlobalKeyHandler sets a handler for app-wide keyboard shortcuts.
// This handler is called for keys that no focused element consumes.
// Returns a cleanup function to remove the handler.
func (m *FocusManager) SetGlobalKeyHandler(handler func(key string) bool) func() {
	m.mu.Lock()
	m.globalKeyHandler = handler
	m.mu.Unlock()

	return func() {
		m.mu.Lock()
		m.globalKeyHandler = nil
		m.mu.Unlock()
	}
}

// Set manually sets the focused element. Pass nil to blur all.
func (m *FocusManager) Set(f Focusable) {
	if f == nil {
		current := m.currentFocused()
		if current != nil {
			current.Blur()
		}
	} else {
		f.Focus()
	}
}

// GetAll returns all registered focusable elements.
func (m *FocusManager) GetAll() []Focusable {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]Focusable, len(m.registered))
	copy(result, m.registered)
	return result
}

// Clear removes all registered focusables and handlers.
func (m *FocusManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	current := m.currentFocused()
	if current != nil {
		current.SetFocused(false)
	}
	m.setCurrentFocused(nil)
	m.registered = nil
	m.globalKeyHandler = nil
}

// Convenience functions that use the global manager

// Register adds a focusable to the global manager.
func Register(f Focusable) {
	Manager().Register(f)
}

// Unregister removes a focusable from the global manager.
func Unregister(f Focusable) {
	Manager().Unregister(f)
}

// RequestFocus focuses a specific focusable using the global manager.
func RequestFocus(f Focusable) {
	Manager().RequestFocus(f)
}

// RequestBlur blurs a specific focusable using the global manager.
func RequestBlur(f Focusable) {
	Manager().RequestBlur(f)
}

// HandleKey routes a keypress using the global manager.
func HandleKey(key string) bool {
	return Manager().HandleKey(key)
}
