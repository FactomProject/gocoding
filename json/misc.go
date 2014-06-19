package json

import (
	"io"
	"reflect"
	
	"encoding/json"
	
	"github.com/firelizzard18/gocoding"
)

var jsonMarshallerType = reflect.TypeOf(new(json.Marshaler)).Elem()

func NewMarshaller(writer io.Writer) gocoding.Marshaller {
	return gocoding.NewMarshaller(JSONEncoding, RenderJSON(writer))
}

func NewIndentedMarshaller(writer io.Writer, prefix, indent string) gocoding.Marshaller {
	return gocoding.NewMarshaller(JSONEncoding, RenderIndentedJSON(writer, prefix, indent))
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