package json

import (
	"reflect"
	"strconv"
	
	"github.com/firelizzard18/gocoding"
)

func Scan(reader gocoding.SliceableRuneReader) gocoding.Scanner {
	return &scanner{gocoding.BasicErrorable{}, make([]gocoding.ScannerCode, 0, 5), reader, stateExpectingObjectOrArray, badMarkCode}
}

type scanState func(*scanner, gocoding.SliceableRuneReader, bool) (gocoding.ScannerCode, scanState)

func ErrorState(args...interface{}) scanState {
	return func(s *scanner, r gocoding.SliceableRuneReader, mark bool) (gocoding.ScannerCode, scanState) {
		s.Error(gocoding.ErrorPrint("Scanner", args...))
		return gocoding.ScannerError, nil
	}
}

func ErrorStatef(format string, args...interface{}) scanState {
	return func(s *scanner, r gocoding.SliceableRuneReader, mark bool) (gocoding.ScannerCode, scanState) {
		s.Error(gocoding.ErrorPrintf("Scanner", format, args...))
		return gocoding.ScannerError, nil
	}
}

type markCode uint8

const (
	badMarkCode markCode = iota
	markedString
	markedInt
	markedFloat
	markedBool
	markedNull
)

type scanner struct {
	gocoding.BasicErrorable
	
	stack []gocoding.ScannerCode
	runeReader gocoding.SliceableRuneReader
	step scanState
	mark markCode
}

func (s *scanner) Mark(code markCode) {
	s.mark = code
	s.runeReader.Mark()
}

func (s *scanner) Peek() gocoding.ScannerCode {
	if len(s.stack) == 0 {
		return gocoding.ScannerBadCode
	}
	
	return s.stack[len(s.stack) - 1]
}

func (s *scanner) NextCode() gocoding.ScannerCode {
	return s.nextCode(true)
}

func (s *scanner) Continue() gocoding.ScannerCode {
	return s._continue(true)
}

func (s *scanner) NextValue() reflect.Value {
	return s.nextValue(true)
}

func (s *scanner) NextString() string {
	s.runeReader.Mark()
	s.nextValue(false)
	return s.runeReader.Slice().String()
}

func (s *scanner) nextCode(mark bool) gocoding.ScannerCode {
	var code gocoding.ScannerCode
	
	code, s.step = s.step(s, s.runeReader, mark)
	
//	switch c := s.runeReader.Peek(); c {
//	case ' ', '\t', '\n', '\r':
//	default:
//		switch code {
//		case gocoding.Scanning, gocoding.ScannedLiteralEnd:
//			fmt.Print(string(c))
//			
//		case gocoding.ScannedToEnd:
//			fmt.Println("\nEOF\n")
//			
//		default:
//			fmt.Print("\n", code.String(), ":\t", string(c))
//		}
//	}
	
	if code == gocoding.ScannedToEnd {
		return code
	}
	
	switch code {
	case gocoding.Scanning:
		
	case gocoding.ScannerError:
		s.step(s, s.runeReader, mark)
		
	case gocoding.ScannedLiteralBegin:
		last := len(s.stack) - 1
		if last < 0 {
			s.Error(gocoding.ErrorPrint("Scanner", "Inconsistent state: found literal on base level"))
			return gocoding.ScannerError
		}
		
		switch top := s.stack[last]; top {
		case gocoding.ScannedStructBegin:
			code = gocoding.ScannedKeyBegin
			s.stack = append(s.stack, code)
			
		case gocoding.ScannedKeyEnd:
			s.stack[last] = code
			
		case gocoding.ScannedArrayBegin:
			s.stack = append(s.stack, code)
		
		default:
			s.Error(gocoding.ErrorPrintf("Scanner", "Inconsistent state: expecting struct/array begin or key end, got %s", top.String()))
			return gocoding.ScannerError
		}
		
	case gocoding.ScannedLiteralEnd:
		last := len(s.stack) - 1
		if last < 0 {
			s.Error(gocoding.ErrorPrint("Scanner", "Inconsistent state: found literal on base level"))
			return gocoding.ScannerError
		}
		
		switch top := s.stack[last]; top {
		case gocoding.ScannedKeyBegin:
			code = gocoding.ScannedKeyEnd
			s.stack[last], s.step = code, stateInObjectExpectingColon
			
		case gocoding.ScannedLiteralBegin:
			s.stack = s.stack[:last]
		
		default:
			code = gocoding.ScannerError
			s.Error(gocoding.ErrorPrintf("Scanner", "Inconsistent state: expecting literal or key begin, got %s", top.String()))
		}
		
	case gocoding.ScannedStructBegin, gocoding.ScannedArrayBegin:
		last := len(s.stack) - 1
		if last >= 0 && s.stack[last] == gocoding.ScannedKeyEnd {
			s.stack[last] = code
		} else {
			s.stack = append(s.stack, code)
		}
		
	case gocoding.ScannedStructEnd, gocoding.ScannedArrayEnd:
		idx := len(s.stack) - 1
		refl := s.stack[idx].Reflection()
		if code != refl {
			s.Error(gocoding.ErrorPrintf("Scanner", "Inconsistent state: expected %s, got %s", refl.String(), code.String()))
			return gocoding.ScannerError
		}
		s.stack = s.stack[:idx]
		
	case gocoding.ScannedToEnd:
	}
	
	return code
}

