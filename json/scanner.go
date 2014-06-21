package json

import (
	"github.com/firelizzard18/gocoding"
)

type scanner struct {
	scannerState []gocoding.ScannerCode
}

func (s *scanner) Error(err *gocoding.Error) {
	panic(err)
}

type scanState func(s scanner, r gocoding.SliceableRuneReader) (gocoding.ScannerCode, scanState)

func ErrorState(args...interface{}) scanState {
	return func(s scanner, r gocoding.SliceableRuneReader) (gocoding.ScannerCode, scanState) {
		s.Error(gocoding.ErrorPrint("Scanner", args...))
		return gocoding.ScannerError, nil
	}
}

func ErrorStatef(format string, args...interface{}) scanState {
	return func(s scanner, r gocoding.SliceableRuneReader) (gocoding.ScannerCode, scanState) {
		s.Error(gocoding.ErrorPrintf("Scanner", format, args...))
		return gocoding.ScannerError, nil
	}
}

// initial state
func stateExpectingObjectOrArray(s scanner, r gocoding.SliceableRuneReader) (gocoding.ScannerCode, scanState) {
	switch c := r.Read(); c {
	case ' ', '\t', '\r', '\n':
		return gocoding.Scanning, stateExpectingObjectOrArray
		
	case '{':
		return gocoding.ScannedStructBegin, stateExpectingString
		
	case '[':
		return gocoding.ScannedArrayBegin, stateExpectingValue
		
	default:
		return gocoding.ScannerError, ErrorStatef(`Expecting { or [, got %q`, c)
	}
}

func stateExpectingString(s scanner, r gocoding.SliceableRuneReader) (gocoding.ScannerCode, scanState) {
	switch c := r.Read(); c {
	case ' ', '\t', '\r', '\n':
		return gocoding.Scanning, stateExpectingString
		
	case '"':
		r.Mark()
		return gocoding.ScannedLiteralBegin, stateInString
		
	default:
		return gocoding.ScannerError, ErrorStatef(`Expecting ", got %q`, c)
	}
}

func stateExpectingValue(s scanner, r gocoding.SliceableRuneReader) (gocoding.ScannerCode, scanState) {
	switch c := r.Read(); c {
	case ' ', '\t', '\r', '\n':
		return gocoding.Scanning, stateExpectingString
		
	case '"':
		r.Mark()
		return gocoding.ScannedLiteralBegin, stateInString
		
	case '-':
		r.Mark()
		return gocoding.ScannedLiteralBegin, stateInNumberNeg
		
	case '0':
		r.Mark()
		return gocoding.ScannedLiteralBegin, stateInNumber0
		
	case '1', '2', '3', '4', '5', '6', '7', '8', '9':
		r.Mark()
		return gocoding.ScannedLiteralBegin, stateInNumber1
		
	case '{':
		return gocoding.ScannedStructBegin, stateExpectingString
		
	case '[':
		return gocoding.ScannedArrayBegin, stateExpectingValue
		
	case 't':
		return gocoding.ScannedLiteralBegin, stateInTrueT
		
	case 'f':
		return gocoding.ScannedLiteralBegin, stateInFalseF
		
	case 'n':
		return gocoding.ScannedLiteralBegin, stateInNullN
		
	default:
		return gocoding.ScannerError, ErrorStatef(`Expecting ", -, 0-9, {, [, t, f, or n, got %q`, c)
	}
}








func stateInNumberNeg(s scanner, r gocoding.SliceableRuneReader) (gocoding.ScannerCode, scanState) {
	switch c := r.Read(); c {
	case '"':
		r.Mark()
		return gocoding.ScannedLiteralBegin, stateInString
		
	default:
		return gocoding.ScannerError, ErrorStatef(`Expecting '"', got %q`, c)
	}
}











