package json

import (
	"encoding/json"
	"github.com/firelizzard18/gocoding"
	"github.com/firelizzard18/gocoding/text"
	"io"
	"reflect"
)

func NewMarshaller() gocoding.Marshaller {
	return gocoding.NewMarshaller(JSONEncoding)
}

func Marshal(writer io.Writer, obj interface{}) {
	NewMarshaller().Marshal(RenderJSON(writer), obj)
}

func MarshalIndent(writer io.Writer, obj interface{}, prefix, indent string) {
	NewMarshaller().Marshal(RenderIndentedJSON(writer, prefix, indent), obj)
}

func NewUnmarshaller() gocoding.Unmarshaller {
	return gocoding.NewUnmarshaller(JSONDecoding)
}

func Unmarshal(reader gocoding.SliceableRuneReader, obj interface{}) {
	NewUnmarshaller().Unmarshal(ScanJSON(reader), obj)
}



var jsonMarshallerType = reflect.TypeOf(new(json.Marshaler)).Elem()

func JSONEncoding(marshaller gocoding.Marshaller, theType reflect.Type) gocoding.Encoder {
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

func JSONDecoding(unmarshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
	if theType.ConvertibleTo(jsonUnmarshallerType) {
		return jsonUnmarshallerDecoder
	}
	
	return text.Decoding(unmarshaller, theType)
}

func jsonUnmarshallerDecoder(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
	juvalue := value.Interface().(json.Unmarshaler)
	scanner.Continue()
	json := scanner.NextString()
	err := juvalue.UnmarshalJSON([]byte(json))
	if err != nil { scanner.Error(gocoding.ErrorPrint("JSON Marshal", err)) }
}