func (s *scanner) _continue(mark bool) gocoding.ScannerCode {
	next := s.nextCode(mark)
	for next == gocoding.Scanning { next = s.nextCode(mark) }
//	fmt.Println("Continued to ", next.String())
	return next
}

var interType = reflect.TypeOf(new(interface{})).Elem()
var arrayType = reflect.TypeOf([]interface{}{})
var mapType = reflect.TypeOf(map[string]interface{}{})

func (s *scanner) nextValue(mark bool) reflect.Value {
	// make sure there's a begin code on the stack
	for len(s.stack) == 0 {
		switch code := s.nextCode(mark); code {
		case gocoding.ScannerError, gocoding.ScannedToEnd:
			return reflect.ValueOf(nil)
		}
	}
	
	last := len(s.stack) - 1
	code := s.stack[last]
	
	switch code {
	case gocoding.ScannedKeyBegin:
		next := s._continue(mark)
		
		if next != gocoding.ScannedKeyEnd {
			s.Error(gocoding.ErrorPrintf("Scanner", "Scanning: expected %s, got %s", gocoding.ScannedKeyEnd.String(), gocoding.ScannedKeyEnd.String()))
			return reflect.ValueOf(nil)
		}
		
		if !mark { break }
		
		val, err := strconv.Unquote(s.runeReader.Slice().String())
		if err != nil {
			s.Error(gocoding.ErrorPrintf("Scanner", "Scanning: %s", err.Error()))
			return reflect.ValueOf(nil)
		}
		
		return reflect.ValueOf(val)
		
	case gocoding.ScannedLiteralBegin:
		next := s._continue(mark)
		
		if next != gocoding.ScannedLiteralEnd {
			s.Error(gocoding.ErrorPrintf("Scanner", "Scanning: expected %s, got %s", gocoding.ScannedLiteralEnd.String(), gocoding.ScannedLiteralEnd.String()))
			return reflect.ValueOf(nil)
		}
		
		if !mark { break }
		
		var err error
		var val interface{}
		
		str := s.runeReader.Slice().String()
		
		switch s.mark {
		case markedString:
			val, err = strconv.Unquote(str)
			
		case markedInt:
			val, err = strconv.ParseInt(str, 10, 64)
			
		case markedFloat:
			val, err = strconv.ParseFloat(str, 64)
			
		case markedBool:
			val, err = strconv.ParseBool(str)
			
		case markedNull:
			return reflect.Zero(interType)
			
		default:
			s.Error(gocoding.ErrorPrintf("Scanner", "Scanning: unexpected mark %d", s.mark))
			return reflect.ValueOf(nil)
		}
		
		if err != nil {
			s.Error(gocoding.ErrorPrintf("Scanner", "Scanning: %s", err.Error()))
			return reflect.ValueOf(nil)
		}
		
		return reflect.ValueOf(val)
		
	case gocoding.ScannedArrayBegin:
		if !mark {
			for len(s.stack) >= last {
				if s._continue(mark) == gocoding.ScannedArrayEnd { break }
				s.nextValue(mark)
			}
			break
		}
		
		array := reflect.MakeSlice(arrayType, 0, 3)
		
		for len(s.stack) >= last {
			if s._continue(mark) == gocoding.ScannedArrayEnd { break }
			val := s.nextValue(mark)
			if !val.IsValid() { break }
			array = reflect.Append(array, val)
		}
		
		return array
		
	case gocoding.ScannedStructBegin:
		if !mark {
			for len(s.stack) >= last {
				if s._continue(mark) == gocoding.ScannedStructEnd { break }
				s.nextValue(mark)
				
				s._continue(mark)
				s.nextValue(mark)
			}
			break
		}
		
		mapv := reflect.MakeMap(mapType)
		
		for len(s.stack) >= last {
			if s._continue(mark) == gocoding.ScannedStructEnd { break }
			key := s.nextValue(mark)
			if !key.IsValid() { break }
			
			s._continue(mark)
			val := s.nextValue(mark)
			if !val.IsValid() {
				s.Error(gocoding.ErrorPrintf("Scanner", "Scanning map: valid key %s but invalid value", key.Interface()))
				return reflect.ValueOf(nil)
			}
			
			mapv.SetMapIndex(key, val)
		}
		
		return mapv
		
	default:
		s.Error(gocoding.ErrorPrintf("Scanner", "Scanning: unexpected code %s", code.String()))
		return reflect.ValueOf(nil)
	}
	
	return reflect.ValueOf(nil)
}

