// Package goli provides text input handling for terminal UI.
package goli

import (
	"unicode"

	"github.com/germtb/goli/signals"
)

// InputState represents the state of an input field.
type InputState struct {
	Value     string
	CursorPos int
}

// InputKeyHandler is a keypress handler.
// Return new state to consume the key, or nil to let it bubble up.
type InputKeyHandler func(key string, state InputState) *InputState

// InputOptions configures input creation.
type InputOptions struct {
	// InitialValue is the starting text.
	InitialValue string
	// MaxLength limits the number of characters (0 = unlimited).
	MaxLength int
	// Mask character for passwords (e.g., "*").
	Mask rune
	// Placeholder text shown when input is empty.
	Placeholder string
	// OnKeypress is a custom keypress handler.
	OnKeypress InputKeyHandler
}

// Input represents a text input field.
type Input struct {
	value      signals.Accessor[string]
	setValue   signals.Setter[string]
	cursorPos  signals.Accessor[int]
	setCursor  signals.Setter[int]
	focused    signals.Accessor[bool]
	setFocused signals.Setter[bool]

	maxLength   int
	mask        rune
	placeholder string
	onKeypress  InputKeyHandler
}

// NewInput creates a new input field.
func NewInput(opts InputOptions) *Input {
	value, setValue := signals.CreateSignal(opts.InitialValue)
	cursorPos, setCursor := signals.CreateSignal(len(opts.InitialValue))
	focused, setFocused := signals.CreateSignal(false)

	handler := opts.OnKeypress
	if handler == nil {
		handler = DefaultInputHandler
	}

	inp := &Input{
		value:       value,
		setValue:    setValue,
		cursorPos:   cursorPos,
		setCursor:   setCursor,
		focused:     focused,
		setFocused:  setFocused,
		maxLength:   opts.MaxLength,
		mask:        opts.Mask,
		placeholder: opts.Placeholder,
		onKeypress:  handler,
	}

	// Register with focus manager
	Register(inp)

	return inp
}

// Value returns the current text value.
func (i *Input) Value() string {
	return i.value()
}

// CursorPos returns the cursor position.
func (i *Input) CursorPos() int {
	return i.cursorPos()
}

// Focused returns whether the input is focused.
func (i *Input) Focused() bool {
	return i.focused()
}

// Focus gives focus to this input.
func (i *Input) Focus() {
	RequestFocus(i)
}

// Blur removes focus from this input.
func (i *Input) Blur() {
	RequestBlur(i)
}

// SetFocused sets the focused state (called by focus manager).
func (i *Input) SetFocused(f bool) {
	i.setFocused(f)
}

// Dispose unregisters from the focus manager.
func (i *Input) Dispose() {
	Unregister(i)
}

// HandleKey processes a key press.
// Returns true if the key was consumed.
func (i *Input) HandleKey(key string) bool {
	if !i.focused() {
		return false
	}

	state := i.GetState()
	newState := i.onKeypress(key, state)
	if newState == nil {
		return false
	}
	i.setState(*newState)
	return true
}

// SetValue updates the text value.
func (i *Input) SetValue(value string) {
	limited := i.applyMaxLength(value)
	signals.BatchVoid(func() {
		i.setValue(limited)
		i.setCursor(i.clampCursor(i.cursorPos(), len(limited)))
	})
}

// SetCursorPos updates the cursor position.
func (i *Input) SetCursorPos(pos int) {
	i.setCursor(i.clampCursor(pos, len(i.value())))
}

// Clear clears the input.
func (i *Input) Clear() {
	signals.BatchVoid(func() {
		i.setValue("")
		i.setCursor(0)
	})
}

// DisplayValue returns the display text (with masking/placeholder).
func (i *Input) DisplayValue() string {
	val := i.value()
	if len(val) == 0 && i.placeholder != "" {
		return i.placeholder
	}
	if i.mask != 0 {
		masked := make([]rune, len(val))
		for j := range masked {
			masked[j] = i.mask
		}
		return string(masked)
	}
	return val
}

// ShowingPlaceholder returns true if displaying placeholder text.
func (i *Input) ShowingPlaceholder() bool {
	return len(i.value()) == 0 && i.placeholder != ""
}

