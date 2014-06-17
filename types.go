package gocoding

import (
	"reflect"
)

type Encoder func([64]byte, Renderer, reflect.Value) error

type Encoding func(Encoding, reflect.Type) (Encoder, error)

type Renderer interface {
	StartElement(id string) error
	StartMap() error
	StartArray() error
	
	WriteData(data []byte) error
	WriteString(str string) error
	WriteNil() error
	
	StopElement(id string) error
	StopMap() error
	StopArray() error
}

type Scanner interface {
	
}

type EncoderFunc func([64]byte, Renderer, reflect.Value) error