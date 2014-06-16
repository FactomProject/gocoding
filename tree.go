package gocoding

import (

)

type Node struct {
	nodes map[string]*Node
}

type Raw []byte

func mkraw(data []byte) *Raw {
	r := new(Raw)
	*r = data
	return r
}

func (r *Raw) MarshalText() ([]byte, error) {
	return *r, nil
}

func (r *Raw) MarshalBinary() ([]byte, error) {
	return *r, nil
}

func (r *Raw) MarshalJSON() ([]byte, error) {
	return *r, nil
}

func (r *Raw) MarshalXML() ([]byte, error) {
	return *r, nil
}

func (r *Raw) UnmarshalText(data []byte) error {
	*r = append((*r)[0:0], data...)
	return nil
}

func (r *Raw) UnmarshalBinary(data []byte) error {
	*r = append((*r)[0:0], data...)
	return nil
}

func (r *Raw) UnmarshalJSON(data []byte) error {
	*r = append((*r)[0:0], data...)
	return nil
}

func (r *Raw) UnmarshalXML(data []byte) error {
	*r = append((*r)[0:0], data...)
	return nil
}