package gocoding

import (
	"reflect"
	"sync"
)

func NewMarshaller(encoding Encoding, renderer Renderer) Marshaller {
	return &marshaller{encoding: encoding, renderer: renderer, cache: make(map[reflect.Type]Encoder)}
}

type marshaller struct {
	encoding Encoding
	renderer Renderer
	
	sync.RWMutex
	cache map[reflect.Type]Encoder
	
	scratch [64]byte
}

func (m *marshaller) Marshal(obj interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = m.renderer.Recover(r)
		}
	}()
	
	m.MarshalObject(obj)
	
	return
}

func (m *marshaller) MarshalObject(obj interface{}) {
	m.MarshalValue(reflect.ValueOf(obj))
}

func (m *marshaller) MarshalValue(value reflect.Value) {
	if !value.IsValid() {
		m.renderer.Error(ErrorPrint("Marshalling", "Invalid value"))
		return
	}
	
	encoder := m.FindEncoder(value.Type())
	if encoder == nil { return }
	
	encoder(m.scratch, m.renderer, value)
}

func (m *marshaller) FindEncoder(theType reflect.Type) (encoder Encoder) {
	// check the cache
	m.RLock()
	encoder = m.cache[theType]
	m.RUnlock()
	if encoder != nil {
		return encoder
	}
	
	switch theType.Kind() {
	case reflect.Array, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice, reflect.Struct:
		encoder = m.recurseSafeFindAndCacheEncoder(theType)
		
	case reflect.Bool, reflect.String,
		 reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		 reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		 /*reflect.Complex64, reflect.Complex128,*/
		 reflect.Float32, reflect.Float64:
		// simple types don't need locking
		encoder = m.encoding(m, theType)
		
	default:
		m.renderer.Error(ErrorPrint("Encoding", "Unsupported type: ", theType))
	}
	
	m.CacheEncoder(theType, encoder)
	
	return encoder
}

func (m *marshaller) recurseSafeFindAndCacheEncoder(theType reflect.Type) (encoder Encoder) {
	// to deal with recursive types, create a indirect encoder
	var wg sync.WaitGroup
	wg.Add(1)
	indirect := func(scratch [64]byte, renderer Renderer, value reflect.Value) {
		wg.Wait()
		encoder(scratch, renderer, value)
	}
	
	// safely add the indirect encoder
	m.CacheEncoder(theType, indirect)
	
	// find the encoder
	encoder = m.encoding(m, theType)
	
	// replace the encoder with one that returns an error so the indirect encoder doesn't explode
	if encoder == nil {
		encoder = func(scratch [64]byte, renderer Renderer, value reflect.Value) {
			m.renderer.Error(ErrorPrint("Encoding", "Unsupported type: ", theType))
		}
	}
	
	// unblock the indirect encoder
	wg.Done()
	
	return
}

func (m *marshaller) CacheEncoder(theType reflect.Type, encoder Encoder) {
	m.Lock()
	m.cache[theType] = encoder
	m.Unlock()
}