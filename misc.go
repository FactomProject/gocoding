package gocoding

import (
	"fmt"
	"reflect"
)

func ErrorPrint(class string, args...interface{}) *Error {
	return &Error{class, fmt.Sprint(args...)}
}

func ErrorPrintf(class, format string, args...interface{}) *Error {
	return &Error{class, fmt.Sprintf(format, args...)}
}

func (e *Error) Error() string {
	return fmt.Sprint(e.Class, ": ", e.Value)
}

type nilValue struct{}
var NilValue = reflect.ValueOf(nilValue{})