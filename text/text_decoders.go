package text

import (
	"encoding"
	"encoding/base64"
	"github.com/firelizzard18/gocoding"
	"reflect"
	"strconv"
	"strings"
)

//var encodableType1 = reflect.TypeOf(new(gocoding.Decodable1)).Elem()
//var encodableType2 = reflect.TypeOf(new(gocoding.Decodable2)).Elem()
var textUnmarshallerType = reflect.TypeOf(new(encoding.TextUnmarshaler)).Elem()

func Decoding(marshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
//	if theType.ConvertibleTo(encodableType1) {
//		return Decodable1Decoding(marshaller, theType)
//	}
//	
//	if theType.ConvertibleTo(encodableType2) {
//		return Decodable2Decoding(marshaller, theType)
//	}
//	
//	if theType.ConvertibleTo(textUnmarshallerType) {
//		return textUnmarshallerDecoder
//	}
	
	var decoder gocoding.Decoder
	switch theType.Kind() {
	case reflect.Bool, reflect.Float32, reflect.Float64, reflect.String,
		 reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		 reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		decoder = decoderType{theType}.decode
		
	case reflect.Interface:
		decoder = InterfaceDecoding(marshaller, theType)
	
	case reflect.Struct:
		decoder = StructDecoding(marshaller, theType)
		
	case reflect.Map:
		decoder = MapDecoding(marshaller, theType)
		
	case reflect.Slice:
		decoder = SliceDecoding(marshaller, theType)
		
	case reflect.Array:
		decoder = ArrayDecoding(marshaller, theType)
		
	case reflect.Ptr:
		decoder = PtrDecoding(marshaller, theType)
	
	default:
		decoder = errorDecoding(gocoding.ErrorPrint("Decoding", "Unsupported type: ", theType))
	}
	
//	if theType.Kind() == reflect.Ptr {
		return decoder
//	}
//	
//	ptrType := reflect.PtrTo(theType)
//	if ptrType.ConvertibleTo(decodableType1) ||
//	   ptrType.ConvertibleTo(decodableType2) ||
//	   ptrType.ConvertibleTo(textUarshallerType) {
//		return indirectDecoder(decoder, unmarshaller.FindDecoder(ptrType))
//	} else {
//		return decoder
//	}
}

func errorDecoding(err *gocoding.Error) gocoding.Decoder {
	return func(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
		scanner.Error(err)
	}
}

func errorCheck(scanner gocoding.Scanner, err error) {
	scanner.Error(gocoding.ErrorPrint("Decoding", "An error occured while decoding: ", err.Error()))
}

//func nextString(scanner gocoding.Scanner) string {
//	value := scanner.NextValue()
//	if !value.IsValid() {
//		scanner.Error(gocoding.ErrorPrint("Decoding", "Scanner's next value is not valid"))
//		return ""
//	}
//	if !value.IsNil() {
//		return ""
//	}
//	return value.String()
//}

func continueToCode(scanner gocoding.Scanner, codes...gocoding.ScannerCode) bool {
	next := scanner.Continue()
	
	if next.Matches(codes...) {
		return true
	}
	
	codestrs := make([]string, len(codes))
	for i, code := range codes {
		codestrs[i] = code.String()
	}
	
	scanner.Error(gocoding.ErrorPrintf("Decoding", "Expected one of%s, got %s", strings.Join(codestrs, ", "), next.String()))
	return false
}

//func Decodable1Decoding(marshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
//	return func(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
//		if value.IsNil() {
//			scanner.WriteNil()
//		} else {
//			encoding := value.Interface().(gocoding.Decodable1).Decoding(marshaller, theType)
//			if encoding == nil { return }
//			encoding(scratch, scanner, value)
//		}
//	}
//}
//
//func Decodable2Decoding(marshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
//	return func(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
//		if value.IsNil() {
//			scanner.WriteNil()
//		} else {
//			scanner.StartStruct()
//			for name, value := range value.Interface().(gocoding.Decodable2).DecodableFields() {
//				scanner.StartElement(name)
//				marshaller.UnmarshalValue(scanner, value)
//				scanner.StopElement(name)
//			}
//			scanner.StopStruct()
//		}
//	}
//}
//
//func textUnmarshallerDecoder(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
//	tmvalue := value.Interface().(encoding.TextUnmarshaler)
//	err := tmvalue.UnmarshalText(error)
//	if err != nil { scanner.Error(gocoding.ErrorPrint("Text Unmarshal", err)) }
//}

type decoderType struct {
	reflect.Type
}

func (t decoderType) decode(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
	if !value.Type().ConvertibleTo(t.Type) {
		scanner.Error(gocoding.ErrorPrintf("Decoding", "Attempted to unmarshal %s with a %s decoder", t.String(), value.Type().String()))
	}
	
	json := scanner.NextValue()
	if !json.Type().ConvertibleTo(value.Type()) {
		scanner.Error(gocoding.ErrorPrintf("Decoding", "Scanned %s while unmarshalling %s", json.Type(), value.Type()))
	}
	
	value.Set(json)
}

func InterfaceDecoding(unmarshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
	return func (scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
		if !value.IsValid() || value.IsNil() {
			value.Set(scanner.NextValue())
		} else {
			unmarshaller.UnmarshalValue(scanner, value.Elem())
		}
	}
}

