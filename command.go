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

	// Aliases for the command
	Aliases []string

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
	Behavior func(ctx *Context)

	Middleware func(ctx *Context)

	node *commandNode
}

func (c *Command) fullName() string {
	if c.node.parent != nil {
		return fmt.Sprintf("%s %s", c.node.parent.value.Name, c.Name)
	}
	return filepath.Base(os.Args[0])
}

func (c *Command) exec(args []string, ctx *Context) {

	// build the context
	buildContext(c, args, ctx)

	c.Behavior(ctx)
}

func validateRoot(c *Command) error {
	if len(c.Aliases) > 0 {
		return newSetupError("Root command cannot have aliases.")
	}

	return validateCommand(c)
}

func validateCommand(c *Command) error {
	if c.Name == "" {
		return newSetupError("Received Command definition with an empty name")
	}

	if c.Options != nil {
		return validateOptions(&(c.Options))
	}

	return nil
}
