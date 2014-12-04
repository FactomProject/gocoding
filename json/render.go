package json

import (
	"fmt"
	"github.com/FactomProject/gocoding"
	"io"
	"strconv"
)

func Render(writer io.Writer) gocoding.Renderer {
	return &jsonRendererStack{Writer: writer, renderers: make([]gocoding.Renderer, 0, 10)}
}

func RenderIndented(writer io.Writer, prefix, tabstr string) gocoding.Renderer {
	return &jsonRendererStack{Writer: writer, renderers: make([]gocoding.Renderer, 0, 10), indent: true, prefix: []string{prefix}, tabstr: tabstr}
}

type jsonRendererStack struct {
	gocoding.BasicErrorable
	io.Writer
	
	renderers []gocoding.Renderer
	
	indent bool
	prefix []string
	tabstr string
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
	n, _ := s.Write([]byte("null"))
	return n
}

func (s *jsonRendererStack) PrintString(str string) int {
	n, _ := s.Write([]byte(strconv.Quote(str)))
	return n
}

func (s *jsonRendererStack) writeIndent() {
	if !s.indent { return }
	
	s.Write([]byte{byte('\n')})
	for _, str := range s.prefix {
		s.Write([]byte(str))
	}
}

func (s *jsonRendererStack) pushIndent() {
	if !s.indent { return }
	
	s.prefix = append(s.prefix, s.tabstr)
}

func (s *jsonRendererStack) popIndent() {
	if !s.indent { return }
	
	s.prefix = s.prefix[:len(s.prefix)-1]
}

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
	r.pushIndent()
	return r.Print(`{`)
}

func (r *jsonMapRenderer) StartElement(id string) (n int) {
	if r.comma {
		n = r.Print(`,`)
	} else {
		r.comma = true
	}
	
	r.writeIndent()
	
	n += r.Printf(`"%s":`, id)
	n += r.newElementRenderer(id).start()
	
	return
}

func (r *jsonMapRenderer) StopMap() int {
	r.popIndent()
	r.writeIndent()
	n := r.Print(`}`)
	r.pop()
	return n
}

type jsonArrayRenderer jsonCollectionRenderer

func (r *jsonArrayRenderer) start() int {
	r.push(r)
//	r.pushIndent()
	return r.Print(`[`)
}

func (r *jsonArrayRenderer) StartElement(id string) (n int) {
	if r.comma {
		n = r.Print(`,`)
		r.writeIndent()
	} else {
		r.comma = true
	}
	
	n += r.newElementRenderer(id).start()
	
	return
}

func (r *jsonArrayRenderer) StopArray() int {
//	r.popIndent()
//	r.writeIndent()
	n := r.Print(`]`)
	r.pop()
	return n
}
