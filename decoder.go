package gocoding

import (
	"reflect"
	"strings"
)

func ErrorDecoding(err *Error) Decoder {
	return func(scratch [64]byte, scanner Scanner, value reflect.Value) {
		scanner.Error(err)
	}
}

func ErrorScanning(got ScannerCode, expected...ScannerCode) *Error {
	codestrs := make([]string, len(expected))
	for i, code := range expected {
		codestrs[i] = code.String()
	}
	return ErrorPrintf("Decoding", "Expected one of %s, got %s", strings.Join(codestrs, ", "), got.String())
}

func PeekCheck(scanner Scanner, expected...ScannerCode) bool {
	got := scanner.Peek()
	
	if got.Matches(expected...) {
		return true
	}
	
	scanner.Error(ErrorScanning(got, expected...))
	return false
}

func TryIndirectDecoding(direct Decoder, indirect Decoder) Decoder {
	return func(scratch [64]byte, scanner Scanner, value reflect.Value) {
		if value.CanAddr() {
			indirect(scratch, scanner, value.Addr())
		} else {
			direct(scratch, scanner, value)
		}
	}
}

func Decodable2Decoding(unmarshaller Unmarshaller, theType reflect.Type) Decoder {
	return func(scratch [64]byte, scanner Scanner, value reflect.Value) {
		if theType.Kind() == reflect.Ptr && value.IsNil() {
			value.Set(reflect.New(theType.Elem()))
		}
		
		fields := value.Interface().(Decodable2).DecodableFields()
		
		if scanner.Peek() == ScannedLiteralBegin {
			null := scanner.NextValue()
			if null.IsValid() && null.IsNil() {
				value.Set(reflect.Zero(theType))
				return
			}
		}
		
		if !PeekCheck(scanner, ScannedStructBegin, ScannedMapBegin) { return }
		
		for {
			// get the next code, check for the end
			code := scanner.Continue()
			if code.Matches(ScannedStructEnd, ScannedMapEnd) { break }
			
			// check for key begin
			if code != ScannedKeyBegin {
				// this will generate an appropriate error message
				PeekCheck(scanner, ScannedKeyBegin, ScannedStructEnd, ScannedMapEnd)
				return
			}
			
			// get the key
			key := scanner.NextValue()
			if key.Kind() != reflect.String {
				ErrorDecoding(ErrorPrint("Decoding", "Invalid key type %s", key.Type().String()))
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