// initial state
func stateExpectingObjectOrArray(s *scanner, r gocoding.SliceableRuneReader, mark bool) (gocoding.ScannerCode, scanState) {
	c := r.Next()
	
	if c == gocoding.EndOfText {
		return gocoding.ScannedToEnd, stateDone
	}
	
	switch c {
	case ' ', '\t', '\r', '\n':
		return gocoding.Scanning, stateExpectingObjectOrArray
		
	case '\u007B':
		return gocoding.ScannedStructBegin, stateInObjectExpectingKey
		
	case '[':
		return gocoding.ScannedArrayBegin, stateExpectingValue
		
	default:
		return gocoding.ScannerError, ErrorStatef(`Expecting \u007B or [, got %c`, c)
	}
}

func stateDone(s *scanner, r gocoding.SliceableRuneReader, mark bool) (gocoding.ScannerCode, scanState) {
	return gocoding.ScannedToEnd, stateDone
}

func stateExpectingValue(s *scanner, r gocoding.SliceableRuneReader, mark bool) (gocoding.ScannerCode, scanState) {
	c := r.Next()
	
	if c == gocoding.EndOfText {
		return gocoding.ScannedToEnd, stateDone
	}
	
	switch c {
	case ' ', '\t', '\r', '\n':
		return gocoding.Scanning, stateExpectingValue
		
	case '"':
		if mark { s.Mark(markedString) }
		return gocoding.ScannedLiteralBegin, stateInString
		
	case '-':
		if mark { s.Mark(markedInt) }
		return gocoding.ScannedLiteralBegin, stateInNumberNeg
		
	case '0':
		if mark { s.Mark(markedInt) }
		return gocoding.ScannedLiteralBegin, stateInNumber0
		
	case '1', '2', '3', '4', '5', '6', '7', '8', '9':
		if mark { s.Mark(markedInt) }
		return gocoding.ScannedLiteralBegin, stateInNumberDigit
		
		// putting { directly in a string breaks goclipse parsing
	case '\u007B':
		return gocoding.ScannedStructBegin, stateInObjectExpectingKey
		
	case '\u007D':
		return gocoding.ScannedStructEnd, stateInObjectOrArrayExpectingComma
		
	case '[':
		return gocoding.ScannedArrayBegin, stateExpectingValue
		
	case ']':
		return gocoding.ScannedArrayEnd, stateInObjectOrArrayExpectingComma
		
	case 't':
		if mark { s.Mark(markedBool) }
		return gocoding.ScannedLiteralBegin, stateInTrue
		
	case 'f':
		if mark { s.Mark(markedBool) }
		return gocoding.ScannedLiteralBegin, stateInFalse
		
	case 'n':
		if mark { s.Mark(markedNull) }
		return gocoding.ScannedLiteralBegin, stateInNull
		
	default:
		return gocoding.ScannerError, ErrorStatef(`Expecting ", -, 0-9, \u007B, [, t, f, or n, got %c`, c)
	}
}

