package gocli

import (
	"fmt"
	"os"
	"path/filepath"
)

// CLI Command
type Command struct {
	// Name of the command (as referenced in the CLI)
	Name string

	// Description that is shown when "--help" is present
	LongDesc string

	// Description that is shown when "--help" is present
	// on the parent command
	ShortDesc string

	// Options
	Options Options

	// Argument
	Arguments interface{}

	// Behavior of the command
	Behavior func(ctx Context)

	node *commandNode
}

func (c *Command) fullName() string {
	if c.node.parent != nil {
		return fmt.Sprintf("%s %s", c.node.parent.value.Name, c.Name)
	}
	return filepath.Base(os.Args[0])
}

func (c *Command) exec(args []string) {

	// build the context
	ctx := buildContext(c, args)

	c.Behavior(ctx)
}

func validateCommand(c *Command) error {
	if c == nil {
		return newSetupError("Received nil Command definition")
	}

	if c.Name == "" {
		return newSetupError("Received Command definition with an empty name")
	}

	return validateOptions(&(c.Options))
}
