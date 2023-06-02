package gocli

import (
	"fmt"
	"os"
)

func newSetupError(msg string, a ...any) error {
	return fmt.Errorf("%s Error occurred during setup.\n%s %s", Red("[ERROR]"), Red("[ERROR]"), fmt.Sprintf(msg, a...))
}

func newError(msg string, a ...any) error {
	return fmt.Errorf("%s %s", Red("[ERROR]"), fmt.Sprintf(msg, a...))
}

func fatal(err error) {
	fmt.Println(err)
	os.Exit(1)
}
