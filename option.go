package gocli

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"unicode"
)

type Options interface{}

type option struct {
	long        string
	short       string
	required    bool
	populated   bool
	kind        reflect.Kind
	value       interface{}
	description string
}

func (o *option) getName() string {
	return fmt.Sprintf("--%s", o.long)
}

func (o *option) getKind() reflect.Kind {
	return o.kind
}

func (o *option) setValue(i interface{}) {
	o.value = i
}

func isOption(arg string) bool {
	if len(arg) <= 0 {
		return false
	}

	return arg[0] != '-'
}

func validateOptions(options interface{}) error {
	optionsValue := reflect.ValueOf(options).Elem().Elem()
	optionsType := optionsValue.Type()

	for i := 0; i < optionsValue.NumField(); i++ {
		field := optionsValue.Field(i)
		fType := optionsType.Field(i)

		if strings.ToLower(fType.Name) == "help" {
			return newSetupError("'help' is a reserved option. You cannot configure %s to use it.", optionsValue.Type())
		}

		if tag, exists := fType.Tag.Lookup("short"); exists && strings.ToLower(tag) == "h" {
			return newSetupError("'h' is a reserved option. You cannot configure %s to use it.", optionsValue.Type())
		}

		if !field.CanInterface() {
			return newSetupError("Will not be able to access the option defined by '%s.%s'. Please verify that the option is exportable (i.e. begins with a capital letter).", optionsValue.Type(), field.Type().Name())
		}

		kind := field.Type().Kind()
		if kind != reflect.Bool &&
			kind != reflect.String &&
			kind != reflect.Float64 &&
			kind != reflect.Float32 &&
			kind != reflect.Int &&
			kind != reflect.Int8 &&
			kind != reflect.Int16 &&
			kind != reflect.Int32 &&
			kind != reflect.Int64 {
			return newSetupError("Invalid type for option '%s'. Allowed types are string, bool, ints, and floats", field.Type().Name())
		}
	}

	return nil
}

func setOptions(c *Command, args []string) {
	c.Options = map[string]interface{}{}
}

func printOption(options interface{}) {
	optionsType := reflect.TypeOf(options)

	for i := 0; i < optionsType.NumField(); i++ {
		field := optionsType.Field(i)
		tag := field.Tag
		fmt.Println(field.Name, tag.Get("short"), tag.Get("required"), tag.Get("description"))
	}
}

func buildOptionsMap(c *Command) map[string]*option {
	help := &option{
		long:        "help",
		short:       "h",
		description: "Display the help text and exit",
		kind:        reflect.Bool,
		required:    false,
	}
	optionsMap := map[string]*option{"help": help, "h": help}
	if c.Options == nil {
		return optionsMap
	}

	optionsDefValue := reflect.ValueOf(c.Options)
	optionsDefType := optionsDefValue.Type()

	for idx := 0; idx < optionsDefType.NumField(); idx++ {
		opt := &option{}

		field := optionsDefType.Field(idx)
		opt.long = convertToJSONCase(field.Name)
		opt.kind = field.Type.Kind()
		if short, exists := field.Tag.Lookup("short"); exists {
			opt.short = short
			optionsMap[opt.short] = opt
		}
		if required, exists := field.Tag.Lookup("required"); exists && required == "true" {
			opt.required = true
		}
		if description, exists := field.Tag.Lookup("description"); exists {
			opt.description = description
		}

		optionsMap[opt.long] = opt
		setValue(opt, optionsDefValue.Field(idx).Interface())
	}

	return optionsMap
}

func getEqualsSides(input string) (string, string, bool) {
	re := regexp.MustCompile(`^(.*?)=(.*)$`)
	matches := re.FindStringSubmatch(input)

	if len(matches) == 3 {
		leftSide := matches[1]
		rightSide := matches[2]
		return leftSide, rightSide, true
	}

	return "", "", false
}

func populateOptionsMap(optionsMap map[string]*option, args []string) (remaining []string, err error) {
	defer func() {
		for _, option := range optionsMap {
			if option.required && !option.populated {
				err = newError("Missing or empty option: '%s'.", Yellow(fmt.Sprintf("--%s", option.long)))
				return
			}
		}
	}()

	temp := []string{}
	for _, arg := range args {
		if left, right, hasEquals := getEqualsSides(arg); hasEquals {
			temp = append(temp, left, right)
		} else {
			temp = append(temp, arg)
		}
	}
	args = temp

	var prev *option
	for i, arg := range args {
		if isShortFlag(arg) {
			current, exists := optionsMap[arg[1:]]
			if !exists {
				err = newError("Invalid option: '%s'", Yellow(fmt.Sprintf("-%s", arg[1:])))
				return
			}
			if current.kind == reflect.Bool {
				current.value = true
				current.populated = true
				prev = nil
			} else {
				prev = current
			}

		} else if isLongFlag(arg) {
			current, exists := optionsMap[arg[2:]]
			if !exists {
				err = newError("Invalid option: '%s'", Yellow(fmt.Sprintf("--%s", arg[2:])))
				return
			}
			if current.kind == reflect.Bool {
				current.value = true
				current.populated = true
				prev = nil
			} else {
				prev = current
			}

		} else { // is not an option flag, must be an argument.

			// if there is no previous option, then return the remaining arguments
			if prev == nil {
				remaining = args[i:]
				return
			}

			// the current arg must be a value for the previous option flag
			setValue(prev, arg)
			prev.populated = true
			prev = nil
		}
	}

	remaining = []string{}

	return
}

func optionsMapToArray(optionsMap map[string]*option) []*option {
	optionSet := map[*option]bool{}
	for _, option := range optionsMap {
		optionSet[option] = true
	}

	options := make([]*option, len(optionSet))
	i := 0
	for option := range optionSet {
		options[i] = option
		i++
	}
	return options
}

func isFloatKind(k reflect.Kind) bool {
	return k == reflect.Float32 || k == reflect.Float64
}

func isIntKind(k reflect.Kind) bool {
	return k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 || k == reflect.Int32 || k == reflect.Int64
}

// return true if a short flag, e.g. "-g" but not "--global"
func isShortFlag(s string) bool {
	return strings.HasPrefix(s, "-") && len(s) > 1 && s[1] != '-'
}

// return true if a long flag, e.g. "--global" but not "-g"
func isLongFlag(s string) bool {
	return strings.HasPrefix(s, "--") && len(s) > 2 && s[2] != '-'
}

func convertToJSONCase(input string) string {
	var output bytes.Buffer

	for i, r := range input {
		if unicode.IsUpper(r) {
			if i > 0 {
				output.WriteByte('-')
			}
			output.WriteRune(unicode.ToLower(r))
		} else {
			output.WriteRune(r)
		}
	}

	return output.String()
}
func convertToCamelCase(jsonCase string) string {
	// Split the string by dashes
	words := strings.Split(jsonCase, "-")

	// Capitalize the first letter of each word (including the first word)
	for i := 0; i < len(words); i++ {
		words[i] = strings.Title(words[i])
	}

	// Join the words and return the result
	return strings.Join(words, "")
}
