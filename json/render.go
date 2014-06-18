package json

import (
	"errors"
	"fmt"
	"io"
	
	"github.com/firelizzard18/gocoding"
)

func RenderJSON(writer io.Writer) gocoding.Renderer {
	var ( fw gocoding.FormatWriter; ok bool )
	if fw, ok = writer.(gocoding.FormatWriter); !ok {
		fw = &gocoding.FWriter{writer}
	}
	
	s := &jsonRendererStack{FormatWriter: fw, renderers: make([]*jsonRenderer, 0, 10)}
	s.renderer = &jsonRenderer{s}
	return s
}

type jsonRendererStack struct {
	gocoding.FormatWriter
	renderers []*jsonRenderer
	renderer *jsonRenderer
}

func (s *jsonRendererStack) push() *jsonRenderer {
	s.renderer = &jsonRenderer{s}
	s.renderers = append(s.renderers, s.renderer)
	return s.renderer
}

func (s *jsonRendererStack) pushElement(id string) *jsonElementRenderer {
	return &jsonElementRenderer{s.push(), id}
}

func (s *jsonRendererStack) pushMap() *jsonMapRenderer {
	return &jsonMapRenderer{s.push(), false}
}

func (s *jsonRendererStack) pushArray() *jsonArrayRenderer {
	return &jsonArrayRenderer{s.push(), false}
}

func (s *jsonRendererStack) pop() {
	count := len(s.renderers)
	s.renderers, s.renderer = s.renderers[:count-1], s.renderers[count-1]
}

func (s *jsonRendererStack) WriteNil() (int, error) {
	return s.Print("null")
}

func (s *jsonRendererStack) StartElement(id string) (int, error) {
	return s.renderer.StartElement(id)
}

func (s *jsonRendererStack) StopElement(id string) (int, error) {
	return s.renderer.StopElement(id)
}

func (s *jsonRendererStack) StartStruct() (int, error) {
	return s.StartMap()
}

func (s *jsonRendererStack) StopStruct() (int, error) {
	return s.StopMap()
}

func (s *jsonRendererStack) StartMap() (int, error) {
	return s.pushMap().start()
}

func (s *jsonRendererStack) StopMap() (int, error) {
	return s.renderer.StopMap()
}

func (s *jsonRendererStack) StartArray() (int, error) {
	return s.pushArray().start()
}

func (s *jsonRendererStack) StopArray() (int, error) {
	return s.renderer.StopArray()
}

type jsonRenderer struct {
	*jsonRendererStack
}

func (r *jsonRenderer) start() (int, error) {
	return 0, errors.New("Operation not supported")
}

func (r *jsonRenderer) StartElement(id string) (int, error) {
	return 0, errors.New("Operation not supported")
}

func (r *jsonRenderer) StopElement(id string) (int, error) {
	return 0, errors.New("Operation not supported")
}

func (r *jsonRenderer) StopMap() (int, error) {
	return 0, errors.New("Operation not supported")
}

func (r *jsonRenderer) StopArray() (int, error) {
	return 0, errors.New("Operation not supported")
}

type jsonElementRenderer struct {
	*jsonRenderer
	id string
}

func (r *jsonElementRenderer) StopElement(id string) (int, error) {
	if id != r.id {
		return 0, errors.New(fmt.Sprintf("gocode/json.StopElement called on %s but current element id is %s", id, r.id))
	}
	
	return 0, nil
}

type jsonCollectionRenderer struct {
	*jsonRenderer
	comma bool
}

type jsonMapRenderer jsonCollectionRenderer

func (r *jsonMapRenderer) start() (int, error) {
	return r.Print(`{`)
}

func (r *jsonMapRenderer) StartElement(id string) (n int, err error) {
	if r.comma {
		n, err = r.Print(`,`)
	} else {
		r.comma = true
	}
	
	m, err := r.Printf(`"%s":`, id)
	n += m
	
	if err == nil {
		r.pushElement(id)
	}
	
	return
}

func (r *jsonMapRenderer) StopMap() (int, error) {
	return r.Print(`}`)
}

type jsonArrayRenderer jsonCollectionRenderer

func (r *jsonArrayRenderer) start() (int, error) {
	return r.Print(`[`)
}

func (r *jsonArrayRenderer) StartElement(id string) (n int, err error) {
	if r.comma {
		n, err = r.Print(`,`)
	} else {
		r.comma = true
	}
	
	if err == nil {
		r.pushElement(id)
	}
	
	return
}

func (r *jsonArrayRenderer) StopArray() (int, error) {
	return r.Print(`]`)
}









