package gocoding

import (
	"io"
	"reflect"
)

type Error struct {
	Class string
	Value interface{}
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

type Marshaller interface {
//	SetRenderer(renderer Renderer)
	Marshal(Renderer, interface{}) error
	MarshalObject(Renderer, interface{})
	MarshalValue(Renderer, reflect.Value)
	FindEncoder(reflect.Type) Encoder
	CacheEncoder(reflect.Type, Encoder)
}

type RuneReader interface {
	Read() rune
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

const EndOfText rune = '\u0003'

type ScannerCode uint8

const (
	Scanning ScannerCode = iota
	
	ScannedKeyBegin
	ScannedKeyEnd
	
	ScannedLiteralBegin
	ScannedLiteralEnd
	
	ScannedStructBegin
	ScannedStructEnd
	
	ScannedMapBegin
	ScannedMapEnd
	
	ScannedArrayBegin
	ScannedArrayEnd
	
	ScannerInitialized
	ScannedToEnd
	
	ScannerError
	ScannerBadCode
)

func (sc ScannerCode) ScannedBegin() bool {
	switch sc {
	case ScannedKeyBegin, ScannedLiteralBegin, ScannedStructBegin, ScannedMapBegin, ScannedArrayBegin:
		return true
		
	default:
		return false
	}
}

func (sc ScannerCode) ScannedEnd() bool {
	switch sc {
	case ScannedKeyEnd, ScannedLiteralEnd, ScannedStructEnd, ScannedMapEnd, ScannedArrayEnd:
		return true
		
	default:
		return false
	}
}

func (sc ScannerCode) Reflection() ScannerCode {
	if sc.ScannedBegin() {
		return ScannerCode(sc + 1)
	}
	
	if sc.ScannedEnd() {
		return ScannerCode(sc - 1)
	}
	
	return ScannerBadCode
}

type Scanner interface {
	NextCode() ScannerCode
	NextValue() reflect.Value
	
	Error(*Error)
}