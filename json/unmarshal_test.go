package json

import (
	"github.com/FactomProject/gocoding"
	"testing"
)

var unmarshaller gocoding.Unmarshaller

func init() {
	unmarshaller = NewUnmarshaller()
}

func test(json string, obj interface{}, t *testing.T) {
	scanner := Scan(gocoding.ReadString(json))
	err := unmarshaller.Unmarshal(scanner, obj)

	if err != nil {
		t.Error(err)
	}
}

func TestTypes(t *testing.T) {
	//	case reflect.Bool, reflect.Float32, reflect.Float64, reflect.String,
	//		 reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
	//		 reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
	//		return decoderType{theType}.decode
	s := `{"A": true, "B": 0, "C": 0, "D": 0, "E": 0, "F": 0, "G": 0, "H": 0, "I": 0, "J": 0, "K": 0, "L": 0, "M": 0, "N": "asdf"}`
	r := &struct {
		A bool
		B int
		C int8
		D int16
		E int32
		F int64
		G uint
		H uint8
		I uint16
		J uint32
		K uint64
		L float32
		M float64
		N string
	}{}
	test(s, r, t)
}

func TestStruct(t *testing.T) {
	s := `{"A": false}`
	r := struct{ A bool }{}
	test(s, &r, t)
}

func TestMap(t *testing.T) {
	s := `{"A": false, "B": "asdf", "C": {"1": 23.54, "2": null}}`
	r := map[string]interface{}{}
	test(s, &r, t)
}

func TestEmpty(t *testing.T) {
	var r interface{}

	r = map[string]interface{}{}
	test(`{}`, &r, t)

	r = []int{}
	test(`[]`, &r, t)

	r = struct{ A int }{}
	test(`{}`, &r, t)
}

func TestNull(t *testing.T) {
	r := new(struct{ A struct{ B string } })
	s := `{A: null}`
	test(s, r, t)
}
