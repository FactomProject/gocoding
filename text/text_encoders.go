package text

import (
	"encoding"
	"encoding/base64"
	"github.com/firelizzard18/gocoding"
	"math"
	"reflect"
	"strconv"
)

var encodableType1 = reflect.TypeOf(new(gocoding.Encodable1)).Elem()
var encodableType2 = reflect.TypeOf(new(gocoding.Encodable2)).Elem()
var textMarshallerType = reflect.TypeOf(new(encoding.TextMarshaler)).Elem()

func Encoding(marshaller gocoding.Marshaller, theType reflect.Type) gocoding.Encoder {
	if theType.ConvertibleTo(encodableType1) {
		return Encodable1Encoding(marshaller, theType)
	}
	
	if theType.ConvertibleTo(encodableType2) {
		return Encodable2Encoding(marshaller, theType)
	}
	
	if theType.ConvertibleTo(textMarshallerType) {
		return textMarshallerEncoder
	}
	
	var encoder gocoding.Encoder
	switch theType.Kind() {
	case reflect.Bool:
		encoder = boolEncoder
	
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		encoder = intEncoder
	
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		encoder = uintEncoder
		
	case reflect.Float32:
		encoder = float32Encoder
		
	case reflect.Float64:
		encoder = float64Encoder
		
	case reflect.String:
		encoder = stringEncoder
		
	case reflect.Interface:
		encoder = InterfaceEncoding(marshaller, theType)
	
	case reflect.Struct:
		encoder = StructEncoding(marshaller, theType)
		
	case reflect.Map:
		encoder = MapEncoding(marshaller, theType)
		
	case reflect.Slice:
		encoder = SliceEncoding(marshaller, theType)
		
	case reflect.Array:
		encoder = ArrayEncoding(marshaller, theType)
		
	case reflect.Ptr:
		encoder = PtrEncoding(marshaller, theType)
	
	default:
		encoder = errorEncoding(gocoding.ErrorPrint("Encoding", "Unsupported type: ", theType))
	}
	
	if theType.Kind() == reflect.Ptr {
		return encoder
	}
	
	ptrType := reflect.PtrTo(theType)
	if ptrType.ConvertibleTo(encodableType1) ||
	   ptrType.ConvertibleTo(encodableType2) ||
	   ptrType.ConvertibleTo(textMarshallerType) {
		return tryIndirectEncoder(encoder, marshaller.FindEncoder(ptrType))
	} else {
		return encoder
	}
}

func errorEncoding(err *gocoding.Error) gocoding.Encoder {
	return func(scratch [64]byte, renderer gocoding.Renderer, value reflect.Value) {
		renderer.Error(err)
	}
}

func Encodable1Encoding(marshaller gocoding.Marshaller, theType reflect.Type) gocoding.Encoder {
	return func(scratch [64]byte, renderer gocoding.Renderer, value reflect.Value) {
		if value.IsNil() {
			renderer.WriteNil()
		} else {
			encoding := value.Interface().(gocoding.Encodable1).Encoding(marshaller, theType)
			if encoding == nil { return }
			encoding(scratch, renderer, value)
		}
	}
}

func Encodable2Encoding(marshaller gocoding.Marshaller, theType reflect.Type) gocoding.Encoder {
	return func(scratch [64]byte, renderer gocoding.Renderer, value reflect.Value) {
		if value.IsNil() {
			renderer.WriteNil()
		} else {
			renderer.StartStruct()
			for name, value := range value.Interface().(gocoding.Encodable2).EncodableFields() {
				renderer.StartElement(name)
				marshaller.MarshalValue(renderer, value)
				renderer.StopElement(name)
			}
			renderer.StopStruct()
		}
	}
}

func textMarshallerEncoder(scratch [64]byte, renderer gocoding.Renderer, value reflect.Value) {
	tmvalue := value.Interface().(encoding.TextMarshaler)
	text, err := tmvalue.MarshalText()
	if err != nil { renderer.Error(gocoding.ErrorPrint("Text Marshal", err)) }
	renderer.Write(text)
}

func tryIndirectEncoder(typEncoder, ptrEncoder gocoding.Encoder) gocoding.Encoder {
	return func(scratch [64]byte, renderer gocoding.Renderer, value reflect.Value) {
		if value.CanAddr() {
			ptrEncoder(scratch, renderer, value.Addr())
		} else {
			typEncoder(scratch, renderer, value)
		}
	}
}

func boolEncoder(scratch [64]byte, renderer gocoding.Renderer, value reflect.Value) {
	if value.Bool() {
		renderer.Print("true")
	} else {
		renderer.Print("false")
	}
}
	
func intEncoder(scratch [64]byte, renderer gocoding.Renderer, value reflect.Value) {
	renderer.Write(strconv.AppendInt(scratch[:0], value.Int(), 10))
}
	
func uintEncoder(scratch [64]byte, renderer gocoding.Renderer, value reflect.Value) {
	renderer.Write(strconv.AppendUint(scratch[:0], value.Uint(), 10))
}

type floatEncoder int

func (bits floatEncoder) encode(scratch [64]byte, renderer gocoding.Renderer, value reflect.Value) {
	f := value.Float()
	if math.IsInf(f, 0) || math.IsNaN(f) {
		renderer.Error(gocoding.ErrorPrint("Encoder", "Unsupported float value: ", strconv.FormatFloat(f, 'g', -1, int(bits))))
	} else {
		renderer.Write(strconv.AppendFloat(scratch[:0], f, 'g', -1, int(bits)))
	}
}

