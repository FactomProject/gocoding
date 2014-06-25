package json

import (
	"io"
	"reflect"
	
	"encoding/json"
	
	"github.com/firelizzard18/gocoding"
)

var jsonMarshallerType = reflect.TypeOf(new(json.Marshaler)).Elem()

func NewMarshaller() gocoding.Marshaller {
	return gocoding.NewMarshaller(JSONEncoding)
}

func Marshal(writer io.Writer, obj interface{}) {
	NewMarshaller().Marshal(RenderJSON(writer), obj)
}

func MarshalIndent(writer io.Writer, obj interface{}, prefix, indent string) {
	NewMarshaller().Marshal(RenderIndentedJSON(writer, prefix, indent), obj)
}

func JSONEncoding(marshaller gocoding.Marshaller, theType reflect.Type) gocoding.Encoder {
	if theType.ConvertibleTo(jsonMarshallerType) {
		return jsonMarshallerEncoder
	}
	
	return gocoding.TextEncoding(marshaller, theType)
}

func jsonMarshallerEncoder(scratch [64]byte, renderer gocoding.Renderer, value reflect.Value) {
	jmvalue := value.Interface().(json.Marshaler)
	json, err := jmvalue.MarshalJSON()
	if err != nil { renderer.Error(gocoding.ErrorPrint("JSON Marshal", err)) }
	renderer.Write(json)
}