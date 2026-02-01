package main

import (
	"strings"
	"testing"

	"github.com/germtb/goli"
	"github.com/germtb/gox"
)

// resetState resets the editor state for testing
func resetState() {
	setEditorState(EditorState{
		Lines:    []string{"Hello", "World"},
		CursorX:  0,
		CursorY:  0,
		Mode:     NormalMode,
		Command:  "",
		Message:  "",
		Filename: "test.txt",
	})
	// Clear any previous focus manager state
	goli.Manager().Clear()
}

func TestVimEditor_InitialRender(t *testing.T) {
	resetState()

	var output strings.Builder
	application := goli.Render(func() gox.VNode {
		return Editor(EditorProps{
			Width:  40,
			Height: 10,
		})
	}, goli.Options{Width: 40, Height: 10, Output: &output, DisableThrottle: true})

	buf := application.Renderer().CurrentBuffer()
	debugStr := buf.ToDebugString()
	t.Logf("Initial render:\n%s", debugStr)

	// Verify line numbers are present
	if !strings.Contains(debugStr, "1 ") {
		t.Error("Expected line number 1")
	}
	if !strings.Contains(debugStr, "Hello") {
		t.Error("Expected 'Hello' text")
	}
	if !strings.Contains(debugStr, "NORMAL") {
		t.Error("Expected 'NORMAL' mode indicator")
	}

	application.Dispose()
}

func TestVimEditor_InsertMode(t *testing.T) {
	resetState()
	setEditorState(EditorState{
		Lines:    []string{"Hello"},
		CursorX:  0,
		CursorY:  0,
		Mode:     NormalMode,
		Command:  "",
		Message:  "",
		Filename: "test.txt",
	})

	var output strings.Builder
	var application *goli.App

	application = goli.Render(func() gox.VNode {
		return Editor(EditorProps{
			Width:  40,
			Height: 10,
		})
	}, goli.Options{Width: 40, Height: 10, Output: &output, DisableThrottle: true})

	// Initial state
	state := editorState()
	t.Logf("Initial mode: %s", state.Mode)
	if state.Mode != NormalMode {
		t.Errorf("Expected NormalMode, got %s", state.Mode)
	}

	// Simulate pressing 'i' to enter insert mode
	t.Log("Pressing 'i'...")
	handleNormalMode("i", application)

	state = editorState()
	t.Logf("After 'i' mode: %s", state.Mode)
	if state.Mode != InsertMode {
		t.Errorf("Expected InsertMode after pressing 'i', got %s", state.Mode)
	}

	// Type a character
	t.Log("Typing 'X'...")
	handleInsertMode("X", application)

	state = editorState()
	t.Logf("After typing 'X' - Line: %q, CursorX: %d", state.Lines[0], state.CursorX)
	if state.Lines[0] != "XHello" {
		t.Errorf("Expected 'XHello', got %q", state.Lines[0])
	}
	if state.CursorX != 1 {
		t.Errorf("Expected cursor at 1, got %d", state.CursorX)
	}

	// Verify the render shows the updated text
	buf := application.Renderer().CurrentBuffer()
	debugStr := buf.ToDebugString()
	t.Logf("After insert:\n%s", debugStr)
	if !strings.Contains(debugStr, "XHello") {
		t.Error("Expected 'XHello' in rendered output")
	}
	if !strings.Contains(debugStr, "INSERT") {
		t.Error("Expected 'INSERT' mode indicator")
	}

	// Press Escape to go back to normal mode
	t.Log("Pressing Escape...")
	handleInsertMode(goli.Escape, application)

	state = editorState()
	t.Logf("After Escape mode: %s", state.Mode)
	if state.Mode != NormalMode {
		t.Errorf("Expected NormalMode after Escape, got %s", state.Mode)
	}

	application.Dispose()
}