func stateInObjectExpectingKey(s *scanner, r gocoding.SliceableRuneReader, mark bool) (gocoding.ScannerCode, scanState) {
	c := r.Next()
	
	if c == gocoding.EndOfText {
		return gocoding.ScannedToEnd, stateDone
	}
	
	switch c {
	case ' ', '\t', '\r', '\n':
		return gocoding.Scanning, stateInObjectExpectingKey
		
	case '"':
		if mark { r.Mark() }
		return gocoding.ScannedLiteralBegin, stateInString
		
	default:
		return gocoding.ScannerError, ErrorStatef(`Expecting ", got %c`, c)
	}
}

func stateInObjectExpectingColon(s *scanner, r gocoding.SliceableRuneReader, mark bool) (gocoding.ScannerCode, scanState) {
	c := r.Next()
	
	if c == gocoding.EndOfText {
		return gocoding.ScannedToEnd, stateDone
	}
	
	switch c {
	case ' ', '\t', '\r', '\n':
		return gocoding.Scanning, stateInObjectExpectingColon
		
	case ':':
		return gocoding.Scanning, stateExpectingValue
		
	default:
		return gocoding.ScannerError, ErrorStatef(`Expecting :, got %c`, c)
	}
}

func stateInObjectOrArrayExpectingComma(s *scanner, r gocoding.SliceableRuneReader, mark bool) (gocoding.ScannerCode, scanState) {
	c := r.Next()
	
	if c == gocoding.EndOfText {
		return gocoding.ScannedToEnd, stateDone
	}
	
	switch c {
	case ' ', '\t', '\r', '\n':
		return gocoding.Scanning, stateInObjectOrArrayExpectingComma
		
	case ',':
		return gocoding.Scanning, stateExpectingValue
		
	case '\u007D':
		return gocoding.ScannedStructEnd, stateInObjectOrArrayExpectingComma
		
	case ']':
		return gocoding.ScannedArrayEnd, stateInObjectOrArrayExpectingComma
		
	default:
		return gocoding.ScannerError, ErrorStatef(`Expecting ',', got %c`, c)
	}
}

func stateInString(s *scanner, r gocoding.SliceableRuneReader, mark bool) (gocoding.ScannerCode, scanState) {
	c := r.Next()
	
	if c == gocoding.EndOfText {
		return gocoding.ScannedToEnd, stateDone
	}
	
	switch c {
	case '"':
		return gocoding.ScannedLiteralEnd, stateInObjectOrArrayExpectingComma
		
	case '\\':
		return gocoding.Scanning, stateInStringEscaped
		
	default:
		return gocoding.Scanning, stateInString
	}
}

func stateInStringEscaped(s *scanner, r gocoding.SliceableRuneReader, mark bool) (gocoding.ScannerCode, scanState) {
	c := r.Next()
	
	if c == gocoding.EndOfText {
		return gocoding.ScannedToEnd, stateDone
	}
	
	switch c {
	case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
		return gocoding.Scanning, stateInString
		
	case 'u':
		return gocoding.Scanning, unicodeHexDigitNum(0).stateInStringUnicode
		
	default:
		return gocoding.ScannerError, ErrorStatef(`Expecting ", \, /, b, f, n, r, t, or u, got %c`, c)
	}
}

