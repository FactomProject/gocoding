package gocoding

import (
	"fmt"
	"runtime"
)

func ErrorPrint(class string, args ...interface{}) *Error {
	return &Error{class, fmt.Sprint(args...)}
}

func ErrorPrintf(class, format string, args ...interface{}) *Error {
	return &Error{class, fmt.Sprintf(format, args...)}
}

func (e *Error) Error() string {
	return fmt.Sprint(e.Class, ": ", e.Value)
}

type BasicErrorable struct {
	handler  func(*Error)
	recovery func(interface{}) error
}

func (s *BasicErrorable) Error(err *Error) {
	if s.handler == nil {
		panic(fmt.Sprint(err.Class, ": ", err.Value))
	} else {
		s.handler(err)
	}
}

func (s *BasicErrorable) SetErrorHandler(handler func(*Error)) {
	s.handler = handler
}

func (s *BasicErrorable) Recover(err interface{}) error {
	if s.recovery == nil {
		switch err.(type) {
		case runtime.Error, string:
			panic(err)

		case error:
			return err.(error)

		default:
			panic(err)
		}
	} else {
		return s.recovery(err)
	}
}

func (s *BasicErrorable) SetRecoverHandler(handler func(interface{}) error) {
	s.recovery = handler
}
