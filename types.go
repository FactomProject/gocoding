package gocoding

import (
	"reflect"
)

type Encoder func([64]byte, Renderer, reflect.Value) error

type Encoding func(Encoding, reflect.Type) (Encoder, error)

type Renderer interface {
	StartElement(id string) error
	StartMap() error
	
	WriteData(data []byte) error
	WriteString(str string) error
	WriteNil()
	
	StopElement(id string) error
	StopMap() error
}

type Scanner interface {
	
}

type EncoderFunc func([64]byte, Renderer, reflect.Value) error