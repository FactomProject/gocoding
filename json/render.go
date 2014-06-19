package json

import (
	"fmt"
	"io"
	//"reflect"
	"runtime"
	
	"github.com/firelizzard18/gocoding"
)

func RenderJSON(writer io.Writer) gocoding.Renderer {
	return &jsonRendererStack{Writer: writer, renderers: make([]gocoding.Renderer, 0, 10)}
}

type jsonRendererStack struct {
	io.Writer
	renderers []gocoding.Renderer
	
	handler func(*gocoding.Error)
	recovery func(interface{})
}

func (s *jsonRendererStack) push(r gocoding.Renderer) {
	//fmt.Println("pushing: ", reflect.TypeOf(r))
	s.renderers = append([]gocoding.Renderer{r}, s.renderers...)
}

func (s *jsonRendererStack) pop() {
	//fmt.Println("popping: ", reflect.TypeOf(s))
	s.renderers = s.renderers[1:]
}

func (s *jsonRendererStack) newElementRenderer(id string) *jsonElementRenderer {
	return &jsonElementRenderer{jsonRenderer{s}, id}
}

func (s *jsonRendererStack) newMapRenderer() *jsonMapRenderer {
	return &jsonMapRenderer{jsonRenderer{s}, false}
}

func (s *jsonRendererStack) newArrayRenderer() *jsonArrayRenderer {
	return &jsonArrayRenderer{jsonRenderer{s}, false}
}

func (s *jsonRendererStack) Write(data []byte) (int, error) {
	n, err := s.Writer.Write(data)
	
	if err != nil {
		s.Error(&gocoding.Error{"Writer", err})
	}
	
	return n, nil
}

func (s *jsonRendererStack) Print(args...interface{}) int {
	n, _ := fmt.Fprint(s, args...)
	return n
}

func (s *jsonRendererStack) Printf(format string, args...interface{}) int {
	n, _ := fmt.Fprintf(s, format, args...)
	return n
}

func (s *jsonRendererStack) WriteNil() int {
	return s.Print("null")
}

func (s *jsonRendererStack) Error(err *gocoding.Error) {
	if s.handler == nil {
		panic(fmt.Sprint(err.Class, ": ", err.Value))
	} else {
		s.handler(err)
	}
}

func (s *jsonRendererStack) Recover(err interface{}) error {
	if s.recovery == nil {
		switch err.(type) {
		case runtime.Error, string:
			panic(err)
			
		case error:
			return err.(error)
			
		default:
			panic(err)
		}
	} else {
		s.recovery(err)
	}
	
	return nil
}

/*func (s *jsonRendererStack) SetErrorHandler(handler func(gocoding.Error)) {
	s.handler = handler
}

func (s *jsonRendererStack) SetRecoverHandler(recovery func(interface{})) {
	s.recovery = recovery
}*/

func (s *jsonRendererStack) StartElement(id string) int {
	return s.renderers[0].StartElement(id)
}

func (s *jsonRendererStack) StopElement(id string) int {
	return s.renderers[0].StopElement(id)
}

func (s *jsonRendererStack) StartStruct() int {
	return s.StartMap()
}

func (s *jsonRendererStack) StopStruct() int {
	return s.StopMap()
}

func (s *jsonRendererStack) StartMap() int {
	return s.newMapRenderer().start()
}

func (s *jsonRendererStack) StopMap() int {
	return s.renderers[0].StopMap()
}

func (s *jsonRendererStack) StartArray() int {
	return s.newArrayRenderer().start()
}

func (s *jsonRendererStack) StopArray() int {
	return s.renderers[0].StopArray()
}

type jsonRenderer struct {
	*jsonRendererStack
}

func (r *jsonRenderer) StartElement(id string) int {
	r.Error(gocoding.ErrorPrint("JSON Renderer", "gocode/json.StartElement called on non-collection"))
	return 0
}

func (r *jsonRenderer) StopElement(id string) int {
	r.Error(gocoding.ErrorPrint("JSON Renderer", "gocode/json.StopElement called on non-element"))
	return 0
}

func (r *jsonRenderer) StopMap() int {
	r.Error(gocoding.ErrorPrint("JSON Renderer", "gocode/json.StopMap called on non-map"))
	return 0
}

func (r *jsonRenderer) StopArray() int {
	r.Error(gocoding.ErrorPrint("JSON Renderer", "gocode/json.StopArray called on non-array"))
	return 0
}

type jsonElementRenderer struct {
	jsonRenderer
	id string
}

func (r *jsonElementRenderer) start() int {
	r.push(r)
	return 0
}

func (r *jsonElementRenderer) StopElement(id string) int {
	if id != r.id {
		r.Error(gocoding.ErrorPrintf("JSON Renderer", "gocode/json.StopElement called on %s but current element id is %s", id, r.id))
	} else {
		r.pop()
	}
	
	return 0
}

type jsonCollectionRenderer struct {
	jsonRenderer
	comma bool
}

type jsonMapRenderer jsonCollectionRenderer

func (r *jsonMapRenderer) start() int {
	r.push(r)
	return r.Print(`{`)
}

func (r *jsonMapRenderer) StartElement(id string) (n int) {
	if r.comma {
		n = r.Print(`,`)
	} else {
		r.comma = true
	}
	
	n += r.Printf(`"%s":`, id)
	n += r.newElementRenderer(id).start()
	
	return
}

func (r *jsonMapRenderer) StopMap() int {
	n := r.Print(`}`)
	r.pop()
	return n
}

type jsonArrayRenderer jsonCollectionRenderer

func (r *jsonArrayRenderer) start() int {
	r.push(r)
	return r.Print(`[`)
}

func (r *jsonArrayRenderer) StartElement(id string) (n int) {
	if r.comma {
		n = r.Print(`,`)
	} else {
		r.comma = true
	}
	
	n += r.newElementRenderer(id).start()
	
	return
}

func (r *jsonArrayRenderer) StopArray() int {
	n := r.Print(`]`)
	r.pop()
	return n
}