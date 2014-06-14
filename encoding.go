package gocoding

import (
	"errors"
	"fmt"
	"reflect"
	
	"encoding"
	"encoding/json"
	"encoding/xml"
)

type Field struct {
	Name string
	Value reflect.Value
	Type reflect.Type
}

type MarshalFunction func(interface{}) ([]byte, error)

type Marshaller interface {
	MarshallableFields() []Field
}

func getValue(f Field) (interface{}, error) {
	if f.Value.Kind() == reflect.Func {
		t := f.Value.Type()
		
		if t.NumIn() != 0 {
			return nil, errors.New(fmt.Sprint("Function valued marshallable field ", f.Name,
											  " should take no arguments but takes ", t.NumIn()))
		}
		
		if t.NumOut() < 1 {
			return nil, errors.New(fmt.Sprint("Function valued marshallable field ", f.Name,
											  " should return at least one value but returns ", t.NumOut()))
		}
		
		if f.Type != nil && !t.Out(0).ConvertibleTo(f.Type) {
			return nil, errors.New(fmt.Sprint("Function valued marshallable field ", f.Name,
											  "'s first return value should be of type ", f.Type,
											  " but is of type ", t.Out(0)))
		}
		
		if t.NumOut() > 2 {
			return nil, errors.New(fmt.Sprint("Function valued marshallable field ", f.Name,
											  " should return at most two values but returns ", t.NumOut()))
		}
		
		if t.NumOut() == 2 && t.Out(1) != reflect.TypeOf(errors.New("")) {
			return nil, errors.New(fmt.Sprint("Function valued marshallable field ", f.Name,
											  "'s second return value should be of type error but is of type ", t.Out(1)))
		}
		
		r := f.Value.Call([]reflect.Value{})
		
		if len(r) > 1 && !r[1].IsNil() {
			return nil, r[1].Interface().(error)
		}
		
		f.Value = r[0]
	}
	
	if f.Value.IsValid() {
		if f.Type != nil {
			f.Value = f.Value.Convert(f.Type)
		}
		return f.Value.Interface(), nil
	} else {
		return reflect.Zero(f.Type), nil
	}
}

func Marshal(mf MarshalFunction, v interface{}) ([]byte, error) {
	if _, ok := v.(Marshaller); !ok { return mf(v) }
	
	m := make(map[string]string)
	for _, f := range v.(Marshaller).MarshallableFields() {
		i, err := getValue(f)
		if err != nil { return nil, err }
		
		data, err := Marshal(mf, i)
		if err != nil { return nil, err }
		
		m[f.Name] = string(data)
	}
	
	return mf(m)
}

func MarshalJSON(v interface{}) ([]byte, error) {
	mf := json.Marshal
	
	if _, ok := v.(json.Marshaler); ok { return mf(v) }
	if _, ok := v.(encoding.TextMarshaler); ok { return mf(v) }
	
	return Marshal(mf, v)
}

func MarshalXML(v interface{}) ([]byte, error) {
	mf := xml.Marshal
	
	if _, ok := v.(xml.Marshaler); ok { return mf(v) }
	if _, ok := v.(encoding.TextMarshaler); ok { return mf(v) }
	
	return Marshal(mf, v)
}

type UnmarshalFunction func([]byte, interface{}) error

type Unmarshaller interface {
	UnmarshallableFields() []Field
}

func setField(f Field, v interface{}) error {
	r := reflect.ValueOf(v)
	
	if f.Type != nil {
		if !r.Type().ConvertibleTo(f.Type) {
			return errors.New(fmt.Sprint("Unmarshallable field is of type ", f.Type,
										 " but value is of type ", r.Type()))
		}
		r = r.Convert(f.Type)
	}
	
	if f.Value.Kind() == reflect.Func {
		t := f.Value.Type()
		
		if t.NumIn() != 1 {
			return errors.New(fmt.Sprint("Function valued unmarshallable field ", f.Name,
										 " should take one arguments but takes ", t.NumIn()))
		}
		
		if !r.Type().ConvertibleTo(t.In(0)) {
			return errors.New(fmt.Sprint("Function valued unmarshallable field ", f.Name,
										 "'s first argument is of type ", t.In(0),
										 " but value is of type ", r.Type()))
		}
		
		if t.NumOut() > 1 {
			return errors.New(fmt.Sprint("Function valued unmarshallable field ", f.Name,
										 " should return at most one value but returns ", t.NumOut()))
		}
		
		if t.NumOut() == 1 && t.Out(0).ConvertibleTo(reflect.TypeOf(errors.New(""))) {
			return errors.New(fmt.Sprint("Function valued unmarshallable field ", f.Name,
										 "'s first return value should be of type error but is of type ", t.Out(0)))
		}
		
		//r = r.Convert(t.In(0))
		e := f.Value.Call([]reflect.Value{r})
		
		if len(e) > 0 && !e[0].IsNil() {
			return e[0].Interface().(error)
		}
		
		return nil
	}
	
	if f.Value.Kind() == reflect.Ptr {
		f.Value = f.Value.Elem()
		
		if !f.Value.CanSet() {
			return errors.New(fmt.Sprint("Pointer valued unmarshallable field ", f.Name,
										 " should be settable but is not"))
		}
		
		if !r.Type().ConvertibleTo(f.Value.Type()) {
			return errors.New(fmt.Sprint("Pointer valued unmarshallable field ", f.Name,
										 " is of type ", f.Value.Type(),
										 " but value is of type ", r.Type()))
		}
		
		f.Value.Set(r)
		
		return nil
	}
	
	return errors.New(fmt.Sprint("Unmarshallable field is of kind ", f.Value.Kind(),
								 " but should be of kind Func or Ptr"))
}

