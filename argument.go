package gocli

import (
	"reflect"
)

type valued interface {
	getName() string
	getKind() reflect.Kind
	setValue(interface{})
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

		arg.value, _ = cast(argumentsDefValue.Field(idx).Interface(), arg.kind)
		arguments[idx] = arg
	}
	return arguments
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
