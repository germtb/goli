//go:build linux

// Package term provides terminal handling utilities.
package goli

import (
	"os"
	"syscall"
	"unsafe"
)

// termios represents the terminal I/O settings (Linux).
type termios struct {
	Iflag  uint32
	Oflag  uint32
	Cflag  uint32
	Lflag  uint32
	Line   uint8
	Cc     [32]byte
	Ispeed uint32
	Ospeed uint32
}

const (
	// Flags for ioctl - Linux uses TCGETS/TCSETS
	getTermios = 0x5401 // TCGETS on Linux
	setTermios = 0x5402 // TCSETS on Linux

	// Input mode flags
	ICRNL  = 0x00000100
	IXON   = 0x00000400
	BRKINT = 0x00000002
	INPCK  = 0x00000010
	ISTRIP = 0x00000020

	// Local mode flags
	ECHO   = 0x00000008
	ICANON = 0x00000002
	ISIG   = 0x00000001
	IEXTEN = 0x00008000

	// Output mode flags
	OPOST = 0x00000001

	// Control mode flags
	CS8 = 0x00000030
)

// State holds the terminal state for later restoration.
type State struct {
	termios termios
}

// MakeRaw puts the terminal into raw mode and returns the previous state.
func MakeRaw(fd int) (*State, error) {
	var oldState termios

	// Get current terminal settings
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		getTermios,
		uintptr(unsafe.Pointer(&oldState)),
	)
	if errno != 0 {
		return nil, errno
	}

	newState := oldState

	// Disable input processing
	newState.Iflag &^= BRKINT | ICRNL | INPCK | ISTRIP | IXON

	// Disable output processing
	newState.Oflag &^= OPOST

	// Set character size to 8 bits
	newState.Cflag |= CS8

	// Disable canonical mode, echo, and signals
	newState.Lflag &^= ECHO | ICANON | IEXTEN | ISIG

	// Minimum number of characters for non-canonical read
	// On Linux, VMIN is at index 6, VTIME is at index 5
	newState.Cc[6] = 1 // VMIN
	newState.Cc[5] = 0 // VTIME

	// Apply new settings
	_, _, errno = syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		setTermios,
		uintptr(unsafe.Pointer(&newState)),
	)
	if errno != 0 {
		return nil, errno
	}

	return &State{termios: oldState}, nil
}

// Restore restores the terminal to a previous state.
func Restore(fd int, state *State) error {
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		setTermios,
		uintptr(unsafe.Pointer(&state.termios)),
	)
	if errno != 0 {
		return errno
	}
	return nil
}

// GetSize returns the terminal dimensions.
func GetSize(fd int) (width, height int, err error) {
	var ws struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}

	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		syscall.TIOCGWINSZ,
		uintptr(unsafe.Pointer(&ws)),
	)
	if errno != 0 {
		return 0, 0, errno
	}

	return int(ws.Col), int(ws.Row), nil
}

// IsTerminal returns whether the file descriptor is a terminal.
func IsTerminal(fd int) bool {
	var termios termios
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		getTermios,
		uintptr(unsafe.Pointer(&termios)),
	)
	return errno == 0
}

// Stdin returns the file descriptor for stdin.
func Stdin() int {
	return int(os.Stdin.Fd())
}

// Stdout returns the file descriptor for stdout.
func Stdout() int {
	return int(os.Stdout.Fd())
}
