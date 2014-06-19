package gocoding

import (
	"io"
	"reflect"
)

type Marshaller interface {
	SetRenderer(renderer Renderer)
	Marshal(interface{}) error
	MarshalObject(obj interface{})
	MarshalValue(value reflect.Value)
	FindEncoder(reflect.Type) Encoder
	CacheEncoder(t reflect.Type, encoder Encoder)
}

type Encoder func([64]byte, Renderer, reflect.Value)

type Encoding func(Marshaller, reflect.Type) Encoder

type Encodable1 interface {
	Encoding(Marshaller, reflect.Type) Encoder
}

type Encodable2 interface {
	EncodableFields() map[string]reflect.Value
}

type Renderer interface {
	io.Writer
	Print(args...interface{}) int
	Printf(format string, args...interface{}) int
	WriteNil() int
	
	StartStruct() int
	StopStruct() int
	
	StartMap() int
	StopMap() int
	
	StartArray() int
	StopArray() int
	
	StartElement(id string) int
	StopElement(id string) int
	
	Error(*Error)
	Recover(interface{}) error
}

type Scanner interface {
	
}

type Error struct {
	Class string
	Value interface{}
}