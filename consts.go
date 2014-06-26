package gocoding

import (

)

const EndOfText rune = '\u0003'

const (
	Scanning ScannerCode = iota
	ScannedKeyBegin
	ScannedKeyEnd
	ScannedLiteralBegin
	ScannedLiteralEnd
	ScannedStructBegin
	ScannedStructEnd
	ScannedMapBegin
	ScannedMapEnd
	ScannedArrayBegin
	ScannedArrayEnd
	ScannerInitialized
	ScannedToEnd
	ScannerError
	ScannerBadCode
)

func (sc ScannerCode) String() string {
	switch sc {
	case Scanning:
		return "Scanning"
		
	case ScannedKeyBegin:
		return "ScannedKeyBegin"
		
	case ScannedKeyEnd:
		return "ScannedKeyEnd"
		
	case ScannedLiteralBegin:
		return "ScannedLiteralBegin"
		
	case ScannedLiteralEnd:
		return "ScannedLiteralEnd"
		
	case ScannedStructBegin:
		return "ScannedStructBegin"
		
	case ScannedStructEnd:
		return "ScannedStructEnd"
		
	case ScannedMapBegin:
		return "ScannedMapBegin"
		
	case ScannedMapEnd:
		return "ScannedMapEnd"
		
	case ScannedArrayBegin:
		return "ScannedArrayBegin"
		
	case ScannedArrayEnd:
		return "ScannedArrayEnd"
		
	case ScannerInitialized:
		return "ScannerInitialized"
		
	case ScannedToEnd:
		return "ScannedToEnd"
		
	case ScannerError:
		return "ScannerError"
		
	default:
		return "ScannerBadCode"
	}
}

func (sc ScannerCode) ScannedBegin() bool {
	switch sc {
	case ScannedKeyBegin, ScannedLiteralBegin, ScannedStructBegin, ScannedMapBegin, ScannedArrayBegin:
		return true
		
	default:
		return false
	}
}

func (sc ScannerCode) ScannedEnd() bool {
	switch sc {
	case ScannedKeyEnd, ScannedLiteralEnd, ScannedStructEnd, ScannedMapEnd, ScannedArrayEnd:
		return true
		
	default:
		return false
	}
}

func (sc ScannerCode) Reflection() ScannerCode {
	if sc.ScannedBegin() {
		return ScannerCode(sc + 1)
	}
	
	if sc.ScannedEnd() {
		return ScannerCode(sc - 1)
	}
	
	return ScannerBadCode
}

func (sc ScannerCode) Matches(codes...ScannerCode) bool {
	for _, code := range codes {
		if sc == code {
			return true
		}
	}
	return false
}
