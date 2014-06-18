package gocoding

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
)

func TextEncoding(marshaller Marshaller, theType reflect.Type) (Encoder, error) {
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
		return InterfaceEncoding(marshaller, theType)
	
	case reflect.Struct:
		return StructEncoding(marshaller, theType)
		
	case reflect.Map:
		return MapEncoding(marshaller, theType)
		
	case reflect.Slice:
		return SliceEncoding(marshaller, theType)
		
	case reflect.Array:
		return ArrayEncoding(marshaller, theType)
		
	case reflect.Ptr:
		return PtrEncoding(marshaller, theType)
	
	default:
		return nil, errors.New(fmt.Sprint("Unsupported type: ", theType))
	}
}
	
func boolEncoder(scratch [64]byte, renderer Renderer, value reflect.Value) error {
	if value.Bool() {
		_, err := renderer.Print("true")
		return err
	} else {
		_, err := renderer.Print("false")
		return err
	}
}
	
func intEncoder(scratch [64]byte, renderer Renderer, value reflect.Value) error {
	_, err := renderer.Print(strconv.AppendInt(scratch[:0], value.Int(), 10))
	return err
}
	
func uintEncoder(scratch [64]byte, renderer Renderer, value reflect.Value) error {
	_, err := renderer.Print(strconv.AppendUint(scratch[:0], value.Uint(), 10))
	return err
}

type floatEncoder int

func (bits floatEncoder) encode(scratch [64]byte, renderer Renderer, value reflect.Value) error {
	f := value.Float()
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return errors.New(fmt.Sprint("Unsupported float value: ", strconv.FormatFloat(f, 'g', -1, int(bits))))
	}
	_, err := renderer.Print(strconv.AppendFloat(scratch[:0], f, 'g', -1, int(bits)))
	return err
}

var (
	float32Encoder = (floatEncoder(32)).encode
	float64Encoder = (floatEncoder(64)).encode
)

func stringEncoder(scratch [64]byte, renderer Renderer, value reflect.Value) error {
	_, err := renderer.Print(value.String())
	return err
}

func InterfaceEncoding(marshaller Marshaller, theType reflect.Type) (Encoder, error) {
	return func (scratch [64]byte, renderer Renderer, value reflect.Value) error {
		if value.IsNil() {
			_, err := renderer.WriteNil()
			return err
		}
		
		encoder, err := marshaller.FindEncoder(value.Type())
		if err != nil { return err }
		
		return encoder(scratch, renderer, value)
	}, nil
}

func StructEncoding(marshaller Marshaller, theType reflect.Type) (Encoder, error) {
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
				if sf.Anonymous {
					next = append(next, sf.Type)
					continue
				}
				
				encoder, err := marshaller.FindEncoder(sf.Type)
				if err != nil { return nil, err }
				
				encoders[sf.Name] = encoder
			}
		}
	}
	
	return func(scratch [64]byte, renderer Renderer, value reflect.Value) error {
		if _, err := renderer.StartStruct(); err != nil { return err }
		
		for name, encoder := range encoders {
			if _, err := renderer.StartElement(name); err != nil { return err }
			if err := encoder(scratch, renderer, value.FieldByName(name)); err != nil { return err }
			if _, err := renderer.StopElement(name); err != nil { return err }
		}
		
		_, err := renderer.StopStruct()
		return err
	}, nil
}

func MapEncoding(marshaller Marshaller, theType reflect.Type) (Encoder, error) {
	if theType.Key().Kind() != reflect.String {
		return nil, errors.New(fmt.Sprint("Unsupported map key type: ", theType.Key()))
	}
	
	encoder, err := marshaller.FindEncoder(theType.Elem())
	if err != nil { return nil, err }
	
	return func(scratch [64]byte, renderer Renderer, value reflect.Value) error {
		if value.IsNil() {
			_, err := renderer.WriteNil()
			return err
		}
		
		if _, err := renderer.StartMap(); err != nil { return err }
		
		for _, key := range value.MapKeys() {
			if _, err := renderer.StartElement(key.String()); err != nil { return err }
			if err := encoder(scratch, renderer, value.MapIndex(key)) ; err != nil { return err }
			if _, err := renderer.StopElement(key.String()); err != nil { return err }
		}
		
		_, err := renderer.StopMap()
		return err
	}, nil
}

func SliceEncoding(marshaller Marshaller, theType reflect.Type) (Encoder, error) {
	encoder, err := ArrayEncoding(marshaller, theType)
	if err != nil { return nil, err }
	
	return func(scratch [64]byte, renderer Renderer, value reflect.Value) error {
		if value.IsNil() {
			_, err := renderer.WriteNil()
			return err
		}
		encoder(scratch, renderer, value)
		return nil
	}, nil
}

func ArrayEncoding(marshaller Marshaller, theType reflect.Type) (Encoder, error) {
	eType := theType.Elem()
	if eType.Kind() == reflect.Uint8 {
		// some shit
	}
	
	encoder, err := marshaller.FindEncoder(eType)
	if err != nil { return nil, err }
	
	return func(scratch [64]byte, renderer Renderer, value reflect.Value) error {
		count := value.Len()
		
		if _, err := renderer.StartArray(); err != nil { return err }
		
		for i := 0; i < count; i++ {
			id := strconv.Itoa(i)
			if _, err := renderer.StartElement(id); err != nil { return err }
			if err := encoder(scratch, renderer, value.Index(i)); err != nil { return err }
			if _, err := renderer.StopElement(id); err != nil { return err }
		}
		
		_, err := renderer.StopArray()
		return err
	}, nil
}

func PtrEncoding(marshaller Marshaller, theType reflect.Type) (Encoder, error) {
	encoder, err := marshaller.FindEncoder(theType.Elem())
	if err != nil { return nil, err }
	
	return func(scratch [64]byte, renderer Renderer, value reflect.Value) error {
		if value.IsNil() {
			_, err := renderer.WriteNil()
			return err
		}
		
		return encoder(scratch, renderer, value.Elem())
	}, nil
}