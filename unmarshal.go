package gocoding

import (
	"reflect"
	"sync"
)

func NewUnmarshaller(decoding Decoding) Unmarshaller {
	return &unmarshaller{decoding: decoding, cache: make(map[reflect.Type]Decoder)}
}

type unmarshaller struct {
	decoding Decoding
	
	sync.RWMutex
	cache map[reflect.Type]Decoder
	
	scratch [64]byte
}

func (u *unmarshaller) Unmarshal(scanner Scanner, obj interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = scanner.Recover(r)
		}
	}()
	
	scanner.Continue()
	u.UnmarshalObject(scanner, obj)
	return
}

func (u *unmarshaller) UnmarshalObject(scanner Scanner, obj interface{}) {
	u.UnmarshalValue(scanner, reflect.ValueOf(obj))
}

func (u *unmarshaller) UnmarshalValue(scanner Scanner, value reflect.Value) {
	// asdf valid? not pointer?
	value = NormalizeValue(scanner, "Unmarshalling", value)
	
	decoder := u.FindDecoder(value.Type())
	if decoder == nil { return }
	
	decoder(u.scratch, scanner, value)
}

func (u *unmarshaller) FindDecoder(theType reflect.Type) (decoder Decoder) {
	// check the cache
	u.RLock()
	decoder = u.cache[theType]
	u.RUnlock()
	if decoder != nil {
		return decoder
	}
	
	switch theType.Elem().Kind() {
	case reflect.Array, reflect.Interface, reflect.Map, reflect.Slice, reflect.Struct:
		decoder = u.recurseSafeFindAndCacheDecoder(theType)
		
	case reflect.Bool, reflect.String,
		 reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		 reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		 /*reflect.Complex64, reflect.Complex128,*/
		 reflect.Float32, reflect.Float64:
		// simple types don't need locking
		decoder = u.decoding(u, theType)
		
	default:
		panic(ErrorPrint("Decoding", "Unsupported type: ", theType))
	}
	
	u.CacheDecoder(theType, decoder)
	
	return decoder
}

func (u *unmarshaller) recurseSafeFindAndCacheDecoder(theType reflect.Type) (decoder Decoder) {
	// to deal with recursive types, create an indirect decoder
	var wg sync.WaitGroup
	wg.Add(1)
	indirect := func(scratch [64]byte, scanner Scanner, value reflect.Value) {
		wg.Wait()
		decoder(scratch, scanner, value)
	}
	
	// safely add the indirect decoder
	u.CacheDecoder(theType, indirect)
	
	// find the decoder
	decoder = u.decoding(u, theType)
	
	// replace the decoder with one that returns an error so the indirect decoder doesn't explode
	if decoder == nil {
		decoder = func([64]byte, Scanner, reflect.Value) {
			panic(ErrorPrint("Decoding", "Unsupported type: ", theType))
		}
	}
	
	// unblock the indirect encoder
	wg.Done()
	return
}

func (u *unmarshaller) IsCached(theType reflect.Type) bool {
	_, ok := u.cache[theType]
	return ok
}

func (u *unmarshaller) CacheDecoder(theType reflect.Type, decoder Decoder) {
	u.Lock()
	u.cache[theType] = decoder
	u.Unlock()
}