package json

import (
	"encoding/json"
	"github.com/firelizzard18/gocoding"
	"github.com/firelizzard18/gocoding/text"
	"io"
	"reflect"
)

func NewMarshaller() gocoding.Marshaller {
	return gocoding.NewMarshaller(Encoding)
}

func Marshal(writer io.Writer, obj interface{}) error {
	return NewMarshaller().Marshal(Render(writer), obj)
}

func MarshalIndent(writer io.Writer, obj interface{}, prefix, indent string) error {
	return NewMarshaller().Marshal(RenderIndented(writer, prefix, indent), obj)
}

func NewUnmarshaller() gocoding.Unmarshaller {
	return gocoding.NewUnmarshaller(Decoding)
}

func Unmarshal(reader gocoding.SliceableRuneReader, obj interface{}) error {
	return NewUnmarshaller().Unmarshal(Scan(reader), obj)
}



var jsonMarshallerType = reflect.TypeOf(new(json.Marshaler)).Elem()

func Encoding(marshaller gocoding.Marshaller, theType reflect.Type) gocoding.Encoder {
	if theType.ConvertibleTo(jsonMarshallerType) {
		return jsonMarshallerEncoder
	}
	
	return text.Encoding(marshaller, theType)
}

func jsonMarshallerEncoder(scratch [64]byte, renderer gocoding.Renderer, value reflect.Value) {
	jmvalue := value.Interface().(json.Marshaler)
	json, err := jmvalue.MarshalJSON()
	if err != nil { renderer.Error(gocoding.ErrorPrint("JSON Marshal", err)) }
	renderer.Write(json)
}



var jsonUnmarshallerType = reflect.TypeOf(new(json.Unmarshaler)).Elem()

func Decoding(unmarshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
	if theType.ConvertibleTo(jsonUnmarshallerType) {
		return jsonUnmarshallerDecoder
	}
	
	decoder := text.Decoding(unmarshaller, theType)
	
	if reflect.PtrTo(theType).ConvertibleTo(jsonUnmarshallerType) {
		return gocoding.TryIndirectDecoding(decoder, jsonUnmarshallerDecoder)
	}
	
	return decoder
}

func jsonUnmarshallerDecoder(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
	juvalue := value.Interface().(json.Unmarshaler)
	json := scanner.NextString()
	err := juvalue.UnmarshalJSON([]byte(json))
	if err != nil { scanner.Error(gocoding.ErrorPrint("JSON Marshal", err)) }
}