func getSubclass(v interface{}, f Field) (interface{}, error) {
	r := reflect.ValueOf(v)
	
	if f.Type != nil {
		if !r.Type().ConvertibleTo(f.Type) {
			return nil, errors.New(fmt.Sprint("Subclass field is of type ", f.Type,
											  " but value is of type ", r.Type()))
		}
		r = r.Convert(f.Type)
	}
	
	if f.Value.Kind() == reflect.Func {
		t := f.Value.Type()
		
		if t.NumIn() != 1 {
			return nil, errors.New(fmt.Sprint("Function valued subclass field ", f.Name,
											  " should take one arguments but takes ", t.NumIn()))
		}
		
		if !r.Type().ConvertibleTo(t.In(0)) {
			return nil, errors.New(fmt.Sprint("Function valued subclass field ", f.Name,
											  "'s first argument is of type ", t.In(0),
											  " but value is of type ", r.Type()))
		}
		
		if t.NumOut() < 1 {
			return nil, errors.New(fmt.Sprint("Function valued subclass field ", f.Name,
											  " should return at least one value but returns ", t.NumOut()))
		}
		
		if f.Type != nil && !t.Out(0).ConvertibleTo(f.Type) {
			return nil, errors.New(fmt.Sprint("Function valued subclass field ", f.Name,
											  "'s first return value should be of type ", f.Type,
											  " but is of type ", t.Out(0)))
		}
		
		if t.NumOut() > 2 {
			return nil, errors.New(fmt.Sprint("Function valued subclass field ", f.Name,
											  " should return at most two values but returns ", t.NumOut()))
		}
		
		if t.NumOut() == 2 && t.Out(1) != reflect.TypeOf(errors.New("")) {
			return nil, errors.New(fmt.Sprint("Function valued subclass field ", f.Name,
											  "'s second return value should be of type error but is of type ", t.Out(1)))
		}
		
		r := f.Value.Call([]reflect.Value{reflect.ValueOf(v)})
		
		if len(r) > 1 && !r[1].IsNil() {
			return nil, r[1].Interface().(error)
		}
		
		f.Value = r[0]
	
		if f.Value.IsValid() {
			if f.Type != nil {
				f.Value = f.Value.Convert(f.Type)
			}
			return f.Value.Interface(), nil
		} else {
			return reflect.Zero(f.Type), nil
		}
	}
	
	return nil, errors.New(fmt.Sprint("Subclass field is of kind ", f.Value.Kind(),
								 " but should be of kind Func"))
}

func Unmarshal(uf UnmarshalFunction, d []byte, v interface{}) error {
	if _, ok := v.(Unmarshaller); !ok { return uf(d, v) }
	
	m := make(map[string]string)
	if err := uf(d, m); err != nil { return err }
	
loop:
	for _, f := range v.(Unmarshaller).UnmarshallableFields() {
		if f.Name == "" {
			i, err := getSubclass(v, f)
			if err != nil { return err }
			
			if _, ok := i.(Unmarshaller); !ok {
				return errors.New(fmt.Sprint("Subclass field does not implement gocoding.Unmarshaller"))
			}
			
			v = i
			
			goto loop
		}
		
		i := reflect.New(f.Type).Interface()
		s := m[f.Name]
		if err := Unmarshal(uf, []byte(s), i); err != nil { return err }
		if err := setField(f, i); err != nil { return err }
	}
	
	return nil
}

func UnmarshalJSON(d []byte, v interface{}) error {
	uf := json.Unmarshal
	
	if _, ok := v.(json.Unmarshaler); ok { return uf(d, v) }
	if _, ok := v.(encoding.TextUnmarshaler); ok { return uf(d, v) }
	
	return Unmarshal(uf, d, v)
}

func UnmarshalXML(d []byte, v interface{}) error {
	uf := xml.Unmarshal
	
	if _, ok := v.(xml.Unmarshaler); ok { return uf(d, v) }
	if _, ok := v.(encoding.TextUnmarshaler); ok { return uf(d, v) }
	
	return Unmarshal(uf, d, v)
}