// GetState returns the current state snapshot.
func (i *Input) GetState() InputState {
	return InputState{
		Value:     i.value(),
		CursorPos: i.cursorPos(),
	}
}

func (i *Input) setState(state InputState) {
	limited := i.applyMaxLength(state.Value)
	clamped := i.clampCursor(state.CursorPos, len(limited))
	signals.BatchVoid(func() {
		i.setValue(limited)
		i.setCursor(clamped)
	})
}

func (i *Input) applyMaxLength(val string) string {
	if i.maxLength > 0 && len(val) > i.maxLength {
		return val[:i.maxLength]
	}
	return val
}

func (i *Input) clampCursor(pos, length int) int {
	if pos < 0 {
		return 0
	}
	if pos > length {
		return length
	}
	return pos
}

// DefaultInputHandler implements standard text editing behavior.
var DefaultInputHandler = ComposeInputHandlers(
	InputNavigationHandler,
	InputDeletionHandler,
	InputShiftEnterHandler,
	InputPrintableHandler,
)

// ComposeInputHandlers combines multiple handlers into one.
// Handlers are tried in order until one returns non-nil.
func ComposeInputHandlers(handlers ...InputKeyHandler) InputKeyHandler {
	return func(key string, state InputState) *InputState {
		for _, h := range handlers {
			if result := h(key, state); result != nil {
				return result
			}
		}
		return nil
	}
}

// InputPrintableHandler inserts printable characters at cursor.
func InputPrintableHandler(key string, state InputState) *InputState {
	if len(key) >= 1 && isPrintable(key) {
		newValue := state.Value[:state.CursorPos] + key + state.Value[state.CursorPos:]
		return &InputState{
			Value:     newValue,
			CursorPos: state.CursorPos + len(key),
		}
	}
	return nil
}

// InputNavigationHandler handles arrow keys, home/end, word navigation.
func InputNavigationHandler(key string, state InputState) *InputState {
	switch key {
	case Left:
		if state.CursorPos > 0 {
			return &InputState{Value: state.Value, CursorPos: state.CursorPos - 1}
		}
		return &state

	case Right:
		if state.CursorPos < len(state.Value) {
			return &InputState{Value: state.Value, CursorPos: state.CursorPos + 1}
		}
		return &state

	case AltLeft, AltLeftCSI:
		// Move to start of previous word
		newPos := state.CursorPos
		for newPos > 0 && !isWordChar(rune(state.Value[newPos-1])) {
			newPos--
		}
		for newPos > 0 && isWordChar(rune(state.Value[newPos-1])) {
			newPos--
		}
		return &InputState{Value: state.Value, CursorPos: newPos}

	case AltRight, AltRightCSI:
		// Move to end of next word
		newPos := state.CursorPos
		for newPos < len(state.Value) && !isWordChar(rune(state.Value[newPos])) {
			newPos++
		}
		for newPos < len(state.Value) && isWordChar(rune(state.Value[newPos])) {
			newPos++
		}
		return &InputState{Value: state.Value, CursorPos: newPos}

	case Home, HomeAlt, CtrlA:
		lineStart := getLineStart(state.Value, state.CursorPos)
		return &InputState{Value: state.Value, CursorPos: lineStart}

	case End, EndAlt, CtrlE:
		lineEnd := getLineEnd(state.Value, state.CursorPos)
		return &InputState{Value: state.Value, CursorPos: lineEnd}

	case Up:
		newPos := moveCursorUp(state.Value, state.CursorPos)
		if newPos != state.CursorPos {
			return &InputState{Value: state.Value, CursorPos: newPos}
		}
		return &state

	case Down:
		newPos := moveCursorDown(state.Value, state.CursorPos)
		if newPos != state.CursorPos {
			return &InputState{Value: state.Value, CursorPos: newPos}
		}
		return &state
	}

	return nil
}

