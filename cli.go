package gocli

import (
	"fmt"
	"os"
)

type cli struct {
	nodes map[*Command]*commandNode
	root  *commandNode
}

func NewCli(root *Command) *cli {
	if err := validateRoot(root); err != nil {
		fatal(err)
	}

	rootNode := newCommandNode(nil, root)
	return &cli{
		nodes: map[*Command]*commandNode{root: rootNode},
		root:  rootNode,
	}
}

func (cli *cli) AddChild(p, c *Command) {
	if err := validateCommand(p); err != nil {
		fatal(err)
	}

	if err := validateCommand(c); err != nil {
		fatal(err)
	}

	if !cli.hasCommand(p) {
		fatal(newSetupError("Could not addChild with parent %s and child %s. Parent command does not exist.", p.Name, c.Name))
	}

	if cli.hasCommand(c) {
		fatal(newSetupError("Could not addChild with parent %s and child %s. Child command already exists", p.Name, c.Name))
	}

	parent := cli.nodes[p]
	if parent.hasChild(c.Name) {
		fatal(newSetupError("Could not addChild %s to parent %s. A child command already exists with the name %s", c.Name, p.Name, c.Name))
	}

	child := newCommandNode(parent, c)
	cli.nodes[c] = child
}

func (cli *cli) hasCommand(c *Command) (exists bool) {
	if c == nil {
		return false
	}

	_, exists = cli.nodes[c]
	return
}

func (c *cli) Exec() {
	args := os.Args[1:]
	if i, ok := hasHelp(args); ok {
		args[i], args[len(args)-1] = args[len(args)-1], args[i]
	}

	node := c.root
	for {
		sca := getSubCommandArg(args)
		if node.hasChild(sca.subCommand) {
			node = node.children[sca.subCommand]
			args = sca.args
			continue
		}

		node.value.exec(args)
		return
	}
}

type commandNode struct {
	parent   *commandNode
	children map[string]*commandNode
	value    *Command
}

func newCommandNode(parent *commandNode, command *Command) *commandNode {
	node := &commandNode{
		parent:   parent,
		children: map[string]*commandNode{},
		value:    command,
	}
	command.node = node

	if parent != nil {
		parent.children[command.Name] = node

		for _, alias := range command.Aliases {
			if _, exists := parent.children[alias]; exists {
				fatal(newError("Command '%s' has two identical children/aliases '%s'. This is not allowed", parent.value.Name, alias))
			}
			parent.children[alias] = node
		}
	}

	if command.Behavior == nil {
		command.Behavior = func(ctx Context) {
			fmt.Println(ctx.GetHelpStr())
		}
	}

	return node
}

func (n *commandNode) hasChild(child string) (exists bool) {
	_, exists = n.children[child]
	return
}

type subCommandArg struct {
	subCommand string
	args       []string
}

func getSubCommandArg(args []string) subCommandArg {
	if len(args) <= 0 {
		return subCommandArg{"", []string{}}
	}

	if _, is := isOption(args[0]); is {
		return subCommandArg{"", args}
	}

	return subCommandArg{args[0], args[1:]}
}
