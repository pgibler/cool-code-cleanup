package tui

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type IO struct {
	In  *bufio.Reader
	Out io.Writer
}

func NewIO(in io.Reader, out io.Writer) IO {
	return IO{
		In:  bufio.NewReader(in),
		Out: out,
	}
}

func (io IO) Prompt(prompt string) (string, error) {
	fmt.Fprint(io.Out, prompt)
	line, err := io.In.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func (io IO) RunToggleStep(screen StepScreen, list *ToggleList) (accepted bool, canceled bool, err error) {
	if canUseRawTTY() {
		return io.runToggleStepRaw(screen, list)
	}
	return io.runToggleStepLine(screen, list)
}

func (io IO) runToggleStepLine(screen StepScreen, list *ToggleList) (accepted bool, canceled bool, err error) {
	for {
		render := screen
		render.Content = list.RenderLines()
		fmt.Fprintln(io.Out, render.Render())
		input, readErr := io.Prompt("Command (up/down/space/accept/back/cancel): ")
		if readErr != nil {
			return false, false, readErr
		}
		switch strings.ToLower(strings.TrimSpace(input)) {
		case "up", "u":
			list.MoveUp()
		case "down", "d":
			list.MoveDown()
		case "space", "toggle", "t":
			_, reason := list.ToggleCurrent()
			render.InlineError = reason
		case "accept", "a", "":
			return true, false, nil
		case "back", "b":
			return false, false, nil
		case "cancel", "c", "quit", "q":
			return false, true, nil
		default:
			render.InlineError = "unknown command"
		}
	}
}

func (io IO) runToggleStepRaw(screen StepScreen, list *ToggleList) (accepted bool, canceled bool, err error) {
	prev, err := enableRawMode()
	if err != nil {
		return io.runToggleStepLine(screen, list)
	}
	defer restoreRawMode(prev)
	enterFullscreen(io.Out)
	defer leaveFullscreen(io.Out)

	resizeCh := make(chan os.Signal, 1)
	signal.Notify(resizeCh, syscall.SIGWINCH)
	defer signal.Stop(resizeCh)

	inlineErr := ""
	dirty := true
	for {
		if dirty {
			render := screen
			render.Content = list.RenderLines()
			render.InlineError = inlineErr
			render.Actions = append(append([]Action{}, screen.Actions...),
				Action{Key: "↑/↓", Label: "Move"},
				Action{Key: "Space", Label: "Toggle"},
				Action{Key: "Enter", Label: "Accept"},
			)
			width := terminalWidth()
			frame := render.RenderWithWidth(width)
			fmt.Fprint(io.Out, "\x1b[H\x1b[2J")
			fmt.Fprint(io.Out, strings.ReplaceAll(frame, "\n", "\r\n"))
			dirty = false
		}

		select {
		case <-resizeCh:
			dirty = true
			continue
		default:
		}

		b, ok, rerr := readByteWithTimeout(200 * time.Millisecond)
		if rerr != nil {
			return false, false, rerr
		}
		if !ok {
			continue
		}
		inlineErr = ""
		switch b {
		case 27: // ESC sequence
			next, ok, e1 := readByteWithTimeout(15 * time.Millisecond)
			if e1 != nil || !ok {
				continue
			}
			if next != '[' {
				continue
			}
			key, ok, e2 := readByteWithTimeout(15 * time.Millisecond)
			if e2 != nil || !ok {
				continue
			}
			switch key {
			case 'A':
				list.MoveUp()
			case 'B':
				list.MoveDown()
			}
			dirty = true
		case ' ':
			_, reason := list.ToggleCurrent()
			inlineErr = reason
			dirty = true
		case '\r', '\n':
			return true, false, nil
		case 'b', 'B':
			return false, false, nil
		case 'c', 'C', 'q', 'Q':
			return false, true, nil
		default:
			dirty = true
		}
	}
}

func canUseRawTTY() bool {
	if runtime.GOOS == "windows" {
		return false
	}
	if _, err := exec.LookPath("stty"); err != nil {
		return false
	}
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func enableRawMode() (string, error) {
	prev, err := stty("-g")
	if err != nil {
		return "", err
	}
	if _, err := stty("raw", "-echo"); err != nil {
		return "", err
	}
	return prev, nil
}

func restoreRawMode(prev string) {
	if strings.TrimSpace(prev) == "" {
		return
	}
	_, _ = stty(prev)
}

func stty(args ...string) (string, error) {
	cmd := exec.Command("stty", args...)
	cmd.Stdin = os.Stdin
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return strings.TrimSpace(out.String()), err
}

func enterFullscreen(out io.Writer) {
	fmt.Fprint(out, "\x1b[?1049h\x1b[?25l\x1b[H\x1b[2J")
}

func leaveFullscreen(out io.Writer) {
	fmt.Fprint(out, "\x1b[?25h\x1b[?1049l")
}

func terminalWidth() int {
	out, err := stty("size")
	if err != nil {
		return 100
	}
	fields := strings.Fields(out)
	if len(fields) != 2 {
		return 100
	}
	cols, err := strconv.Atoi(fields[1])
	if err != nil || cols < 40 {
		return 100
	}
	return cols
}

func readByteWithTimeout(timeout time.Duration) (byte, bool, error) {
	fd := int(os.Stdin.Fd())
	var readfds syscall.FdSet
	fdSet(fd, &readfds)
	tv := syscall.NsecToTimeval(timeout.Nanoseconds())
	n, err := syscall.Select(fd+1, &readfds, nil, nil, &tv)
	if err != nil {
		return 0, false, err
	}
	if n == 0 || !fdIsSet(fd, &readfds) {
		return 0, false, nil
	}
	var one [1]byte
	rn, err := os.Stdin.Read(one[:])
	if err != nil {
		return 0, false, err
	}
	if rn == 0 {
		return 0, false, nil
	}
	return one[0], true, nil
}

func fdSet(fd int, set *syscall.FdSet) {
	set.Bits[fd/64] |= 1 << (uint(fd) % 64)
}

func fdIsSet(fd int, set *syscall.FdSet) bool {
	return set.Bits[fd/64]&(1<<(uint(fd)%64)) != 0
}