// InputDeletionHandler handles backspace, delete, word delete.
func InputDeletionHandler(key string, state InputState) *InputState {
	switch key {
	case Backspace, BackspaceCtrl:
		if state.CursorPos == 0 {
			return &state
		}
		return &InputState{
			Value:     state.Value[:state.CursorPos-1] + state.Value[state.CursorPos:],
			CursorPos: state.CursorPos - 1,
		}

	case Delete:
		if state.CursorPos >= len(state.Value) {
			return &state
		}
		return &InputState{
			Value:     state.Value[:state.CursorPos] + state.Value[state.CursorPos+1:],
			CursorPos: state.CursorPos,
		}

	case CtrlU:
		// Delete from cursor to start of line
		lineStart := getLineStart(state.Value, state.CursorPos)
		return &InputState{
			Value:     state.Value[:lineStart] + state.Value[state.CursorPos:],
			CursorPos: lineStart,
		}

	case CtrlW, AltBackspace:
		// Delete previous word
		if state.CursorPos == 0 {
			return &state
		}
		newPos := state.CursorPos
		for newPos > 0 && !isWordChar(rune(state.Value[newPos-1])) {
			newPos--
		}
		for newPos > 0 && isWordChar(rune(state.Value[newPos-1])) {
			newPos--
		}
		return &InputState{
			Value:     state.Value[:newPos] + state.Value[state.CursorPos:],
			CursorPos: newPos,
		}
	}

	return nil
}

// InputNewlineHandler inserts newline on Enter (for multiline editors).
func InputNewlineHandler(key string, state InputState) *InputState {
	if key == Enter || key == EnterLF || key == ShiftEnter {
		return &InputState{
			Value:     state.Value[:state.CursorPos] + "\n" + state.Value[state.CursorPos:],
			CursorPos: state.CursorPos + 1,
		}
	}
	return nil
}

// InputShiftEnterHandler inserts newline only on Shift+Enter.
func InputShiftEnterHandler(key string, state InputState) *InputState {
	if key == ShiftEnter || key == EnterLF {
		return &InputState{
			Value:     state.Value[:state.CursorPos] + "\n" + state.Value[state.CursorPos:],
			CursorPos: state.CursorPos + 1,
		}
	}
	return nil
}

// Helper functions

func isPrintable(s string) bool {
	for _, r := range s {
		if r < ' ' || r > '~' {
			return false
		}
	}
	return true
}

func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func getLineStart(value string, pos int) int {
	for i := pos - 1; i >= 0; i-- {
		if value[i] == '\n' {
			return i + 1
		}
	}
	return 0
}

func getLineEnd(value string, pos int) int {
	for i := pos; i < len(value); i++ {
		if value[i] == '\n' {
			return i
		}
	}
	return len(value)
}

func moveCursorUp(value string, pos int) int {
	lineStarts := []int{0}
	for i := 0; i < len(value); i++ {
		if value[i] == '\n' {
			lineStarts = append(lineStarts, i+1)
		}
	}

	// Find current line
	lineIndex := 0
	for i := len(lineStarts) - 1; i >= 0; i-- {
		if pos >= lineStarts[i] {
			lineIndex = i
			break
		}
	}

	if lineIndex == 0 {
		return pos // Already on first line
	}

	column := pos - lineStarts[lineIndex]
	prevLineStart := lineStarts[lineIndex-1]
	prevLineEnd := lineStarts[lineIndex] - 1
	prevLineLen := prevLineEnd - prevLineStart

	newPos := prevLineStart + column
	if column > prevLineLen {
		newPos = prevLineEnd
	}
	return newPos
}

func moveCursorDown(value string, pos int) int {
	lineStarts := []int{0}
	for i := 0; i < len(value); i++ {
		if value[i] == '\n' {
			lineStarts = append(lineStarts, i+1)
		}
	}

	// Find current line
	lineIndex := 0
	for i := len(lineStarts) - 1; i >= 0; i-- {
		if pos >= lineStarts[i] {
			lineIndex = i
			break
		}
	}

	if lineIndex >= len(lineStarts)-1 {
		return pos // Already on last line
	}

	column := pos - lineStarts[lineIndex]
	nextLineStart := lineStarts[lineIndex+1]
	var nextLineEnd int
	if lineIndex+2 < len(lineStarts) {
		nextLineEnd = lineStarts[lineIndex+2] - 1
	} else {
		nextLineEnd = len(value)
	}
	nextLineLen := nextLineEnd - nextLineStart

	newPos := nextLineStart + column
	if column > nextLineLen {
		newPos = nextLineEnd
	}
	return newPos
}