type unicodeHexDigitNum uint8

func (u unicodeHexDigitNum) stateInStringUnicode(s *scanner, r gocoding.SliceableRuneReader, mark bool) (gocoding.ScannerCode, scanState) {
	c := r.Next()
	
	if c == gocoding.EndOfText {
		return gocoding.ScannedToEnd, stateDone
	}
	
	switch c {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
		 'a', 'b', 'c', 'd', 'e', 'f', 'A', 'B', 'C', 'D', 'E', 'F':
		if u < 3 {
			return gocoding.Scanning, unicodeHexDigitNum(u+1).stateInStringUnicode
		} else {
			return gocoding.ScannedLiteralEnd, stateInString
		}
		
	default:
		return gocoding.ScannerError, ErrorStatef(`Expecting '0-9, a-f, or A-F, got %c`, c)
	}
}

func stateInNumberNeg(s *scanner, r gocoding.SliceableRuneReader, mark bool) (gocoding.ScannerCode, scanState) {
	c := r.Next()
	
	if c == gocoding.EndOfText {
		return gocoding.ScannedToEnd, stateDone
	}
	
	switch c {
	case '0':
		return gocoding.Scanning, stateInNumber0
		
	case '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return gocoding.Scanning, stateInNumberDigit
		
	default:
		return gocoding.ScannerError, ErrorStatef(`Expecting 0-9, got %c`, c)
	}
}

func stateInNumber0(s *scanner, r gocoding.SliceableRuneReader, mark bool) (gocoding.ScannerCode, scanState) {
	c := r.Next()
	
	if c == gocoding.EndOfText {
		return gocoding.ScannedToEnd, stateDone
	}
	
	switch c {
	case '.':
		s.mark = markedFloat
		return gocoding.Scanning, stateInNumberDot
	
	case 'e', 'E':
		s.mark = markedFloat
		return gocoding.Scanning, stateInNumberExponent
		
	default:
		r.Backup()
		return gocoding.ScannedLiteralEnd, stateInObjectOrArrayExpectingComma
	}
}

func stateInNumberDigit(s *scanner, r gocoding.SliceableRuneReader, mark bool) (gocoding.ScannerCode, scanState) {
	c := r.Next()
	
	if c == gocoding.EndOfText {
		return gocoding.ScannedToEnd, stateDone
	}
	
	switch c {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return gocoding.Scanning, stateInNumberDigit
		
	case '.':
		s.mark = markedFloat
		return gocoding.Scanning, stateInNumberDot
	
	case 'e', 'E':
		s.mark = markedFloat
		return gocoding.Scanning, stateInNumberExponent
		
	default:
		r.Backup()
		return gocoding.ScannedLiteralEnd, stateInObjectOrArrayExpectingComma
	}
}

func stateInNumberDot(s *scanner, r gocoding.SliceableRuneReader, mark bool) (gocoding.ScannerCode, scanState) {
	c := r.Next()
	
	if c == gocoding.EndOfText {
		return gocoding.ScannedToEnd, stateDone
	}
	
	switch c {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return gocoding.Scanning, stateInNumberPostDot
		
	default:
		return gocoding.ScannerError, ErrorStatef(`Expecting 0-9, got %c`, c)
	}
}

func stateInNumberPostDot(s *scanner, r gocoding.SliceableRuneReader, mark bool) (gocoding.ScannerCode, scanState) {
	c := r.Next()
	
	if c == gocoding.EndOfText {
		return gocoding.ScannedToEnd, stateDone
	}
	
	switch c {
	case 'e', 'E':
		return gocoding.Scanning, stateInNumberExponent
		
	default:
		r.Backup()
		return gocoding.ScannedLiteralEnd, stateInObjectOrArrayExpectingComma
	}
}

