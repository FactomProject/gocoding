package gocoding

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
)

func TextEncoding(encoding Encoding, theType reflect.Type) (Encoder, error) {
	switch theType.Kind() {
	case reflect.Bool:
		return boolEncoder, nil
	
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return intEncoder, nil
	
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return uintEncoder, nil
		
	case reflect.Float32:
		return float32Encoder, nil
		
	case reflect.Float64:
		return float64Encoder, nil
		
	case reflect.String:
		return stringEncoder, nil
		
	case reflect.Interface:
		return interfaceEncoder, nil
	
	case reflect.Struct:
		return newStructEncoder(encoding, theType)
		
	case reflect.Map:
		return newMapEncoder(encoding, theType)
		
	case reflect.Slice:
		return newSliceEncoder(encoding, theType)
		
	case reflect.Array:
		return newArrayEncoder(encoding, theType)
		
	case reflect.Ptr:
		return newPtrEncoder(encoding, theType)
	
	default:
		return nil, errors.New(fmt.Sprint("Unsupported type: ", theType))
	}
}
	
func boolEncoder(scratch [64]byte, renderer Renderer, value reflect.Value) error {
	if value.Bool() {
		renderer.WriteString("true")
	} else {
		renderer.WriteString("false")
	}
	return nil
}
	
func intEncoder(scratch [64]byte, renderer Renderer, value reflect.Value) error {
	renderer.WriteData(strconv.AppendInt(scratch[:0], value.Int(), 10))
	return nil
}
	
func uintEncoder(scratch [64]byte, renderer Renderer, value reflect.Value) error {
	renderer.WriteData(strconv.AppendUint(scratch[:0], value.Uint(), 10))
	return nil
}

type floatEncoder int

func (bits floatEncoder) encode(scratch [64]byte, renderer Renderer, value reflect.Value) error {
	f := value.Float()
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return errors.New(fmt.Sprint("Unsupported float value: ", strconv.FormatFloat(f, 'g', -1, int(bits))))
	}
	renderer.WriteData(strconv.AppendFloat(scratch[:0], f, 'g', -1, int(bits)))
	return nil
}

var (
	float32Encoder = (floatEncoder(32)).encode
	float64Encoder = (floatEncoder(64)).encode
)

func stringEncoder(scratch [64]byte, renderer Renderer, value reflect.Value) error {
	renderer.WriteString(value.String())
	return nil
}

func interfaceEncoder(scratch [64]byte, renderer Renderer, value reflect.Value) error {
	if value.IsNil() {
		renderer.WriteNil()
		return nil
	}
	panic(errors.New("figure out what to do here"))
}

func newStructEncoder(encoding Encoding, theType reflect.Type) (Encoder, error) {
	fields := make([]reflect.StructField, 0)
	
	panic(errors.New("build list"))
	
	return func(scratch [64]byte, renderer Renderer, value reflect.Value) error {
		for _, field := range fields {
			encoder, err := encoding(encoding, field.Type)
			if err != nil { return err }
			
			renderer.StartElement(field.Name)
			encoder(scratch, renderer, value.FieldByName(field.Name))
			renderer.StopElement(field.Name)
		}
		
		return nil
	}, nil
}

func newMapEncoder(encoding Encoding, theType reflect.Type) (Encoder, error) {
	if theType.Key().Kind() != reflect.String {
		return nil, errors.New(fmt.Sprint("Unsupported map key type: ", theType.Key()))
	}
	
	encoder, err := encoding(encoding, theType.Elem())
	if err != nil { return nil, err }
	
	return func(scratch [64]byte, renderer Renderer, value reflect.Value) error {
		if value.IsNil() {
			renderer.WriteNil()
			return nil
		}
		
		renderer.StartMap()
		
		for _, key := range value.MapKeys() {
			renderer.StartElement(key.String())
			encoder(scratch, renderer, value.MapIndex(key))
			renderer.StopElement(key.String())
		}
		
		renderer.StopMap()
		return nil
	}, nil
}