var (
	float32Encoder = (floatEncoder(32)).encode
	float64Encoder = (floatEncoder(64)).encode
)

func stringEncoder(scratch [64]byte, renderer gocoding.Renderer, value reflect.Value) {
	renderer.Printf(`"%s"`, value.String())
}

func InterfaceEncoding(marshaller gocoding.Marshaller, theType reflect.Type) gocoding.Encoder {
	return func (scratch [64]byte, renderer gocoding.Renderer, value reflect.Value) {
		if value.IsNil() {
			renderer.WriteNil()
			return
		}
		
		marshaller.MarshalValue(renderer, value.Elem())
	}
}

func StructEncoding(marshaller gocoding.Marshaller, theType reflect.Type) gocoding.Encoder {
	encoders := make(map[string]gocoding.Encoder)
	
	current := []reflect.Type{}
	next := []reflect.Type{theType}
	
	for len(next) > 0 {
		current, next = next, current[:0]
		
		for _, aType := range current {
			for i := 0; i < aType.NumField(); i++ {
				sf := aType.Field(i)
				
				// skip masked fields
				if _, ok := encoders[sf.Name]; ok {
					continue
				}
				
				// skip unexported fields
				if sf.PkgPath != "" {
					continue
				}
				
				// add anonymous fields & skip
				if sf.Anonymous && sf.Type.Kind() == reflect.Struct {
					next = append(next, sf.Type)
					continue
				}
				
				encoders[sf.Name] = marshaller.FindEncoder(sf.Type)
			}
		}
	}
	
	return func(scratch [64]byte, renderer gocoding.Renderer, value reflect.Value) {
		renderer.StartStruct()
		
		for name, encoder := range encoders {
			renderer.StartElement(name)
			encoder(scratch, renderer, value.FieldByName(name))
			renderer.StopElement(name)
		}
		
		renderer.StopStruct()
	}
}

func MapEncoding(marshaller gocoding.Marshaller, theType reflect.Type) gocoding.Encoder {
	if theType.Key().Kind() != reflect.String {
		return errorEncoding(gocoding.ErrorPrint("Encoding", "Unsupported map key type: ", theType.Key()))
	}
	
	encoder := marshaller.FindEncoder(theType.Elem())
	if encoder == nil { return nil }
	
	return func(scratch [64]byte, renderer gocoding.Renderer, value reflect.Value) {
		if value.IsNil() {
			renderer.WriteNil()
			return
		}
		
		renderer.StartMap()
		
		for _, key := range value.MapKeys() {
			renderer.StartElement(key.String())
			encoder(scratch, renderer, value.MapIndex(key)) 
			renderer.StopElement(key.String())
		}
		
		renderer.StopMap()
	}
}

func SliceEncoding(marshaller gocoding.Marshaller, theType reflect.Type) gocoding.Encoder {
	if theType.Elem().Kind() == reflect.Uint8 {
		return byteSliceEncoder
	}
	
	encoder := ArrayEncoding(marshaller, theType)
	if encoder == nil { return nil }
	
	return func(scratch [64]byte, renderer gocoding.Renderer, value reflect.Value) {
		if value.IsNil() {
			renderer.WriteNil()
		} else {
			encoder(scratch, renderer, value)
		}
	}
}

func byteSliceEncoder(scratch [64]byte, renderer gocoding.Renderer, value reflect.Value) {
	if value.IsNil() {
		renderer.WriteNil()
		return
	}
	
	bytes := value.Bytes()
	count := len(bytes)
	
	renderer.Print(`"`)
	
	// based on http://golang.org/src/pkg/encoding/json/encode.go
	if count < 1024 {
		// for small buffers, using Encode directly is much faster.
		dst := make([]byte, base64.StdEncoding.EncodedLen(count))
		base64.StdEncoding.Encode(dst, bytes)
		
		renderer.Write(dst)
	} else {
		// for large buffers, avoid unnecessary extra temporary
		// buffer space.
		enc := base64.NewEncoder(base64.StdEncoding, renderer)
		
		_, err := enc.Write(bytes)
		if err != nil { renderer.Error(gocoding.ErrorPrint("Encoder", err)) }
		
		err = enc.Close()
		if err != nil { renderer.Error(gocoding.ErrorPrint("Encoder", err)) }
	}
	
	renderer.Print(`"`)
}

func ArrayEncoding(marshaller gocoding.Marshaller, theType reflect.Type) gocoding.Encoder {
	encoder := marshaller.FindEncoder(theType.Elem())
	if encoder == nil { return nil }
	
	return func(scratch [64]byte, renderer gocoding.Renderer, value reflect.Value) {
		count := value.Len()
		
		renderer.StartArray()
		
		for i := 0; i < count; i++ {
			id := strconv.Itoa(i)
			renderer.StartElement(id)
			encoder(scratch, renderer, value.Index(i))
			renderer.StopElement(id)
		}
		
		renderer.StopArray()
	}
}

func PtrEncoding(marshaller gocoding.Marshaller, theType reflect.Type) gocoding.Encoder {
	encoder := marshaller.FindEncoder(theType.Elem())
	if encoder == nil { return nil }
	
	return func(scratch [64]byte, renderer gocoding.Renderer, value reflect.Value) {
		if value.IsNil() {
			renderer.WriteNil()
		} else {
			encoder(scratch, renderer, value.Elem())
		}
	}
}