func stateInNumberExponent(s *scanner, r gocoding.SliceableRuneReader, mark bool) (gocoding.ScannerCode, scanState) {
	c := r.Next()
	
	if c == gocoding.EndOfText {
		return gocoding.ScannedToEnd, stateDone
	}
	
	switch c {
	case '+', '-':
		return gocoding.Scanning, stateInNumberSignedExponent
		
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return gocoding.Scanning, stateInNumberExponentDigit
		
	default:
		return gocoding.ScannerError, ErrorStatef(`Expecting +, -, or 0-9, got %c`, c)
	}
}

func stateInNumberSignedExponent(s *scanner, r gocoding.SliceableRuneReader, mark bool) (gocoding.ScannerCode, scanState) {
	c := r.Next()
	
	if c == gocoding.EndOfText {
		return gocoding.ScannedToEnd, stateDone
	}
	
	switch c {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return gocoding.Scanning, stateInNumberExponentDigit
		
	default:
		return gocoding.ScannerError, ErrorStatef(`Expecting +, -, or 0-9, got %c`, c)
	}
}

func stateInNumberExponentDigit(s *scanner, r gocoding.SliceableRuneReader, mark bool) (gocoding.ScannerCode, scanState) {
	c := r.Next()
	
	if c == gocoding.EndOfText {
		return gocoding.ScannedToEnd, stateDone
	}
	
	switch c {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return gocoding.Scanning, stateInNumberExponentDigit
		
	default:
		r.Backup()
		return gocoding.ScannedLiteralEnd, stateInObjectOrArrayExpectingComma
	}
}

func stateInTrue(s *scanner, r gocoding.SliceableRuneReader, mark bool) (code gocoding.ScannerCode, state scanState) {
	p, c := r.Peek(), r.Next()
	
	if c == gocoding.EndOfText {
		return gocoding.ScannedToEnd, stateDone
	}
	
	switch p {
	case 't':
		if c == 'r' {
			return gocoding.Scanning, stateInTrue
		}
	
	case 'r':
		if c == 'u' {
			return gocoding.Scanning, stateInTrue
		}
	
	case 'u':
		if c == 'e' {
			return gocoding.ScannedLiteralEnd, stateInObjectOrArrayExpectingComma
		}
		
	default:
		return gocoding.ScannerError, ErrorState(`Bad internal state`)
	}
	
	return gocoding.ScannerError, ErrorStatef(`Expecting 'true', got %c`, c)
}

func stateInFalse(s *scanner, r gocoding.SliceableRuneReader, mark bool) (code gocoding.ScannerCode, state scanState) {
	p, c := r.Peek(), r.Next()
	
	if c == gocoding.EndOfText {
		return gocoding.ScannedToEnd, stateDone
	}
	
	switch p {
	case 'f':
		if c == 'a' {
			return gocoding.Scanning, stateInFalse
		}
	
	case 'a':
		if c == 'l' {
			return gocoding.Scanning, stateInFalse
		}
	
	case 'l':
		if c == 's' {
			return gocoding.Scanning, stateInFalse
		}
	
	case 's':
		if c == 'e' {
			return gocoding.ScannedLiteralEnd, stateInObjectOrArrayExpectingComma
		}
		
	default:
		return gocoding.ScannerError, ErrorState(`Bad internal state`)
	}
	
	return gocoding.ScannerError, ErrorStatef(`Expecting 'false', got %c`, c)
}

func stateInNull(s *scanner, r gocoding.SliceableRuneReader, mark bool) (code gocoding.ScannerCode, state scanState) {
	p, c := r.Peek(), r.Next()
	
	if c == gocoding.EndOfText {
		return gocoding.ScannedToEnd, stateDone
	}
	
	switch p {
	case 'n':
		if c == 'u' {
			return gocoding.Scanning, stateInNull
		}
	
	case 'u':
		if c == 'l' {
			return gocoding.Scanning, stateInNull
		}
	
	case 'l':
		if c == 'l' {
			return gocoding.ScannedLiteralEnd, stateInObjectOrArrayExpectingComma
		}
		
	default:
		return gocoding.ScannerError, ErrorState(`Bad internal state`)
	}
	
	return gocoding.ScannerError, ErrorStatef(`Expecting 'null', got %c`, c)
}