func TestVimEditor_CursorMovement(t *testing.T) {
	resetState()
	setEditorState(EditorState{
		Lines:    []string{"Hello", "World", "Test"},
		CursorX:  0,
		CursorY:  0,
		Mode:     NormalMode,
		Command:  "",
		Message:  "",
		Filename: "test.txt",
	})

	var output strings.Builder
	var application *goli.App

	application = goli.Render(func() gox.VNode {
		return Editor(EditorProps{
			Width:  40,
			Height: 10,
		})
	}, goli.Options{Width: 40, Height: 10, Output: &output, DisableThrottle: true})

	// Move right
	handleNormalMode("l", application)
	state := editorState()
	t.Logf("After 'l': CursorX=%d, CursorY=%d", state.CursorX, state.CursorY)
	if state.CursorX != 1 {
		t.Errorf("Expected CursorX=1 after 'l', got %d", state.CursorX)
	}

	// Move down
	handleNormalMode("j", application)
	state = editorState()
	t.Logf("After 'j': CursorX=%d, CursorY=%d", state.CursorX, state.CursorY)
	if state.CursorY != 1 {
		t.Errorf("Expected CursorY=1 after 'j', got %d", state.CursorY)
	}

	// Move left
	handleNormalMode("h", application)
	state = editorState()
	t.Logf("After 'h': CursorX=%d, CursorY=%d", state.CursorX, state.CursorY)
	if state.CursorX != 0 {
		t.Errorf("Expected CursorX=0 after 'h', got %d", state.CursorX)
	}

	// Move up
	handleNormalMode("k", application)
	state = editorState()
	t.Logf("After 'k': CursorX=%d, CursorY=%d", state.CursorX, state.CursorY)
	if state.CursorY != 0 {
		t.Errorf("Expected CursorY=0 after 'k', got %d", state.CursorY)
	}

	application.Dispose()
}

func TestVimEditor_CommandMode(t *testing.T) {
	resetState()
	setEditorState(EditorState{
		Lines:    []string{"Hello"},
		CursorX:  0,
		CursorY:  0,
		Mode:     NormalMode,
		Command:  "",
		Message:  "",
		Filename: "test.txt",
	})

	var output strings.Builder
	var application *goli.App

	application = goli.Render(func() gox.VNode {
		return Editor(EditorProps{
			Width:  40,
			Height: 10,
		})
	}, goli.Options{Width: 40, Height: 10, Output: &output, DisableThrottle: true})

	// Enter command mode with ':'
	handleNormalMode(":", application)
	state := editorState()
	t.Logf("After ':' mode: %s", state.Mode)
	if state.Mode != CommandMode {
		t.Errorf("Expected CommandMode after ':', got %s", state.Mode)
	}

	// Verify command line is rendered
	buf := application.Renderer().CurrentBuffer()
	debugStr := buf.ToDebugString()
	t.Logf("Command mode render:\n%s", debugStr)
	if !strings.Contains(debugStr, "COMMAND") {
		t.Error("Expected 'COMMAND' mode indicator")
	}

	// Type 'help'
	handleCommandMode("h", application)
	handleCommandMode("e", application)
	handleCommandMode("l", application)
	handleCommandMode("p", application)

	state = editorState()
	t.Logf("Command: %q", state.Command)
	if state.Command != "help" {
		t.Errorf("Expected command 'help', got %q", state.Command)
	}

	// Execute with Enter
	handleCommandMode(goli.Enter, application)
	state = editorState()
	t.Logf("After Enter - Mode: %s, Message: %q", state.Mode, state.Message)
	if state.Mode != NormalMode {
		t.Errorf("Expected NormalMode after command execution, got %s", state.Mode)
	}
	if !strings.Contains(state.Message, "Commands:") {
		t.Errorf("Expected help message, got %q", state.Message)
	}

	application.Dispose()
}

