package text

import (
	"encoding"
	"encoding/base64"
	"github.com/firelizzard18/gocoding"
	"reflect"
	"strings"
)

// get type type string
func GTTS(theType reflect.Type) string {
	return theType.Elem().String()
}

// get value type string
func GVTS(value reflect.Value) string {
	return value.Type().Elem().String()
}

var decodableType1 = reflect.TypeOf(new(gocoding.Decodable1)).Elem()
var decodableType2 = reflect.TypeOf(new(gocoding.Decodable2)).Elem()
var textUnmarshallerType = reflect.TypeOf(new(encoding.TextUnmarshaler)).Elem()

func Decoding(unmarshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
	if theType.ConvertibleTo(decodableType1) {
		return Decodable1Decoding(unmarshaller, theType)
	}
	
	if theType.ConvertibleTo(decodableType2) {
		return Decodable2Decoding(unmarshaller, theType)
	}
	
	if theType.ConvertibleTo(textUnmarshallerType) {
		return textUnmarshallerDecoder
	}
	
	switch theType.Elem().Kind() {
	case reflect.Bool, reflect.Float32, reflect.Float64, reflect.String,
		 reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		 reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return decoderType{theType}.decode
		
	case reflect.Interface:
		return InterfaceDecoding(unmarshaller, theType)
	
	case reflect.Struct:
		return StructDecoding(unmarshaller, theType)
		
	case reflect.Map:
		return MapDecoding(unmarshaller, theType)
		
	case reflect.Slice:
		return SliceDecoding(unmarshaller, theType)
		
	case reflect.Array:
		return ArrayDecoding(unmarshaller, theType)
	
	default:
		return errorDecoding(gocoding.ErrorPrint("Decoding", "Unsupported type: ", GTTS(theType)))
	}
}

func errorDecoding(err *gocoding.Error) gocoding.Decoder {
	return func(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
		scanner.Error(err)
	}
}

func errorCheck(scanner gocoding.Scanner, err error) {
	if err == nil { return }
	scanner.Error(gocoding.ErrorPrint("Decoding", "An error occured while decoding: ", err.Error()))
}

func peekMatches(scanner gocoding.Scanner, codes...gocoding.ScannerCode) bool {
	peek := scanner.Peek()
	
	if peek.Matches(codes...) {
		return true
	}
	
	codestrs := make([]string, len(codes))
	for i, code := range codes {
		codestrs[i] = code.String()
	}
	
	scanner.Error(gocoding.ErrorPrintf("Decoding", "Expected one of %s, got %s", strings.Join(codestrs, ", "), peek.String()))
	return false
}

func Decodable1Decoding(unmarshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
	return func(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
		value = gocoding.NormalizeValue(scanner, "Decoding", value)
		decoder := value.Interface().(gocoding.Decodable1).Decoding(unmarshaller, theType)
		if decoder == nil { return }
		decoder(scratch, scanner, value)
	}
}

func Decodable2Decoding(unmarshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
	return func(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
		value = gocoding.NormalizeValue(scanner, "Decoding", value)
		fields := value.Interface().(gocoding.Decodable2).DecodableFields()
		
		if scanner.Peek() == gocoding.ScannedLiteralBegin {
			null := scanner.NextValue()
			if null.IsValid() && null.IsNil() {
				value.Set(reflect.Zero(theType))
				return
			}
		}
		
		if !peekMatches(scanner, gocoding.ScannedStructBegin, gocoding.ScannedMapBegin) { return }
		
		for {
			// get the next code, check for the end
			code := scanner.Continue()
			if code.Matches(gocoding.ScannedStructEnd, gocoding.ScannedMapEnd) { break }
			
			// check for key begin
			if code != gocoding.ScannedKeyBegin {
				// this will generate an appropriate error message
				peekMatches(scanner, gocoding.ScannedKeyBegin, gocoding.ScannedStructEnd, gocoding.ScannedMapEnd)
				return
			}
			
			// get the key
			key := scanner.NextValue()
			if key.Kind() != reflect.String {
				errorDecoding(gocoding.ErrorPrint("Decoding", "Invalid key type %s", key.Type().String()))
			}
			keystr := key.String()
			
			// check by name
			field := fields[keystr]
			
			// check by case-folded name (disableable?)
			if !field.IsValid() {
				for name, altfield := range fields {
					if strings.EqualFold(keystr, name) {
						field = altfield
						break
					}
				}
			}
			
			scanner.Continue()
			if !field.IsValid() {
				scanner.NextValue()
			} else {
				unmarshaller.UnmarshalValue(scanner, field)
			}
		}
	}
}

