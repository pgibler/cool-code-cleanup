package tui

import (
	"bufio"
	"fmt"
	"io"
	"strings"
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
