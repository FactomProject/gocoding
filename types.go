package gocoding

import (
	"io"
	"reflect"
)

type Error struct {
	Class string
	Value interface{}
}

type Errorable interface {
	Error(*Error)
	SetErrorHandler(func(*Error))
	Recover(interface{}) error
	SetRecoverHandler(func(interface{}) error)
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
	Errorable
	io.Writer

	Print(args ...interface{}) int
	Printf(format string, args ...interface{}) int
	WriteNil() int
	PrintString(string) int

	StartStruct() int
	StopStruct() int

	StartMap() int
	StopMap() int

	StartArray() int
	StopArray() int

	StartElement(id string) int
	StopElement(id string) int
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

type Decodable1 interface {
	Decoding(Unmarshaller, reflect.Type) Decoder
}

type Decodable2 interface {
	DecodableFields() map[string]reflect.Value
}

type Scanner interface {
	Errorable

	// get the code on the top of the stack
	Peek() ScannerCode

	// scan the next rune, returning the appropriate code
	//   this will step through the scanner's state
	NextCode() ScannerCode

	// continue scanning until NextValue() doesn't return Scanning
	Continue() ScannerCode

	// scan the next value
	//   this will scan the next complete value
	NextValue() reflect.Value

	// scan the next value as a string
	//   this will scan the next complete value, returning the data unparsed
	NextString() string
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
