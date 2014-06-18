package gocoding

import (
	"errors"
	"fmt"
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
}

func (m *marshaller) Marshal(obj interface{}) error {
	scratch := [64]byte{}
	value := reflect.ValueOf(obj)
	
	if !value.IsValid() {
		return errors.New("Invalid value")
	}
	
	encoder, err := m.FindEncoder(value.Type())
	if err != nil { return err }
	
	return encoder(scratch, m.renderer, value)
}

func (m *marshaller) FindEncoder(theType reflect.Type) (encoder Encoder, err error) {
	// check the cache
	m.RLock()
	encoder = m.cache[theType]
	m.RUnlock()
	if encoder != nil {
		return encoder, nil
	}
	
	switch theType.Kind() {
	case reflect.Array, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice, reflect.Struct:
		encoder, err = m.recurseSafeFindAndCacheEncoder(theType)
		
	case reflect.Bool, reflect.String,
		 reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		 reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		 /*reflect.Complex64, reflect.Complex128,*/
		 reflect.Float32, reflect.Float64:
		// simple types don't need locking
		encoder, err = m.encoding(m, theType)
		
	default:
		return nil, errors.New(fmt.Sprint("Unsupported type: ", theType))
	}
	
	if err != nil {
		encoder = nil
	}
	m.CacheEncoder(theType, encoder)
	
	return
}

func (m *marshaller) recurseSafeFindAndCacheEncoder(theType reflect.Type) (encoder Encoder, err error) {
	// to deal with recursive types, create a indirect encoder
	var wg sync.WaitGroup
	wg.Add(1)
	indirect := func(scratch [64]byte, renderer Renderer, value reflect.Value) error {
		wg.Wait()
		return encoder(scratch, renderer, value)
	}
	
	// safely add the indirect encoder
	m.CacheEncoder(theType, indirect)
	
	// find the encoder
	encoder, err = m.encoding(m, theType)
	
	// replace the encoder with one that returns an error so the indirect encoder doesn't explode
	if err != nil {
		encoder = func(scratch [64]byte, renderer Renderer, value reflect.Value) error {
			return errors.New("Creating the encoder failed")
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