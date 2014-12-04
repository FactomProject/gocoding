package text

import (
"fmt"
	"encoding"
	"encoding/base64"
	"github.com/FactomProject/gocoding"
	"reflect"
	"strings"
)

// get type type string
func GTTS(theType reflect.Type) string {
	return theType.String()
}

// get value type string
func GVTS(value reflect.Value) string {
	return value.Type().String()
}

var textUnmarshallerType = reflect.TypeOf(new(encoding.TextUnmarshaler)).Elem()

func Decoding(unmarshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
	if theType.ConvertibleTo(textUnmarshallerType) {
		return textUnmarshallerDecoder
	}
	
	var decoder gocoding.Decoder
	switch theType.Kind() {
	case reflect.Bool, reflect.Float32, reflect.Float64, reflect.String,
		 reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		 reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		decoder = decoderType{theType}.decode
		
	case reflect.Interface:
		decoder = InterfaceDecoding(unmarshaller, theType)
	
	case reflect.Struct:
		decoder = StructDecoding(unmarshaller, theType)
		
	case reflect.Map:
		decoder = MapDecoding(unmarshaller, theType)
		
	case reflect.Slice:
		decoder = SliceDecoding(unmarshaller, theType)
		
	case reflect.Array:
		decoder = ArrayDecoding(unmarshaller, theType)
		
	case reflect.Ptr:
		decoder = PtrDecoding(unmarshaller, theType)
	
	default:
		decoder = gocoding.ErrorDecoding(gocoding.ErrorPrint("Decoding", "Unsupported type: ", GTTS(theType)))
	}
	
	if reflect.PtrTo(theType).ConvertibleTo(textUnmarshallerType) {
		return gocoding.TryIndirectDecoding(decoder, textUnmarshallerDecoder)
	}
	
	return decoder
}

func errorCheck(scanner gocoding.Scanner, err error) {
	if err == nil { return }
	scanner.Error(gocoding.ErrorPrint("Decoding", "An error occured while decoding: ", err.Error()))
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
	if !value.Type().ConvertibleTo(t.Type) {
		scanner.Error(gocoding.ErrorPrintf("Decoding", "Attempted to unmarshal %s with a %s decoder", GVTS(value), GTTS(t)))
	}
	
	json := scanner.NextValue()
	if !json.Type().ConvertibleTo(value.Type()) {
		scanner.Error(gocoding.ErrorPrintf("Decoding", "Scanned %s while unmarshalling %s", json.Type().String(), GVTS(value)))
	}
	
	value.Set(json.Convert(value.Type()))
}

