package gocoding

import (
	"errors"
	"fmt"
	"reflect"
	
	"encoding/json"
)

var nv = reflect.ValueOf(nil)

type Field struct {
	Name string
	Value reflect.Value
	Type reflect.Type
}

func MakeField(name string, value interface{}, fType reflect.Type) Field {
	return Field{name, reflect.ValueOf(value), fType}
}

type Marshaller interface {
	MarshallableFields() []Field
}

func getValue(field Field) (interface{}, error) {
	if field.Value.Kind() == reflect.Func {
		fvType := field.Value.Type()
		
		if fvType.NumIn() != 0 {
			return nil, errors.New(fmt.Sprint("Function valued marshallable field ", field.Name,
											 " should take no arguments but takes ", fvType.NumIn()))
		}
		
		if fvType.NumOut() < 1 {
			return nil, errors.New(fmt.Sprint("Function valued marshallable field ", field.Name,
											 " should return at least one value but returns ", fvType.NumOut()))
		}
		
		if field.Type != nil && !fvType.Out(0).ConvertibleTo(field.Type) {
			return nil, errors.New(fmt.Sprint("Function valued marshallable field ", field.Name,
											 "'s first return value should be of type ", field.Type,
											 " but is of type ", fvType.Out(0)))
		}
		
		if fvType.NumOut() > 2 {
			return nil, errors.New(fmt.Sprint("Function valued marshallable field ", field.Name,
											 " should return at most two values but returns ", fvType.NumOut()))
		}
		
		if fvType.NumOut() == 2 && fvType.Out(1) != reflect.TypeOf(errors.New("")) {
			return nil, errors.New(fmt.Sprint("Function valued marshallable field ", field.Name,
											 "'s second return value should be of type error but is of type ", fvType.Out(1)))
		}
		
		rValue := field.Value.Call([]reflect.Value{})
		
		if len(rValue) > 1 && !rValue[1].IsNil() {
			return nil, rValue[1].Interface().(error)
		}
		
		field.Value = rValue[0]
	}
	
	if field.Value.IsValid() && field.Type != nil {
		field.Value = field.Value.Convert(field.Type)
	}
	
	if field.Value.IsValid() {
		return field.Value.Interface(), nil
	} else {
		return nil, nil
	}
}

func Marshal(value interface{}) ([]byte, error) {
	rValue := reflect.ValueOf(value)
	
	switch rValue.Kind() {
	case reflect.Slice, reflect.Array:
		count := rValue.Len()
		array := make([]*Raw, count)
		
		for i := 0; i < count; i++ {
			data, err := Marshal(rValue.Index(i).Interface())
			if err != nil { return nil, err }
			array[i] = mkraw(data)
		}
		
		return json.Marshal(array)
		
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Interface:
		if rValue.IsNil() {
			return json.Marshal(nil)
		}
		
	case reflect.Struct:
	}
	
	if _, ok := value.(Marshaller); !ok { return json.Marshal(value) }
	
	strMap := make(map[string]*Raw)
	for _, field := range value.(Marshaller).MarshallableFields() {
		obj, err := getValue(field)
		if err != nil { return nil, err }
		
		data, err := Marshal(obj)
		if err != nil { return nil, err }
		strMap[field.Name] = mkraw(data)
	}
	
	return json.Marshal(strMap)
}

type Unmarshaller interface {
	UnmarshallableFields() []Field
}

func setField(field Field, value interface{}) error {
	rValue := reflect.ValueOf(value)
	
	if field.Type != nil {
		if !rValue.Type().ConvertibleTo(field.Type) {
			return errors.New(fmt.Sprint("Unmarshallable field is of type ", field.Type,
										 " but value is of type ", rValue.Type()))
		}
		rValue = rValue.Convert(field.Type)
	}
	
	if field.Value.Kind() == reflect.Func {
		fvType := field.Value.Type()
		
		if fvType.NumIn() != 1 {
			return errors.New(fmt.Sprint("Function valued unmarshallable field ", field.Name,
										 " should take one arguments but takes ", fvType.NumIn()))
		}
		
		if !rValue.Type().ConvertibleTo(fvType.In(0)) {
			return errors.New(fmt.Sprint("Function valued unmarshallable field ", field.Name,
										 "'s first argument is of type ", fvType.In(0),
										 " but value is of type ", rValue.Type()))
		}
		
		if fvType.NumOut() > 1 {
			return errors.New(fmt.Sprint("Function valued unmarshallable field ", field.Name,
										 " should return at most one value but returns ", fvType.NumOut()))
		}
		
		if fvType.NumOut() == 1 && fvType.Out(0).ConvertibleTo(reflect.TypeOf(errors.New(""))) {
			return errors.New(fmt.Sprint("Function valued unmarshallable field ", field.Name,
										 "'s first return value should be of type error but is of type ", fvType.Out(0)))
		}
		
		//r = rValue.Convert(t.In(0))
		ret := field.Value.Call([]reflect.Value{rValue})
		
		if len(ret) > 0 && !ret[0].IsNil() {
			return ret[0].Interface().(error)
		}
		
		return nil
	}
	
	if field.Value.Kind() == reflect.Ptr {
		field.Value = field.Value.Elem()
		
		if !field.Value.CanSet() {
			return errors.New(fmt.Sprint("Pointer valued unmarshallable field ", field.Name,
										 " should be settable but is not"))
		}
		
		if !rValue.Type().ConvertibleTo(field.Value.Type()) {
			return errors.New(fmt.Sprint("Pointer valued unmarshallable field ", field.Name,
										 " is of type ", field.Value.Type(),
										 " but value is of type ", rValue.Type()))
		}
		
		field.Value.Set(rValue)
		
		return nil
	}
	
	return errors.New(fmt.Sprint("Unmarshallable field is of kind ", field.Value.Kind(),
								 " but should be of kind Func or Ptr"))
}

