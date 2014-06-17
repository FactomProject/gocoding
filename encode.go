package gocoding

import (
	"reflect"
)

type Marshaller struct {
	Encoding
	Renderer
	
	cache map[reflect.Type]Encoder
}

func Marshal(encoding Encoding, renderer Renderer, obj interface{}) error {
	marshaller := &Marshaller{encoding, renderer, make(map[reflect.Type]Encoder)}
	
	scratch := [64]byte{}
	value := reflect.ValueOf(obj)
	
	encoder, err := marshaller.FindEncoder(encoding, value.Type())
	if err != nil { return err }
	
	return encoder(scratch, renderer, value)
}

func (m *Marshaller) FindEncoder(encoding Encoding, t reflect.Type) (Encoder, error) {
	// check the cache
	if encoder, ok := m.cache[t]; ok {
		return encoder, nil
	}
	
	// cache hit failed, find it in the encoding, cache it
	encoder, err := encoding(m.Encoding, t)
	if err != nil { return nil, err }
	
	m.cache[t] = encoder
	return encoder, nil
}