func TestVimEditor_DeleteLine(t *testing.T) {
	resetState()
	setEditorState(EditorState{
		Lines:    []string{"Line 1", "Line 2", "Line 3"},
		CursorX:  0,
		CursorY:  1, // Start on line 2
		Mode:     NormalMode,
		Command:  "",
		Message:  "",
		Filename: "test.txt",
	})

	var output strings.Builder
	var application *goli.App

	application = goli.Render(func() gox.VNode {
		return Editor(EditorProps{
			Width:  40,
			Height: 10,
		})
	}, goli.Options{Width: 40, Height: 10, Output: &output, DisableThrottle: true})

	// Delete line with 'd'
	handleNormalMode("d", application)

	state := editorState()
	t.Logf("After 'd': Lines=%v, CursorY=%d", state.Lines, state.CursorY)
	if len(state.Lines) != 2 {
		t.Errorf("Expected 2 lines after delete, got %d", len(state.Lines))
	}
	if state.Lines[0] != "Line 1" || state.Lines[1] != "Line 3" {
		t.Errorf("Expected [Line 1, Line 3], got %v", state.Lines)
	}

	application.Dispose()
}

func TestVimEditor_NewLine(t *testing.T) {
	resetState()
	setEditorState(EditorState{
		Lines:    []string{"Line 1"},
		CursorX:  0,
		CursorY:  0,
		Mode:     NormalMode,
		Command:  "",
		Message:  "",
		Filename: "test.txt",
	})

	var output strings.Builder
	var application *goli.App

	application = goli.Render(func() gox.VNode {
		return Editor(EditorProps{
			Width:  40,
			Height: 10,
		})
	}, goli.Options{Width: 40, Height: 10, Output: &output, DisableThrottle: true})

	// Press 'o' to open new line below
	handleNormalMode("o", application)

	state := editorState()
	t.Logf("After 'o': Lines=%v, Mode=%s, CursorY=%d", state.Lines, state.Mode, state.CursorY)
	if len(state.Lines) != 2 {
		t.Errorf("Expected 2 lines after 'o', got %d", len(state.Lines))
	}
	if state.Mode != InsertMode {
		t.Errorf("Expected InsertMode after 'o', got %s", state.Mode)
	}
	if state.CursorY != 1 {
		t.Errorf("Expected cursor on line 1 after 'o', got %d", state.CursorY)
	}

	// Type some text
	handleInsertMode("N", application)
	handleInsertMode("e", application)
	handleInsertMode("w", application)

	state = editorState()
	if state.Lines[1] != "New" {
		t.Errorf("Expected 'New' on line 2, got %q", state.Lines[1])
	}

	application.Dispose()
}

func TestVimEditor_FocusManagerIntegration(t *testing.T) {
	resetState()
	setEditorState(EditorState{
		Lines:    []string{"Test"},
		CursorX:  0,
		CursorY:  0,
		Mode:     NormalMode,
		Command:  "",
		Message:  "",
		Filename: "test.txt",
	})

	var output strings.Builder
	var application *goli.App

	application = goli.Render(func() gox.VNode {
		return Editor(EditorProps{
			Width:  40,
			Height: 10,
		})
	}, goli.Options{Width: 40, Height: 10, Output: &output, DisableThrottle: true})

	// Set up the global key handler like the real main() does
	cleanup := goli.Manager().SetGlobalKeyHandler(func(key string) bool {
		state := editorState()
		switch state.Mode {
		case NormalMode:
			return handleNormalMode(key, application)
		case InsertMode:
			return handleInsertMode(key, application)
		case CommandMode:
			return handleCommandMode(key, application)
		}
		return false
	})
	defer cleanup()

	// Now simulate using goli.HandleKey like the real app does
	goli.HandleKey("i") // Enter insert mode
	state := editorState()
	if state.Mode != InsertMode {
		t.Errorf("Expected InsertMode after goli.HandleKey('i'), got %s", state.Mode)
	}

	goli.HandleKey("A") // Type 'A'
	state = editorState()
	if state.Lines[0] != "ATest" {
		t.Errorf("Expected 'ATest', got %q", state.Lines[0])
	}

	goli.HandleKey(goli.Escape) // Exit insert mode
	state = editorState()
	if state.Mode != NormalMode {
		t.Errorf("Expected NormalMode after Escape, got %s", state.Mode)
	}

	application.Dispose()
}