func getSubclass(field Field) (interface{}, error) {
	if field.Value.Kind() == reflect.Func {
		fvType := field.Value.Type()
		
		if fvType.NumIn() > 0 {
			return nil, errors.New(fmt.Sprint("Function valued subclass field ", field.Name,
											  " should take no arguments but takes ", fvType.NumIn()))
		}
		
		if fvType.NumOut() < 1 {
			return nil, errors.New(fmt.Sprint("Function valued subclass field ", field.Name,
											  " should return at least one value but returns ", fvType.NumOut()))
		}
		
		if field.Type != nil && !fvType.Out(0).ConvertibleTo(field.Type) {
			return nil, errors.New(fmt.Sprint("Function valued subclass field ", field.Name,
											  "'s first return value should be of type ", field.Type,
											  " but is of type ", fvType.Out(0)))
		}
		
		if fvType.NumOut() > 2 {
			return nil, errors.New(fmt.Sprint("Function valued subclass field ", field.Name,
											  " should return at most two values but returns ", fvType.NumOut()))
		}
		
		if fvType.NumOut() == 2 && fvType.Out(1) != reflect.TypeOf(errors.New("")) {
			return nil, errors.New(fmt.Sprint("Function valued subclass field ", field.Name,
											  "'s second return value should be of type error but is of type ", fvType.Out(1)))
		}
		
		ret := field.Value.Call([]reflect.Value{})
		
		if len(ret) > 1 && !ret[1].IsNil() {
			return nil, ret[1].Interface().(error)
		}
		
		field.Value = ret[0]
	
		if field.Value.IsValid() {
			if field.Type != nil {
				field.Value = field.Value.Convert(field.Type)
			}
			return field.Value.Interface(), nil
		} else {
			return reflect.Zero(field.Type), nil
		}
	}
	
	return nil, errors.New(fmt.Sprint("Subclass field is of kind ", field.Value.Kind(),
									  " but should be of kind Func"))
}

func Unmarshal(data []byte, value interface{}) error {
	rValue := reflect.ValueOf(value)
	fmt.Println("unmarshalling ", rValue.Type())
	
	if rValue.Kind() == reflect.Array || rValue.Kind() == reflect.Slice {
		eType := rValue.Type().Elem()
		array := []*Raw{}
		
		err := json.Unmarshal(data, array)
		if err != nil { return err }
		
		for index, str := range array {
			eValue := reflect.New(eType)
			
			err := Unmarshal(*str, eValue.Interface())
			if err != nil { return err }
			
			rValue.Index(index).Set(eValue)
		}
		
		return nil
	}
	
	switch rValue.Kind() {
	case reflect.Array, reflect.Slice:
		
	case reflect.Interface, reflect.Ptr, reflect.Struct:
		if _, ok := value.(Unmarshaller); !ok { return json.Unmarshal(data, value) }
		
	default:
		return json.Unmarshal(data, value)
	}
	
	strMap := make(map[string]*Raw)
	if err := json.Unmarshal(data, &strMap); err != nil { return err }
	
	fmt.Println(strMap)
	
loop:
	for _, field := range value.(Unmarshaller).UnmarshallableFields() {
		fmt.Println("field: ", field.Name)
		
		if field.Name == "" {
			subclass, err := getSubclass(field)
			if err != nil { return err }
			
			if _, ok := subclass.(Unmarshaller); !ok {
				return errors.New(fmt.Sprint("Subclass field does not implement gocoding.Unmarshaller"))
			}
			
			value = subclass
			
			goto loop
		}
		
		obj := reflect.New(field.Type).Elem()
		data := strMap[field.Name]
		if err := Unmarshal(*data, obj.Interface()); err != nil { return err }
		if err := setField(field, obj.Interface()); err != nil { return err }
	}
	
	return nil
}