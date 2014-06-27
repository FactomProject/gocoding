package gocoding

import (
	"fmt"
	"reflect"
	"runtime"
)

// turn everything into a pointer value
//   this makes unmarshalling much simpler
//   this will return a pointer to a non-pointer
//   if value is a pointer to a non-pointer, it will be returned
//   if value is a non-pointer, a pointer to it will be returned
//   else, Normalize will dereference value and try again
func NormalizeValue(e Errorable, class string, value reflect.Value) reflect.Value {
	if value.Kind() != reflect.Ptr {
//		if value.Type().Name() == "" { panic("Don't know what to do") }
		if !value.CanAddr() { e.Error(ErrorPrint(class, "Normalization failed: value is not addressable: ", value)) }
		value = value.Addr()
	}
	
	for {
		if value.IsNil() {
			value.Set(reflect.New(value.Type().Elem()))
		}
		
		if value.Elem().Kind() != reflect.Ptr {
			return value
		}
		
		value = value.Elem()
	}
}

func NormalizeType(theType reflect.Type) reflect.Type {
	if theType.Kind() != reflect.Ptr {
		return reflect.PtrTo(theType)
	}
	
	for theType.Elem().Kind() == reflect.Ptr {
		theType = theType.Elem()
	}
	
	return theType
}

func ErrorPrint(class string, args...interface{}) *Error {
	return &Error{class, fmt.Sprint(args...)}
}

func ErrorPrintf(class, format string, args...interface{}) *Error {
	return &Error{class, fmt.Sprintf(format, args...)}
}

func (e *Error) Error() string {
	return fmt.Sprint(e.Class, ": ", e.Value)
}

type BasicErrorable struct {
	handler func(*Error)
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
		s.recovery(err)
	}
	
	return nil
}

func (s *BasicErrorable) SetRecoverHandler(handler func(interface{}) error) {
	s.recovery = handler
}