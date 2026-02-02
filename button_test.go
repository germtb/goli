package goli

import (
	"testing"
)

func TestButtonCreation(t *testing.T) {
	// Clear focus manager before test
	Manager().Clear()

	btn := NewButton(ButtonOptions{})
	defer btn.Dispose()

	if btn.Focused() {
		t.Error("Button should not be focused initially")
	}
}

func TestButtonOnClick(t *testing.T) {
	// Clear focus manager before test
	Manager().Clear()

	clicked := false
	btn := NewButton(ButtonOptions{
		OnClick: func() {
			clicked = true
		},
	})
	defer btn.Dispose()

	btn.Focus()
	if !btn.Focused() {
		t.Error("Button should be focused after Focus()")
	}

	// Simulate Enter key
	consumed := btn.HandleKey(Enter)
	if !consumed {
		t.Error("Button should consume Enter key when focused")
	}
	if !clicked {
		t.Error("OnClick should be called on Enter")
	}
}

func TestButtonSpaceKey(t *testing.T) {
	// Clear focus manager before test
	Manager().Clear()

	clicked := false
	btn := NewButton(ButtonOptions{
		OnClick: func() {
			clicked = true
		},
	})
	defer btn.Dispose()

	btn.Focus()

	// Simulate Space key
	consumed := btn.HandleKey(Space)
	if !consumed {
		t.Error("Button should consume Space key when focused")
	}
	if !clicked {
		t.Error("OnClick should be called on Space")
	}
}

func TestButtonIgnoresKeysWhenNotFocused(t *testing.T) {
	// Clear focus manager before test
	Manager().Clear()

	clicked := false
	btn := NewButton(ButtonOptions{
		OnClick: func() {
			clicked = true
		},
	})
	defer btn.Dispose()

	// Don't focus the button
	consumed := btn.HandleKey(Enter)
	if consumed {
		t.Error("Button should not consume keys when not focused")
	}
	if clicked {
		t.Error("OnClick should not be called when not focused")
	}
}

func TestButtonCustomKeyHandler(t *testing.T) {
	// Clear focus manager before test
	Manager().Clear()

	customHandled := false
	btn := NewButton(ButtonOptions{
		OnKeypress: func(key string) bool {
			if key == "x" {
				customHandled = true
				return true
			}
			return false
		},
	})
	defer btn.Dispose()

	btn.Focus()

	// Custom handler should be called first
	consumed := btn.HandleKey("x")
	if !consumed {
		t.Error("Custom key handler should consume key")
	}
	if !customHandled {
		t.Error("Custom key handler should be called")
	}
}

func TestButtonDisableFocus(t *testing.T) {
	// Clear focus manager before test
	Manager().Clear()

	btn := NewButton(ButtonOptions{
		DisableFocus: true,
	})
	defer btn.Dispose()

	// Button should not be in the focus manager's registered list
	all := Manager().GetAll()
	for _, f := range all {
		if f == btn {
			t.Error("Button with DisableFocus should not be registered")
		}
	}
}

func TestButtonClick(t *testing.T) {
	// Clear focus manager before test
	Manager().Clear()

	clicked := false
	btn := NewButton(ButtonOptions{
		OnClick: func() {
			clicked = true
		},
	})
	defer btn.Dispose()

	// Programmatic click
	btn.Click()
	if !clicked {
		t.Error("Click() should trigger OnClick")
	}
}

func TestButtonFocusNavigation(t *testing.T) {
	// Clear focus manager before test
	Manager().Clear()

	btn1 := NewButton(ButtonOptions{})
	btn2 := NewButton(ButtonOptions{})
	defer btn1.Dispose()
	defer btn2.Dispose()

	// Focus first button
	btn1.Focus()
	if !btn1.Focused() {
		t.Error("btn1 should be focused")
	}
	if btn2.Focused() {
		t.Error("btn2 should not be focused")
	}

	// Tab to next
	Manager().HandleKey(Tab)
	if btn1.Focused() {
		t.Error("btn1 should not be focused after Tab")
	}
	if !btn2.Focused() {
		t.Error("btn2 should be focused after Tab")
	}

	// Shift+Tab back
	Manager().HandleKey(ShiftTab)
	if !btn1.Focused() {
		t.Error("btn1 should be focused after Shift+Tab")
	}
	if btn2.Focused() {
		t.Error("btn2 should not be focused after Shift+Tab")
	}
}

func TestButtonBlur(t *testing.T) {
	// Clear focus manager before test
	Manager().Clear()

	btn := NewButton(ButtonOptions{})
	defer btn.Dispose()

	btn.Focus()
	if !btn.Focused() {
		t.Error("Button should be focused after Focus()")
	}

	btn.Blur()
	if btn.Focused() {
		t.Error("Button should not be focused after Blur()")
	}
}
