// Package goli provides focus management for terminal UI components.
package goli

// Common terminal key codes.
const (
	// Basic keys
	Space   = " "
	Enter   = "\r"
	EnterLF = "\n"
	Tab     = "\t"
	Escape  = "\x1b"

	// Editing keys
	Backspace     = "\x7f"
	BackspaceCtrl = "\b"
	Delete        = "\x1b[3~"
	Insert        = "\x1b[2~"

	// Navigation keys
	Left     = "\x1b[D"
	Right    = "\x1b[C"
	Up       = "\x1b[A"
	Down     = "\x1b[B"
	Home     = "\x1b[H"
	HomeAlt  = "\x1b[1~"
	End      = "\x1b[F"
	EndAlt   = "\x1b[4~"
	PageUp   = "\x1b[5~"
	PageDown = "\x1b[6~"

	// Shift combinations
	ShiftTab   = "\x1b[Z"
	ShiftEnter = "\x1b[13;2u"
	ShiftUp    = "\x1b[1;2A"
	ShiftDown  = "\x1b[1;2B"
	ShiftLeft  = "\x1b[1;2D"
	ShiftRight = "\x1b[1;2C"

	// Alt combinations
	AltBackspace = "\x1b\x7f"
	AltLeft      = "\x1bb"
	AltLeftCSI   = "\x1b[1;3D"
	AltRight     = "\x1bf"
	AltRightCSI  = "\x1b[1;3C"
	AltUp        = "\x1b[1;3A"
	AltDown      = "\x1b[1;3B"

	// Ctrl combinations (alphabetical)
	CtrlA = "\x01"
	CtrlB = "\x02"
	CtrlC = "\x03"
	CtrlD = "\x04"
	CtrlE = "\x05"
	CtrlF = "\x06"
	CtrlG = "\x07"
	CtrlH = "\x08" // Same as BackspaceCtrl
	CtrlI = "\x09" // Same as Tab
	CtrlJ = "\x0a" // Same as EnterLF
	CtrlK = "\x0b"
	CtrlL = "\x0c"
	CtrlM = "\x0d" // Same as Enter
	CtrlN = "\x0e"
	CtrlO = "\x0f"
	CtrlP = "\x10"
	CtrlQ = "\x11"
	CtrlR = "\x12"
	CtrlS = "\x13"
	CtrlT = "\x14"
	CtrlU = "\x15"
	CtrlV = "\x16"
	CtrlW = "\x17"
	CtrlX = "\x18"
	CtrlY = "\x19"
	CtrlZ = "\x1a"

	// Ctrl+Arrow combinations
	CtrlUp    = "\x1b[1;5A"
	CtrlDown  = "\x1b[1;5B"
	CtrlLeft  = "\x1b[1;5D"
	CtrlRight = "\x1b[1;5C"

	// Function keys
	F1  = "\x1bOP"
	F2  = "\x1bOQ"
	F3  = "\x1bOR"
	F4  = "\x1bOS"
	F5  = "\x1b[15~"
	F6  = "\x1b[17~"
	F7  = "\x1b[18~"
	F8  = "\x1b[19~"
	F9  = "\x1b[20~"
	F10 = "\x1b[21~"
	F11 = "\x1b[23~"
	F12 = "\x1b[24~"
)