func textUnmarshallerDecoder(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
	tuvalue := value.Interface().(encoding.TextUnmarshaler)
	scanner.Continue()
	text := scanner.NextString()
	err := tuvalue.UnmarshalText([]byte(text))
	if err != nil { scanner.Error(gocoding.ErrorPrint("Text Unmarshal", err)) }
}

type decoderType struct {
	reflect.Type
}

func (t decoderType) decode(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
	value = gocoding.NormalizeValue(scanner, "Decoding", value)
	
	if !value.Type().ConvertibleTo(t.Type) {
		scanner.Error(gocoding.ErrorPrintf("Decoding", "Attempted to unmarshal %s with a %s decoder", GVTS(value), GTTS(t)))
	}
	
	json := scanner.NextValue()
	if !json.Type().ConvertibleTo(value.Type().Elem()) {
		scanner.Error(gocoding.ErrorPrintf("Decoding", "Scanned %s while unmarshalling %s", json.Type().String(), GVTS(value)))
	}
	
	value.Elem().Set(json.Convert(value.Type().Elem()))
}

func InterfaceDecoding(unmarshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
	return func (scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
		value = gocoding.NormalizeValue(scanner, "Decoding", value).Elem()
		
		if value.NumMethod() == 0 {
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
			aType = gocoding.NormalizeType(aType)
			for i := 0; i < aType.Elem().NumField(); i++ {
				sf := aType.Elem().Field(i)
				
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
				
				decoders[sf.Name] = unmarshaller.FindDecoder(gocoding.NormalizeType(sf.Type))
			}
		}
	}
	
	return func(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
		value = gocoding.NormalizeValue(scanner, "Decoding", value)
		
		if scanner.Peek() == gocoding.ScannedLiteralBegin {
			null := scanner.NextValue()
			if null.IsValid() && null.IsNil() {
				value.Set(reflect.Zero(theType))
				return
			}
		}
		
		if !peekMatches(scanner, gocoding.ScannedStructBegin, gocoding.ScannedMapBegin) { return }
		
//		if !value.IsValid() || value.IsNil() {
//			if !value.CanSet() { errorDecoding(gocoding.ErrorPrint("Decoding", "Invalid or nil and unsettable value")); return }
//			value.Set(reflect.Zero(theType))
//		}
		
		for {
			// get the next code, check for the end
			code := scanner.Continue()
			if code.Matches(gocoding.ScannedStructEnd, gocoding.ScannedMapEnd) { break }
			
			// check for key begin
			if code != gocoding.ScannedKeyBegin {
				// this will generate an appropriate error message
				peekMatches(scanner, gocoding.ScannedKeyBegin, gocoding.ScannedStructEnd, gocoding.ScannedMapEnd)
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
			
			// check by case-folded name (disableable?)
			if decoder == nil {
				for name, altdec := range decoders {
					if strings.EqualFold(keystr, name) {
						keystr, decoder = name, altdec
						break
					}
				}
			}
			
			scanner.Continue()
			if decoder == nil {
				scanner.NextValue()
			} else {
				decoder(scratch, scanner, gocoding.NormalizeValue(scanner, "Decoding", value.Elem().FieldByName(keystr)))
			}
		}
	}
}

func MapDecoding(unmarshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
	if theType.Key().Kind() != reflect.String {
		return errorDecoding(gocoding.ErrorPrint("Decoding", "Unsupported map key type: ", theType.Key()))
	}
	
	decoder := unmarshaller.FindDecoder(theType.Elem().Elem())
	if decoder == nil { return nil }
	
	return func(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
		value = gocoding.NormalizeValue(scanner, "Decoding", value)
		
		if scanner.Peek() == gocoding.ScannedLiteralBegin {
			null := scanner.NextValue()
			if null.IsValid() && null.IsNil() {
				value.Set(reflect.Zero(theType))
				return
			}
		}
		
		if !peekMatches(scanner, gocoding.ScannedStructBegin, gocoding.ScannedMapBegin) { return }
		
//		if !value.IsValid() || value.IsNil() {
//			if !value.CanSet() { errorDecoding(gocoding.ErrorPrint("Decoding", "Invalid or nil and unsettable value")); return }
//			value.Set(reflect.Zero(theType))
//		}
		
		for {
			// get the next code, check for the end
			code := scanner.Continue()
			if code.Matches(gocoding.ScannedStructEnd, gocoding.ScannedMapEnd) { break }
			
			// check for key begin
			if code != gocoding.ScannedKeyBegin {
				// this will generate an appropriate error message
				peekMatches(scanner, gocoding.ScannedKeyBegin, gocoding.ScannedStructEnd, gocoding.ScannedMapEnd)
				return
			}
			
			// get the key
			key := scanner.NextValue()
			if key.Kind() != reflect.String {
				errorDecoding(gocoding.ErrorPrint("Decoding", "Invalid key type %s", key.Type().String()))
			}
			
			elem := value.Elem().MapIndex(key)
			if !elem.IsValid() || elem.IsNil() {
				elem = reflect.Zero(elem.Type())
				value.Elem().SetMapIndex(key, elem)
			}
			elem = gocoding.NormalizeValue(scanner, "Decoding", elem)
			
			decoder(scratch, scanner, elem)
		}
	}
}

func ArrayDecoding(unmarshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
	decoder := unmarshaller.FindDecoder(gocoding.NormalizeType(theType.Elem().Elem()))
	if decoder == nil { return nil }
	
	return func(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
		value = gocoding.NormalizeValue(scanner, "Decoding", value)
		
		if scanner.Peek() == gocoding.ScannedLiteralBegin {
			null := scanner.NextValue()
			if null.IsValid() && null.IsNil() {
				value.Set(reflect.Zero(theType))
				return
			}
		}
		
		if !peekMatches(scanner, gocoding.ScannedArrayBegin) { return }
		
		for i := 0; true; i++ {
			// get the next code, check for the end
			code := scanner.Continue()
			if code.Matches(gocoding.ScannedArrayEnd) { break }
			
			// decode until full, skip any excess entries
			if i < value.Elem().Len() {
				decoder(scratch, scanner, value.Elem().Index(i))
			}
		}
	}
}

func SliceDecoding(unmarshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
	if theType.Elem().Elem().Kind() == reflect.Uint8 {
		return byteSliceDecoder
	}
	
	decoder := unmarshaller.FindDecoder(gocoding.NormalizeType(theType.Elem().Elem()))
	if decoder == nil { return nil }
	
	return func(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
		value = gocoding.NormalizeValue(scanner, "Decoding", value)
		
		if scanner.Peek() == gocoding.ScannedLiteralBegin {
			null := scanner.NextValue()
			if null.IsValid() && null.IsNil() {
				value.Set(reflect.Zero(theType))
				return
			}
		}
		
		if !peekMatches(scanner, gocoding.ScannedArrayBegin) { return }
		
		for i := 0; true; i++ {
			// get the next code, check for the end
			code := scanner.Continue()
			if code.Matches(gocoding.ScannedArrayEnd) { break }
			
			if i >= value.Elem().Cap() {
				scap := value.Elem().Cap()
				scap = scap + scap/2
				if scap < 4 { scap = 4 }
				
				newv := reflect.MakeSlice(value.Type().Elem(), value.Elem().Len(), scap)
				reflect.Copy(newv, value.Elem())
				value.Elem().Set(newv)
			}
			
			if i >= value.Elem().Len() {
				value.Elem().SetLen(i + 1)
			}
			
			decoder(scratch, scanner, value.Elem().Index(i))
		}
	}
}

func byteSliceDecoder(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
	value = gocoding.NormalizeValue(scanner, "Decoding", value).Elem()
	
	bytes := scanner.NextValue()
	switch bytes.Kind() {
//	case reflect.Something:
//		value.Set(reflect.Zero(value.Type()))
	
	case reflect.String:
		data, err := base64.StdEncoding.DecodeString(bytes.String())
		errorCheck(scanner, err)
		
		value.Set(reflect.ValueOf(data))
	
	default:
		errorDecoding(gocoding.ErrorPrint("Decoding", "Decoding byte slice: expected String, got %s", bytes.Type().String()))
	}
}