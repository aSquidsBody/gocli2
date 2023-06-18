package gocli

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

type Context struct {
	arguments []*argument

	options []*option

	helpStr string

	commandStr string

	Value interface{}
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

func getNext(idx int, args []string) (interface{}, error) {
	if len(args) == idx+1 {
		return nil, fmt.Errorf("Missing value for option '%s'.", Cyan(args[idx]))
	}

	if _, is := isOption(args[idx+1]); is {
		return nil, fmt.Errorf("Missing value for option '%s'", Cyan(args[idx]))
	}

	return args[idx+1], nil
}

func populateArgumentsAndOptions(args []string, optionsMap map[string]*option, arguments []*argument) error {
	idx := 0
	argumentIdx := 0
	for idx < len(args) {
		if arg, ok := isOption(args[idx]); ok {
			opt, exists := optionsMap[arg]
			if !exists {
				return fmt.Errorf("Unexpected option '%s'.", Cyan(args[idx]))
			}
			if opt.kind == reflect.Bool {
				opt.value = true
				opt.populated = true
			} else {
				nextValue, err := getNext(idx, args)
				if err != nil {
					return err
				}
				opt.value, err = cast(nextValue, opt.kind)
				if err != nil {
					return fmt.Errorf("Invalid value for '%s'. %s", args[idx], err.Error())
				}
				opt.populated = true
				idx++
			}
		} else {
			if argumentIdx >= len(arguments) {
				return fmt.Errorf("Unexpected argument '%s'", args[idx])
			}
			argument := arguments[argumentIdx]
			value, err := cast(args[idx], argument.kind)
			if err != nil {
				return fmt.Errorf("Invalid value for '%s'. %s", argument.name, err.Error())
			}
			argument.value = value
			argument.populated = true
			argumentIdx++
		}
		idx++
	}

	// check for missing arguments
	for _, argument := range arguments {
		if argument.required && !argument.populated {
			return fmt.Errorf("Missing or empty argument '%s'.", Yellow(argument.name))
		}
	}

	// check for missing options
	for _, option := range optionsMap {
		if option.required && !option.populated {
			return fmt.Errorf("Missing or empty option: '%s'.", Yellow(fmt.Sprintf("--%s", option.long)))
		}
	}

	return nil
}

func buildContext(c *Command, args []string, ctx *Context) {
	ctx.commandStr = c.fullName()
	optionsMap := buildOptionsMap(c)
	ctx.arguments = buildArguments(c)
	ctx.helpStr = getHelpStr(optionsMap, ctx.arguments, c)

	if _, ok := hasHelp(args); ok {
		fmt.Println(ctx.helpStr)
		os.Exit(0)
	}

	err := populateArgumentsAndOptions(args, optionsMap, ctx.arguments)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	ctx.options = optionsMapToArray(optionsMap)
}

func hasHelp(args []string) (int, bool) {
	// check for help field
	for i := len(args) - 1; i >= 0; i-- {
		arg := args[i]
		if strings.HasPrefix(arg, "-h=") || strings.HasPrefix(arg, "--help=") || arg == "-h" || arg == "--help" {
			return i, true
		}
	}

	return -1, false
}

type Bylong []*option

func (b Bylong) Len() int           { return len(b) }
func (b Bylong) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b Bylong) Less(i, j int) bool { return b[i].long < b[j].long }

type Byfirst [][]string

func (b Byfirst) Len() int           { return len(b[0]) }
func (b Byfirst) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b Byfirst) Less(i, j int) bool { return b[i][0] < b[j][0] }