func StructDecoding(unmarshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
	decoders := make(map[string]gocoding.Decoder)
	
	current := []reflect.Type{}
	next := []reflect.Type{theType}
	
	for len(next) > 0 {
		current, next = next, current[:0]
		
		for _, aType := range current {
			for i := 0; i < aType.NumField(); i++ {
				sf := aType.Field(i)
				
				// skip masked fields
				if _, ok := decoders[sf.Name]; ok {
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
				
				decoders[sf.Name] = unmarshaller.FindDecoder(sf.Type)
			}
		}
	}
	
	return func(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
		if !continueToCode(scanner, gocoding.ScannedStructBegin, gocoding.ScannedMapBegin) { return }
		
		if !value.IsValid() || value.IsNil() {
			if !value.CanSet() { errorDecoding(gocoding.ErrorPrint("Decoding", "Invalid or nil and unsettable value")); return }
			value.Set(reflect.Zero(theType))
		}
		
		for {
			// get the next code, check for the end
			code := scanner.Continue()
			if code.Matches(gocoding.ScannedStructEnd, gocoding.ScannedMapEnd) { break }
			
			// check for key begin
			if code != gocoding.ScannedKeyBegin {
				// this will generate an appropriate error message
				continueToCode(scanner, gocoding.ScannedKeyBegin, gocoding.ScannedStructBegin, gocoding.ScannedMapBegin)
				return
			}
			
			// get the key
			key := scanner.NextValue()
			if key.Kind() != reflect.String {
				errorDecoding(gocoding.ErrorPrint("Decoding", "Invalid key type %s", key.Type().String()))
			}
			keystr := key.String()
			
			// check by name
			decoder := decoders[keystr]
			if decoder != nil {
				decoder(scratch, scanner, value.FieldByName(keystr))
				continue
			}
			
			// check by case-folded name (disableable?)
			for name, decoder := range decoders {
				if !strings.EqualFold(keystr, name) { continue }
				decoder = decoders[keystr]
				if decoder == nil { errorDecoding(gocoding.ErrorPrint("Decoding", "Internal error: nil decoder")); return }
				decoder(scratch, scanner, value.FieldByName(name))
				break
			}
		}
	}
}

func MapDecoding(unmarshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
	if theType.Key().Kind() != reflect.String {
		return errorDecoding(gocoding.ErrorPrint("Decoding", "Unsupported map key type: ", theType.Key()))
	}
	
	elemType := theType.Elem()
	decoder := unmarshaller.FindDecoder(elemType)
	if decoder == nil { return nil }
	
	return func(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
		if !continueToCode(scanner, gocoding.ScannedStructBegin, gocoding.ScannedMapBegin) { return }
		
		if !value.IsValid() || value.IsNil() {
			if !value.CanSet() { errorDecoding(gocoding.ErrorPrint("Decoding", "Invalid or nil and unsettable value")); return }
			value.Set(reflect.Zero(value.Type()))
		}
		
		for {
			// get the next code, check for the end
			code := scanner.Continue()
			if code.Matches(gocoding.ScannedStructEnd, gocoding.ScannedMapEnd) { break }
			
			// check for key begin
			if code != gocoding.ScannedKeyBegin {
				// this will generate an appropriate error message
				continueToCode(scanner, gocoding.ScannedKeyBegin, gocoding.ScannedStructBegin, gocoding.ScannedMapBegin)
				return
			}
			
			// get the key
			key := scanner.NextValue()
			if key.Kind() != reflect.String {
				errorDecoding(gocoding.ErrorPrint("Decoding", "Invalid key type %s", key.Type().String()))
			}
			
			mapElem := value.MapIndex(key)
			if !mapElem.IsValid() || mapElem.IsNil() {
				if !mapElem.CanSet() { errorDecoding(gocoding.ErrorPrint("Decoding", "Invalid or nil and unsettable value")); return }
				mapElem.Set(reflect.Zero(elemType))
				value.SetMapIndex(key, mapElem)
			}
			
			decoder(scratch, scanner, mapElem)
		}
	}
}

func SliceDecoding(unmarshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
	if theType.Elem().Kind() == reflect.Uint8 {
		return byteSliceDecoder
	}
	
	decoder := ArrayDecoding(unmarshaller, theType)
	if decoder == nil { return nil }
	
	return func(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
		if value.IsNil() {
			scanner.WriteNil()
		} else {
			decoder(scratch, scanner, value)
		}
	}
}

func byteSliceDecoder(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
	bytes := scanner.NextValue()
	if bytes.Kind() != reflect.String {
		errorDecoding(gocoding.ErrorPrint("Decoding", "Decoding byte slice: expected String, got %s", bytes.Type().String()))
	}
	
}

func ArrayDecoding(unmarshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
	decoder := unmarshaller.FindDecoder(theType.Elem())
	if decoder == nil { return nil }
	
	return func(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
		count := value.Len()
		
		scanner.StartArray()
		
		for i := 0; i < count; i++ {
			id := strconv.Itoa(i)
			scanner.StartElement(id)
			decoder(scratch, scanner, value.Index(i))
			scanner.StopElement(id)
		}
		
		scanner.StopArray()
	}
}

func PtrDecoding(unmarshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
	decoder := unmarshaller.FindDecoder(theType.Elem())
	if decoder == nil { return nil }
	
	return func(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
		if value.IsNil() {
			scanner.WriteNil()
		} else {
			decoder(scratch, scanner, value.Elem())
		}
	}
}