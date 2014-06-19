package gocoding

import (
	"math"
	"reflect"
	"strconv"
	
	"encoding"
	"encoding/base64"
)

var encodableType1 = reflect.TypeOf(new(Encodable1)).Elem()
var encodableType2 = reflect.TypeOf(new(Encodable2)).Elem()
var textMarshallerType = reflect.TypeOf(new(encoding.TextMarshaler)).Elem()

func TextEncoding(marshaller Marshaller, theType reflect.Type) Encoder {
	if theType.ConvertibleTo(encodableType1) {
		return Encodable1TextEncoding(marshaller, theType)
	}
	
	if theType.ConvertibleTo(encodableType2) {
		return Encodable2TextEncoding(marshaller, theType)
	}
	
	switch theType.Kind() {
	case reflect.Bool:
		return boolTextEncoder
	
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return intTextEncoder
	
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return uintTextEncoder
		
	case reflect.Float32:
		return float32TextEncoder
		
	case reflect.Float64:
		return float64TextEncoder
		
	case reflect.String:
		return stringTextEncoder
		
	case reflect.Interface:
		return InterfaceTextEncoding(marshaller, theType)
	
	case reflect.Struct:
		return StructTextEncoding(marshaller, theType)
		
	case reflect.Map:
		return MapTextEncoding(marshaller, theType)
		
	case reflect.Slice:
		return SliceTextEncoding(marshaller, theType)
		
	case reflect.Array:
		return ArrayTextEncoding(marshaller, theType)
		
	case reflect.Ptr:
		return PtrTextEncoding(marshaller, theType)
	
	default:
		return errorTextEncoding(ErrorPrint("Encoding", "Unsupported type: ", theType))
	}
}

func Encodable1TextEncoding(marshaller Marshaller, theType reflect.Type) Encoder {
	return func(scratch [64]byte, renderer Renderer, value reflect.Value) {
		if value.IsNil() {
			renderer.WriteNil()
		} else {
			encoding := value.Interface().(Encodable1).Encoding(marshaller, theType)
			if encoding == nil { return }
			encoding(scratch, renderer, value)
		}
	}
}

func Encodable2TextEncoding(marshaller Marshaller, theType reflect.Type) Encoder {
	return func(scratch [64]byte, renderer Renderer, value reflect.Value) {
		if value.IsNil() {
			renderer.WriteNil()
		} else {
			renderer.StartStruct()
			for name, value := range value.Interface().(Encodable2).EncodableFields() {
				renderer.StartElement(name)
				marshaller.MarshalValue(value)
				renderer.StopElement(name)
			}
			renderer.StopStruct()
		}
	}
}

func textMarshallerEncoder(scratch [64]byte, renderer Renderer, value reflect.Value) {
	tmvalue := value.Interface().(encoding.TextMarshaler)
	text, err := tmvalue.MarshalText()
	if err != nil { renderer.Error(ErrorPrint("Text Marshal", err)) }
	renderer.Write(text)
}

func boolTextEncoder(scratch [64]byte, renderer Renderer, value reflect.Value) {
	if value.Bool() {
		renderer.Print("true")
	} else {
		renderer.Print("false")
	}
}
	
func intTextEncoder(scratch [64]byte, renderer Renderer, value reflect.Value) {
	renderer.Write(strconv.AppendInt(scratch[:0], value.Int(), 10))
}
	
func uintTextEncoder(scratch [64]byte, renderer Renderer, value reflect.Value) {
	renderer.Write(strconv.AppendUint(scratch[:0], value.Uint(), 10))
}

type floatTextEncoder int

func (bits floatTextEncoder) encode(scratch [64]byte, renderer Renderer, value reflect.Value) {
	f := value.Float()
	if math.IsInf(f, 0) || math.IsNaN(f) {
		renderer.Error(ErrorPrint("Encoder", "Unsupported float value: ", strconv.FormatFloat(f, 'g', -1, int(bits))))
	} else {
		renderer.Write(strconv.AppendFloat(scratch[:0], f, 'g', -1, int(bits)))
	}
}

var (
	float32TextEncoder = (floatTextEncoder(32)).encode
	float64TextEncoder = (floatTextEncoder(64)).encode
)

func stringTextEncoder(scratch [64]byte, renderer Renderer, value reflect.Value) {
	renderer.Printf(`"%s"`, value.String())
}

func InterfaceTextEncoding(marshaller Marshaller, theType reflect.Type) Encoder {
	return func (scratch [64]byte, renderer Renderer, value reflect.Value) {
		if value.IsNil() {
			renderer.WriteNil()
			return
		}
		
		marshaller.MarshalValue(value.Elem())
	}
}

func StructTextEncoding(marshaller Marshaller, theType reflect.Type) Encoder {
	encoders := make(map[string]Encoder)
	
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
	
	return func(scratch [64]byte, renderer Renderer, value reflect.Value) {
		renderer.StartStruct()
		
		for name, encoder := range encoders {
			renderer.StartElement(name)
			encoder(scratch, renderer, value.FieldByName(name))
			renderer.StopElement(name)
		}
		
		renderer.StopStruct()
	}
}

func MapTextEncoding(marshaller Marshaller, theType reflect.Type) Encoder {
	if theType.Key().Kind() != reflect.String {
		return errorTextEncoding(ErrorPrint("Encoding", "Unsupported map key type: ", theType.Key()))
	}
	
	encoder := marshaller.FindEncoder(theType.Elem())
	if encoder == nil { return nil }
	
	return func(scratch [64]byte, renderer Renderer, value reflect.Value) {
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

func SliceTextEncoding(marshaller Marshaller, theType reflect.Type) Encoder {
	if theType.Elem().Kind() == reflect.Uint8 {
		return byteSliceEncoder
	}
	
	encoder := ArrayTextEncoding(marshaller, theType)
	if encoder == nil { return nil }
	
	return func(scratch [64]byte, renderer Renderer, value reflect.Value) {
		if value.IsNil() {
			renderer.WriteNil()
		} else {
			encoder(scratch, renderer, value)
		}
	}
}

func byteSliceEncoder(scratch [64]byte, renderer Renderer, value reflect.Value) {
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
		if err != nil { renderer.Error(ErrorPrint("Encoder", err)) }
		
		err = enc.Close()
		if err != nil { renderer.Error(ErrorPrint("Encoder", err)) }
	}
	
	renderer.Print(`"`)
}

func ArrayTextEncoding(marshaller Marshaller, theType reflect.Type) Encoder {
	encoder := marshaller.FindEncoder(theType.Elem())
	if encoder == nil { return nil }
	
	return func(scratch [64]byte, renderer Renderer, value reflect.Value) {
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

func PtrTextEncoding(marshaller Marshaller, theType reflect.Type) Encoder {
	encoder := marshaller.FindEncoder(theType.Elem())
	if encoder == nil { return nil }
	
	return func(scratch [64]byte, renderer Renderer, value reflect.Value) {
		if value.IsNil() {
			renderer.WriteNil()
		} else {
			encoder(scratch, renderer, value.Elem())
		}
	}
}

func errorTextEncoding(err *Error) Encoder {
	return func(scratch [64]byte, renderer Renderer, value reflect.Value) {
		renderer.Error(err)
	}
}