package gocli

import (
	"fmt"

	color "github.com/fatih/color"
)

func Blue(s string, a ...any) string {
	return color.New(color.FgBlue).SprintFunc()(fmt.Sprintf(s, a...))
}

func Red(s string, a ...any) string {
	return color.New(color.FgRed).SprintFunc()(fmt.Sprintf(s, a...))
}

func Yellow(s string, a ...any) string {
	return color.New(color.FgYellow).SprintFunc()(fmt.Sprintf(s, a...))
}

func Green(s string, a ...any) string {
	return color.New(color.FgGreen).SprintFunc()(fmt.Sprintf(s, a...))
}

func White(s string, a ...any) string {
	return color.New(color.FgWhite).SprintFunc()(fmt.Sprintf(s, a...))
}

func Cyan(s string, a ...any) string {
	return color.New(color.FgCyan).SprintFunc()(fmt.Sprintf(s, a...))
}

func Magenta(s string, a ...any) string {
	return color.New(color.FgMagenta).SprintFunc()(fmt.Sprintf(s, a...))
}
