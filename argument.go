package gocli

import (
	"reflect"
	"strconv"
	"strings"
)

type valued interface {
	getName() string
	getKind() reflect.Kind
	setValue(interface{})
}

func setValue(v valued, value interface{}) {
	kind := v.getKind()

	var err error
	switch kind {
	case reflect.Bool:
		v.setValue(value.(bool))
	case reflect.String:
		v.setValue(value)
	case reflect.Int:
		i, ok := value.(int)
		if !ok {
			i, err = strconv.Atoi(value.(string))
			if err != nil {
				fatal(newError("Invalid integer value '%s'='%s'", Yellow(v.getName()), value.(string)))
			}
		}
		v.setValue(i)
	case reflect.Int8:
		i, ok := value.(int8)
		if !ok {
			in, err := strconv.Atoi(value.(string))
			if err != nil {
				fatal(newError("Invalid integer value '%s'='%s'", Yellow(v.getName()), value.(string)))
			}
			i = int8(in)
		}
		v.setValue(i)
	case reflect.Int16:
		i, ok := value.(int16)
		if !ok {
			in, err := strconv.Atoi(value.(string))
			if err != nil {
				fatal(newError("Invalid integer value '%s'='%s'", Yellow(v.getName()), value.(string)))
			}
			i = int16(in)
		}
		v.setValue(i)
	case reflect.Int32:
		i, ok := value.(int32)
		if !ok {
			in, err := strconv.Atoi(value.(string))
			if err != nil {
				fatal(newError("Invalid integer value '%s'='%s'", Yellow(v.getName()), value.(string)))
			}
			i = int32(in)
		}
		v.setValue(i)
	case reflect.Int64:
		i, ok := value.(int64)
		if !ok {
			in, err := strconv.Atoi(value.(string))
			if err != nil {
				fatal(newError("Invalid integer value '%s'='%s'", Yellow(v.getName()), value.(string)))
			}
			i = int64(in)
		}
		v.setValue(i)
	case reflect.Float32:
		f, ok := value.(float32)
		if !ok {
			fl, err := strconv.ParseFloat(value.(string), 32)
			if err != nil {
				fatal(newError("Invalid float64 value '%s'='%s'", Yellow(v.getName()), value.(string)))
			}
			f = float32(fl)
		}
		v.setValue(f)
	case reflect.Float64:
		f, ok := value.(float64)
		if !ok {
			f, err = strconv.ParseFloat(value.(string), 32)
			if err != nil {
				fatal(newError("Invalid float64 value '%s'='%s'", Yellow(v.getName()), value.(string)))
			}
		}
		v.setValue(f)
	}
}

type argument struct {
	name        string
	value       interface{}
	populated   bool
	required    bool
	kind        reflect.Kind
	description string
}

func (a *argument) getName() string {
	return a.name
}

func (a *argument) getKind() reflect.Kind {
	return a.kind
}

func (a *argument) setValue(i interface{}) {
	a.value = i
}

func buildArguments(c *Command) []*argument {
	if c.Arguments == nil {
		return []*argument{}
	}

	argumentsDefValue := reflect.ValueOf(c.Arguments)
	argumentsDefType := argumentsDefValue.Type()
	arguments := make([]*argument, argumentsDefType.NumField())

	for idx := 0; idx < argumentsDefType.NumField(); idx++ {
		arg := &argument{}

		field := argumentsDefType.Field(idx)
		arg.name = convertToJSONCase(field.Name)
		arg.kind = field.Type.Kind()
		if required, exists := field.Tag.Lookup("required"); exists && required == "true" {
			arg.required = true
		}
		if description, exists := field.Tag.Lookup("description"); exists {
			arg.description = description
		}

		setValue(arg, argumentsDefValue.Field(idx).Interface())
		arguments[idx] = arg
	}
	return arguments
}

func populateArguments(arguments []*argument, args []string) (err error) {
	defer func() {
		if err != nil {
			return
		}

		for _, argument := range arguments {
			if argument.required && !argument.populated {
				err = newError("Missing or empty argument '%s'.", Yellow(argument.name))
				return
			}
		}
	}()

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			err = newError("Invalid option '%s'", Yellow(arg))
			return
		}
	}

	if len(arguments) == 0 {
		if len(args) > 0 {
			err = newError("Invalid argument(s): '%s'", strings.Join(args, " "))
		}
		return
	}

	if len(args) > len(arguments) {
		temp := make([]string, len(arguments))
		copy(temp[0:len(temp)-1], args)
		temp[len(temp)-1] = strings.Join(args[len(temp)-1:], " ")
		args = temp
	}

	for i, arg := range args {
		argument := arguments[i]
		argument.populated = true
		argument.value = arg
	}

	return
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
