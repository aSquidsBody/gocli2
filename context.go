package gocli

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
)

type Context struct {
	arguments []*argument

	options []*option

	helpStr string

	commandStr string
}

// GetHelpStr returns the same help text that is printed when "--help" or "-h" are included.
func (ctx *Context) GetHelpStr() string {
	return ctx.helpStr
}

// GetRawArgs returns the string CLI arguments that are passed by the user. i.e. "$@" in bash terms
func (ctx *Context) GetRawArgs() []string {
	return os.Args
}

// GetParentCommands returns an array containing the command + sub-commands that lead to the current command.
//
// For example, if the user runs `my-command sub-command1 sub-command2 --option1 -o2 arg1`, then this method would return []string{"my-command", "sub-command1"}
func (ctx *Context) GetParentCommands() []string {
	return strings.Split(ctx.commandStr, " ")
}

// GetArguments populates the "args" parameter with the CLI arguments
func (ctx *Context) GetArguments(args interface{}) {
	argumentsMap := map[string]interface{}{}
	for _, argument := range ctx.arguments {
		argumentsMap[convertToCamelCase(argument.name)] = argument.value
	}
	populateInterface(argumentsMap, args)
}

// GetOptions populates the "opt" parameter with CLI options
func (ctx *Context) GetOptions(opt interface{}) {
	optionsMap := map[string]interface{}{}
	for _, option := range ctx.options {
		optionsMap[convertToCamelCase(option.long)] = option.value
	}
	populateInterface(optionsMap, opt)
}

func populateInterface(m map[string]interface{}, i interface{}) {
	iValue := reflect.ValueOf(i).Elem()
	iType := iValue.Type()

	for idx := 0; idx < iType.NumField(); idx++ {
		field := iType.Field(idx)
		if value, ok := m[field.Name]; ok {
			iValue.Field(idx).Set(reflect.ValueOf(value))
		}
	}
}

func buildContext(c *Command, args []string) Context {
	ctx := Context{commandStr: c.fullName()}
	optionsMap := buildOptionsMap(c)
	ctx.arguments = buildArguments(c)
	ctx.helpStr = getHelpStr(optionsMap, ctx.arguments, c)

	if hasHelp(args) {
		fmt.Println(ctx.helpStr)
		os.Exit(0)
	}

	args, err := populateOptionsMap(optionsMap, args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	ctx.options = optionsMapToArray(optionsMap)

	err = populateArguments(ctx.arguments, args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return ctx
}

func hasHelp(args []string) bool {
	// check for help field
	for _, arg := range args {
		if strings.HasPrefix(arg, "-h=") || strings.HasPrefix(arg, "--help=") || arg == "-h" || arg == "--help" {
			return true
		}
	}

	return false
}

type Bylong []*option

func (b Bylong) Len() int           { return len(b) }
func (b Bylong) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b Bylong) Less(i, j int) bool { return b[i].long < b[j].long }

func getHelpStr(optionsMap map[string]*option, arguments []*argument, c *Command) string {
	options := optionsMapToArray(optionsMap)
	sort.Sort(Bylong(options))

	padding := 5
	txt := ""

	txt += fmt.Sprintf("Usage: %s", c.fullName())
	if len(c.node.children) > 0 {
		txt += " [COMMAND]"
	}

	if len(options) > 0 {
		txt += " [" + Green("OPTIONS") + "]"
	}

	if len(arguments) > 0 {
		for _, argument := range arguments {
			txt += fmt.Sprintf(" %s", Yellow(argument.name))
		}
	}

	txt += Sep()

	if c.LongDesc != "" {
		txt += c.LongDesc
		txt += Sep()
	}

	txt += Sep()

	if len(c.node.children) > 0 {
		txt += "Commands:" + Sep()

		maxWidth := 0
		for _, child := range c.node.children {
			maxWidth = max(len(child.value.Name), maxWidth)
		}
		width := maxWidth + padding
		for _, child := range c.node.children {
			txt += "  " + paddedName(child.value.Name, width) + child.value.ShortDesc + Sep()
		}

		txt += Sep()
	}

	if len(options) > 0 {
		txt += "Options:" + Sep()

		maxWidth := 0
		for _, option := range options {
			maxWidth = max(len(Green(optionNameHelp(option))), maxWidth)
		}
		width := maxWidth + padding
		for _, option := range options {
			required := "Optional"
			if option.required {
				required = Blue("Required")
			}
			txt += "  " + paddedName(Green(optionNameHelp(option)), width) + fmt.Sprintf("[%s, Type: %s] ", required, optionTypeHelp(option)) + option.description + Sep()
		}
		txt += Sep()
	}

	if len(arguments) > 0 {
		txt += "Arguments:" + Sep()

		maxWidth := 0
		for _, argument := range arguments {
			maxWidth = max(len(Yellow(argument.name)), maxWidth)
		}
		width := maxWidth + padding
		for _, argument := range arguments {
			required := "Optional"
			if argument.required {
				required = Blue("Required")
			}
			txt += "  " + paddedName(Yellow(argument.name), width) + fmt.Sprintf("[%s, Type: %s] ", required, optionTypeHelp(argument)) + argument.description + Sep()
		}
	}
	return txt
}

func max(i, j int) int {
	if i > j {
		return i
	}

	return j
}

func paddedName(name string, width int) (p string) {
	p += name
	for i := 0; i < width-len(name); i++ {
		p += " "
	}
	return p
}

func optionNameHelp(o *option) string {
	if o.short != "" && o.long != "" {
		return strings.Join([]string{"-" + o.short, "--" + o.long}, ",")
	}
	if o.short != "" {
		return "-" + o.short
	}
	if o.long != "" {
		return "--" + o.long
	}
	return ""
}

func optionTypeHelp(v valued) string {
	kind := v.getKind()
	if kind == reflect.String {
		return "string"
	}

	if kind == reflect.Bool {
		return "bool"
	}

	if kind == reflect.Int ||
		kind == reflect.Int8 ||
		kind == reflect.Int16 ||
		kind == reflect.Int32 ||
		kind == reflect.Int64 {
		return "int"
	}

	if kind == reflect.Float32 || kind == reflect.Float64 {
		return "float"
	}

	return "UNKNOWN (please check option definition)"
}
