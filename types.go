package gocoding

import (
	"io"
	"reflect"
)

type Marshaller interface {
	Marshal(interface{}) error
	FindEncoder(reflect.Type) (Encoder, error)
	CacheEncoder(t reflect.Type, encoder Encoder)
}

type Encoder func([64]byte, Renderer, reflect.Value) error

type Encoding func(Marshaller, reflect.Type) (Encoder, error)

type Renderer interface {
	FormatWriter
	WriteNil() (int, error)
	
	StartStruct() (int, error)
	StopStruct() (int, error)
	
	StartMap() (int, error)
	StopMap() (int, error)
	
	StartArray() (int, error)
	StopArray() (int, error)
	
	StartElement(id string) (int, error)
	StopElement(id string) (int, error)
}

type Scanner interface {
	
}

type FormatWriter interface {
	io.Writer
	Print(args...interface{}) (int, error)
	Printf(format string, args...interface{}) (int, error)
}