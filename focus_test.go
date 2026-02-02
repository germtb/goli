package goli

import (
	"testing"
)

// mockFocusable is a test implementation of Focusable
type mockFocusable struct {
	focused    bool
	handleFunc func(key string) bool
}

func newMockFocusable() *mockFocusable {
	return &mockFocusable{}
}

func (m *mockFocusable) Focused() bool     { return m.focused }
func (m *mockFocusable) Focus()            { Manager().RequestFocus(m) }
func (m *mockFocusable) Blur()             { Manager().RequestBlur(m) }
func (m *mockFocusable) Dispose()          { Manager().Unregister(m) }
func (m *mockFocusable) SetFocused(f bool) { m.focused = f }
func (m *mockFocusable) HandleKey(key string) bool {
	if m.handleFunc != nil {
		return m.handleFunc(key)
	}
	return false
}

func setupTest(t *testing.T) {
	t.Helper()
	Reset()
}

func TestFocusManager_RegistersAutomatically(t *testing.T) {
	setupTest(t)

	if len(Manager().GetAll()) != 0 {
		t.Error("expected no focusables initially")
	}

	f1 := newMockFocusable()
	Register(f1)
	if len(Manager().GetAll()) != 1 {
		t.Error("expected 1 focusable after register")
	}

	f2 := newMockFocusable()
	Register(f2)
	if len(Manager().GetAll()) != 2 {
		t.Error("expected 2 focusables")
	}

	f1.Dispose()
	if len(Manager().GetAll()) != 1 {
		t.Error("expected 1 focusable after dispose")
	}

	f2.Dispose()
	if len(Manager().GetAll()) != 0 {
		t.Error("expected 0 focusables after all disposed")
	}
}

func TestFocusManager_TracksFocusedElement(t *testing.T) {
	setupTest(t)

	f1 := newMockFocusable()
	f2 := newMockFocusable()
	Register(f1)
	Register(f2)

	if Manager().Current() != nil {
		t.Error("expected no focused element initially")
	}

	f1.Focus()
	if Manager().Current() != f1 {
		t.Error("expected f1 to be focused")
	}
	if !f1.focused {
		t.Error("f1 should be focused")
	}
	if f2.focused {
		t.Error("f2 should not be focused")
	}

	f2.Focus()
	if Manager().Current() != f2 {
		t.Error("expected f2 to be focused")
	}
	if f1.focused {
		t.Error("f1 should not be focused")
	}
	if !f2.focused {
		t.Error("f2 should be focused")
	}

	f2.Blur()
	if Manager().Current() != nil {
		t.Error("expected no focused element after blur")
	}
	if f2.focused {
		t.Error("f2 should not be focused after blur")
	}
}

func TestFocusManager_Next(t *testing.T) {
	setupTest(t)

	f1 := newMockFocusable()
	f2 := newMockFocusable()
	f3 := newMockFocusable()
	Register(f1)
	Register(f2)
	Register(f3)

	Manager().Next() // Focus first when none focused
	if Manager().Current() != f1 {
		t.Error("expected f1 to be focused")
	}

	Manager().Next()
	if Manager().Current() != f2 {
		t.Error("expected f2 to be focused")
	}

	Manager().Next()
	if Manager().Current() != f3 {
		t.Error("expected f3 to be focused")
	}

	Manager().Next() // Wraps around
	if Manager().Current() != f1 {
		t.Error("expected f1 to be focused after wrap")
	}
}

func TestFocusManager_Prev(t *testing.T) {
	setupTest(t)

	f1 := newMockFocusable()
	f2 := newMockFocusable()
	f3 := newMockFocusable()
	Register(f1)
	Register(f2)
	Register(f3)

	Manager().Prev() // Focus last when none focused
	if Manager().Current() != f3 {
		t.Error("expected f3 to be focused")
	}

	Manager().Prev()
	if Manager().Current() != f2 {
		t.Error("expected f2 to be focused")
	}

	Manager().Prev()
	if Manager().Current() != f1 {
		t.Error("expected f1 to be focused")
	}

	Manager().Prev() // Wraps around
	if Manager().Current() != f3 {
		t.Error("expected f3 to be focused after wrap")
	}
}

func TestFocusManager_HandleKeyTab(t *testing.T) {
	setupTest(t)

	f1 := newMockFocusable()
	f2 := newMockFocusable()
	Register(f1)
	Register(f2)

	f1.Focus()
	if Manager().Current() != f1 {
		t.Error("expected f1 to be focused")
	}

	consumed := HandleKey(Tab)
	if !consumed {
		t.Error("Tab should be consumed")
	}
	if Manager().Current() != f2 {
		t.Error("expected f2 to be focused after Tab")
	}
}

func TestFocusManager_RoutesKeysToFocused(t *testing.T) {
	setupTest(t)

	keysReceived := ""
	f := newMockFocusable()
	f.handleFunc = func(key string) bool {
		keysReceived += key
		return true
	}
	Register(f)
	f.Focus()

	HandleKey("a")
	HandleKey("b")

	if keysReceived != "ab" {
		t.Errorf("expected 'ab', got %q", keysReceived)
	}
}

func TestFocusManager_ReturnsFalseWhenNotFocused(t *testing.T) {
	setupTest(t)

	f := newMockFocusable()
	Register(f)
	// Not focused

	consumed := HandleKey("a")
	if consumed {
		t.Error("should return false when nothing focused")
	}
}

func TestFocusManager_UnregistersDisposedElements(t *testing.T) {
	setupTest(t)

	f1 := newMockFocusable()
	f2 := newMockFocusable()
	Register(f1)
	Register(f2)

	f1.Focus()
	if Manager().Current() != f1 {
		t.Error("expected f1 to be focused")
	}

	f1.Dispose()
	if Manager().Current() != nil {
		t.Error("expected no focused element after dispose")
	}
	if len(Manager().GetAll()) != 1 {
		t.Error("expected 1 focusable remaining")
	}
	if Manager().GetAll()[0] != f2 {
		t.Error("expected f2 to be remaining")
	}
}

func TestFocusManager_Set(t *testing.T) {
	setupTest(t)

	f := newMockFocusable()
	Register(f)

	Manager().Set(f)
	if Manager().Current() != f {
		t.Error("expected f to be focused")
	}

	Manager().Set(nil)
	if Manager().Current() != nil {
		t.Error("expected no focused element")
	}
}

func TestFocusManager_GlobalKeyHandler(t *testing.T) {
	setupTest(t)

	globalKey := ""
	cleanup := Manager().SetGlobalKeyHandler(func(key string) bool {
		globalKey = key
		return true
	})

	f := newMockFocusable()
	f.handleFunc = func(key string) bool {
		return false // Don't handle
	}
	Register(f)
	f.Focus()

	HandleKey("x")
	if globalKey != "x" {
		t.Errorf("expected 'x', got %q", globalKey)
	}

	cleanup()

	globalKey = ""
	HandleKey("y")
	if globalKey != "" {
		t.Error("handler should be removed after cleanup")
	}
}