func getHelpStr(optionsMap map[string]*option, arguments []*argument, c *Command) string {
	options := optionsMapToArray(optionsMap)
	sort.Sort(Bylong(options))

	padding := 5

	usage := fmt.Sprintf("Usage: %s", c.fullName())
	if len(c.node.children) > 0 {
		usage += " [" + Magenta("COMMAND") + "]"
	}

	if len(options) > 0 {
		usage += " [" + Green("OPTIONS") + "]"
	}

	if len(arguments) > 0 {
		for _, argument := range arguments {
			usage += fmt.Sprintf(" %s", Yellow(argument.name))
		}
	}

	lines := strings.Repeat("-", len(usage)) + Sep()
	txt := lines + usage + "\n" + lines
	if c.LongDesc != "" {
		txt += c.LongDesc
		txt += Sep()
	}

	txt += Sep()

	if len(c.node.children) > 0 {

		txt += "Commands:" + Sep()

		commands := [][]string{}

		maxWidth := 0
		commandNames := map[*commandNode]string{}
		for _, child := range c.node.children {
			if _, seen := commandNames[child]; !seen {
				maxWidth = max(helpNameWidth(child.value), maxWidth)
				name := helpName(child.value)
				commandNames[child] = name
			}
		}
		width := maxWidth + padding
		for child, name := range commandNames {
			commands = append(commands, []string{child.value.Name, paddedNameByLength(name, width, helpNameWidth(child.value)) + child.value.ShortDesc + Sep()})
		}
		sort.Sort(Byfirst(commands))
		for _, pair := range commands {
			txt += "  " + pair[1]
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

func helpName(c *Command) string {
	result := Magenta(c.Name)
	if c.Aliases != nil {
		aliases := make([]string, len(c.Aliases))
		copy(aliases, c.Aliases)
		for i := range aliases {
			aliases[i] = Magenta(aliases[i])
		}
		result += " (" + strings.Join(aliases, ",") + ")"
	}
	return result
}

func helpNameWidth(c *Command) int {
	result := c.Name
	if c.Aliases != nil {
		result += " (" + strings.Join(c.Aliases, ",") + ")"
	}
	return len(result)
}

func max(i, j int) int {
	if i > j {
		return i
	}

	return j
}

func paddedNameByLength(name string, width int, nameLength int) (p string) {
	p += name
	for i := 0; i < width-nameLength; i++ {
		p += " "
	}
	return p
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

func cast(v interface{}, kind reflect.Kind) (interface{}, error) {
	switch kind {
	case reflect.Bool:
		return v.(bool), nil
	case reflect.String:
		return v.(string), nil
	case reflect.Int:
		i, ok := v.(int)
		if ok {
			return i, nil
		}
		i, err := strconv.Atoi(v.(string))
		if err != nil {
			return nil, fmt.Errorf("Expected an integer, got '%s'", v)
		}
		return i, nil
	case reflect.Int8:
		i, ok := v.(int8)
		if ok {
			return i, nil
		}
		in, err := strconv.Atoi(v.(string))
		if err != nil {
			return nil, fmt.Errorf("Expected an integer, got '%s'", v)
		}
		return int8(in), nil
	case reflect.Int16:
		i, ok := v.(int16)
		if ok {
			return i, nil
		}
		in, err := strconv.Atoi(v.(string))
		if err != nil {
			return nil, fmt.Errorf("Expected an integer, got '%s'", v)
		}
		return int16(in), nil
	case reflect.Int32:
		i, ok := v.(int32)
		if ok {
			return i, nil
		}
		in, err := strconv.Atoi(v.(string))
		if err != nil {
			return nil, fmt.Errorf("Expected an integer, got '%s'", v)
		}
		return int32(in), nil
	case reflect.Int64:
		i, ok := v.(int64)
		if ok {
			return i, nil
		}
		in, err := strconv.Atoi(v.(string))
		if err != nil {
			return nil, fmt.Errorf("Expected an integer, got '%s'", v)
		}
		return int64(in), nil
	case reflect.Float32:
		f, ok := v.(float32)
		if ok {
			return f, nil
		}

		fl, err := strconv.ParseFloat(v.(string), 32)
		if err != nil {
			return nil, fmt.Errorf("Expected a float, got '%s'", v)
		}
		return float32(fl), nil
	case reflect.Float64:
		f, ok := v.(float64)
		if ok {
			return f, nil
		}

		fl, err := strconv.ParseFloat(v.(string), 32)
		if err != nil {
			return nil, fmt.Errorf("Expected a float, got '%s'", v)
		}
		return fl, nil
	default:
		return nil, fmt.Errorf("Could not parse value %+v", v)
	}
}
