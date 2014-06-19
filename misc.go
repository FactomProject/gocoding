package gocoding

import (
	"fmt"
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

/*type kindFailure struct {
	target, actual reflect.Kind
}

func mustKindCheck(target, actual reflect.Kind) {
	if target != actual {
		panic(&kindFailure{target, actual})
	}
}

func mustKindsCheck(actual reflect.Kind, targets...reflect.Kind) {
	for _, target := range targets {
		mustKindCheck(target, actual)
	}
}

func (kf *kindFailure) Error() string {
	return fmt.Sprintf("gocoding.Marshal kind failure: kind should be %q but is %q", kf.target, kf.actual)
}

type typeFailure struct {
	target, actual reflect.Type
}

func mustTypeCheck(target, actual reflect.Type) {
	if !target.AssignableTo(actual) {
		panic(&typeFailure{target, actual})
	}
}

func (tf *typeFailure) Error() string {
	return fmt.Sprintf("gocoding.Marshal type failure: type should be %q but is %q", tf.target, tf.actual)
}*/