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
		return renderer.WriteString("true")
	} else {
		return renderer.WriteString("false")
	}
}
	
func intEncoder(scratch [64]byte, renderer Renderer, value reflect.Value) error {
	return renderer.WriteData(strconv.AppendInt(scratch[:0], value.Int(), 10))
}
	
func uintEncoder(scratch [64]byte, renderer Renderer, value reflect.Value) error {
	return renderer.WriteData(strconv.AppendUint(scratch[:0], value.Uint(), 10))
}

type floatEncoder int

func (bits floatEncoder) encode(scratch [64]byte, renderer Renderer, value reflect.Value) error {
	f := value.Float()
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return errors.New(fmt.Sprint("Unsupported float value: ", strconv.FormatFloat(f, 'g', -1, int(bits))))
	}
	return renderer.WriteData(strconv.AppendFloat(scratch[:0], f, 'g', -1, int(bits)))
}

var (
	float32Encoder = (floatEncoder(32)).encode
	float64Encoder = (floatEncoder(64)).encode
)

func stringEncoder(scratch [64]byte, renderer Renderer, value reflect.Value) error {
	return renderer.WriteString(value.String())
}

func interfaceEncoder(scratch [64]byte, renderer Renderer, value reflect.Value) error {
	if value.IsNil() {
		return renderer.WriteNil()
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
			
			if err := renderer.StartElement(field.Name); err != nil { return err }
			if err := encoder(scratch, renderer, value.FieldByName(field.Name)); err != nil { return err }
			if err := renderer.StopElement(field.Name); err != nil { return err }
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
			return renderer.WriteNil()
		}
		
		if err := renderer.StartMap(); err != nil { return err }
		
		for _, key := range value.MapKeys() {
			if err := renderer.StartElement(key.String()); err != nil { return err }
			if err := encoder(scratch, renderer, value.MapIndex(key)) ; err != nil { return err }
			if err := renderer.StopElement(key.String()); err != nil { return err }
		}
		
		return renderer.StopMap()
	}, nil
}

func newSliceEncoder(encoding Encoding, theType reflect.Type) (Encoder, error) {
	encoder, err := newArrayEncoder(encoding, theType)
	if err != nil { return nil, err }
	
	return func(scratch [64]byte, renderer Renderer, value reflect.Value) error {
		if value.IsNil() {
			return renderer.WriteNil()
		}
		encoder(scratch, renderer, value)
		return nil
	}, nil
}

func newArrayEncoder(encoding Encoding, theType reflect.Type) (Encoder, error) {
	eType := theType.Elem()
	if eType.Kind() == reflect.Uint8 {
		// some shit
	}
	
	encoder, err := encoding(encoding, eType)
	if err != nil { return nil, err }
	
	return func(scratch [64]byte, renderer Renderer, value reflect.Value) error {
		count := value.Len()
		
		if err := renderer.StartArray(); err != nil { return err }
		
		for i := 0; i < count; i++ {
			id := strconv.Itoa(i)
			if err := renderer.StartElement(id); err != nil { return err }
			if err := encoder(scratch, renderer, value.Index(i)); err != nil { return err }
			if err := renderer.StopElement(id); err != nil { return err }
		}
		
		return renderer.StopArray()
	}, nil
}

func newPtrEncoder(encoding Encoding, theType reflect.Type) (Encoder, error) {
	encoder, err := encoding(encoding, theType.Elem())
	if err != nil { return nil, err }
	
	return func(scratch [64]byte, renderer Renderer, value reflect.Value) error {
		if value.IsNil() {
			return renderer.WriteNil()
		}
		
		return encoder(scratch, renderer, value.Elem())
	}, nil
}
















