package gocoding

import (
	"io"
	"reflect"
)

type Error struct {
	Class string
	Value interface{}
}

type Marshaller interface {
	Marshal(Renderer, interface{}) error
	MarshalObject(Renderer, interface{})
	MarshalValue(Renderer, reflect.Value)
	FindEncoder(reflect.Type) Encoder
	IsCached(reflect.Type) bool
	CacheEncoder(reflect.Type, Encoder)
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

type Unmarshaller interface {
	Unmarshal(Scanner, interface{}) error
	UnmarshalObject(Scanner, interface{})
	UnmarshalValue(Scanner, reflect.Value)
	FindDecoder(reflect.Type) Decoder
	CacheDecoder(reflect.Type, Decoder)
}

type Decoder func([64]byte, Scanner, reflect.Value)
type Decoding func(Unmarshaller, reflect.Type) Decoder

type Scanner interface {
	NextCode() ScannerCode
	NextValue() reflect.Value
	Continue() ScannerCode
	
	Error(*Error)
}

type ScannerCode uint8

type RuneReader interface {
	Next() rune
	Peek() rune
	Backup() rune
	Done() bool
	String() string // optional
}

type SliceableRuneReader interface {
	RuneReader
	Mark()
	Slice() SliceableRuneReader
}