func InterfaceDecoding(unmarshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
	return func (scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
		fmt.Println("interface decoding", theType, value)
		if value.IsNil() {
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
		if scanner.Peek() == gocoding.ScannedLiteralBegin {
			null := scanner.NextValue()
			if null.IsValid() && null.IsNil() {
				value.Set(reflect.Zero(theType))
				return
			}
		}
		
		if !gocoding.PeekCheck(scanner, gocoding.ScannedStructBegin, gocoding.ScannedMapBegin) { return }
		
		for {
			// get the next code, check for the end
			code := scanner.Continue()
			if code.Matches(gocoding.ScannedStructEnd, gocoding.ScannedMapEnd) { break }
			
			// check for key begin
			if code != gocoding.ScannedKeyBegin {
				// this will generate an appropriate error message
				gocoding.PeekCheck(scanner, gocoding.ScannedKeyBegin, gocoding.ScannedStructEnd, gocoding.ScannedMapEnd)
				return
			}
			
			// get the key
			key := scanner.NextValue()
			if key.Kind() != reflect.String {
				scanner.Error(gocoding.ErrorPrint("Decoding", "Invalid key type %s", key.Type().String()))
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
				decoder(scratch, scanner, value.FieldByName(keystr))
			}
		}
	}
}

func MapDecoding(unmarshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
	if theType.Key().Kind() != reflect.String {
		return gocoding.ErrorDecoding(gocoding.ErrorPrint("Decoding", "Unsupported map key type: ", theType.Key()))
	}
	
	elemType := theType.Elem()
	decoder := unmarshaller.FindDecoder(elemType)
	ptrDecoder := unmarshaller.FindDecoder(reflect.PtrTo(elemType))
	if decoder == nil { return nil }
	
	return func(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
		
		if scanner.Peek() == gocoding.ScannedLiteralBegin {
			null := scanner.NextValue()
			if null.IsValid() && null.IsNil() {
				value.Set(reflect.Zero(theType))
				return
			}
		}
		
		if value.IsNil() {
			value.Set(reflect.MakeMap(theType))
		}
		
		if !gocoding.PeekCheck(scanner, gocoding.ScannedStructBegin, gocoding.ScannedMapBegin) { return }
		
		for {
			// get the next code, check for the end
			code := scanner.Continue()
			if code.Matches(gocoding.ScannedStructEnd, gocoding.ScannedMapEnd) { break }
			
			// check for key begin
			if code != gocoding.ScannedKeyBegin {
				// this will generate an appropriate error message
				gocoding.PeekCheck(scanner, gocoding.ScannedKeyBegin, gocoding.ScannedStructEnd, gocoding.ScannedMapEnd)
				return
			}
			
			// get the key
			key := scanner.NextValue()
			if key.Kind() != reflect.String {
				scanner.Error(gocoding.ErrorPrint("Decoding", "Invalid key type %s", key.Type().String()))
			}
			
			elem := value.MapIndex(key)
			scanner.Continue()
			
			if elem.IsValid() {
				decoder(scratch, scanner, elem)
			} else {
				elem = reflect.New(elemType)
				ptrDecoder(scratch, scanner, elem)
				value.SetMapIndex(key, elem.Elem())
			}
		}
	}
}

func ArrayDecoding(unmarshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
	decoder := unmarshaller.FindDecoder(theType.Elem())
	if decoder == nil { return nil }
	
	return func(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
		if scanner.Peek() == gocoding.ScannedLiteralBegin {
			null := scanner.NextValue()
			if null.IsValid() && null.IsNil() {
				value.Set(reflect.Zero(theType))
				return
			}
		}
		
		if !gocoding.PeekCheck(scanner, gocoding.ScannedArrayBegin) { return }
		
		for i := 0; true; i++ {
			// get the next code, check for the end
			code := scanner.Continue()
			if code.Matches(gocoding.ScannedArrayEnd) { break }
			
			// decode until full, skip any excess entries
			if i < value.Len() {
				decoder(scratch, scanner, value.Index(i))
			}
		}
	}
}

func SliceDecoding(unmarshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
	if theType.Elem().Kind() == reflect.Uint8 {
		return ByteSliceDecoder
	}
	
	decoder := unmarshaller.FindDecoder(theType.Elem())
	if decoder == nil { return nil }
	
	return func(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
		if scanner.Peek() == gocoding.ScannedLiteralBegin {
			null := scanner.NextValue()
			if null.IsValid() && null.IsNil() {
				value.Set(reflect.Zero(theType))
				return
			}
		}
		
		if !gocoding.PeekCheck(scanner, gocoding.ScannedArrayBegin) { return }
		
		for i := 0; true; i++ {
			// get the next code, check for the end
			code := scanner.Continue()
			if code.Matches(gocoding.ScannedArrayEnd) { break }
			
			if i >= value.Cap() {
				scap := value.Cap()
				scap = scap + scap/2
				if scap < 4 { scap = 4 }
				
				newv := reflect.MakeSlice(value.Type(), value.Len(), scap)
				reflect.Copy(newv, value)
				value.Set(newv)
			}
			
			if i >= value.Len() {
				value.SetLen(i + 1)
			}
			
			decoder(scratch, scanner, value.Index(i))
		}
	}
}

func ByteSliceDecoder(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
	bytes := scanner.NextValue()
	switch bytes.Kind() {
//	case reflect.Something:
//		value.Set(reflect.Zero(value.Type()))
	
	case reflect.String:
		data, err := base64.StdEncoding.DecodeString(bytes.String())
		errorCheck(scanner, err)
		
		value.Set(reflect.ValueOf(data))
	
	default:
		scanner.Error(gocoding.ErrorPrint("Decoding", "Decoding byte slice: expected String, got %s", bytes.Type().String()))
	}
}


func PtrDecoding(unmarshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
	decoder := unmarshaller.FindDecoder(theType.Elem())
	if decoder == nil { return nil }
	
	return func(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
		if value.IsNil() {
			value.Set(reflect.New(theType.Elem()))
		}
		
		decoder(scratch, scanner, value.Elem())
	}
}
