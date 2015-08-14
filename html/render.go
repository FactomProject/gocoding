package html

import (
	"fmt"
	"github.com/FactomProject/gocoding"
	"github.com/FactomProject/gocoding/text"
	"io"
)

func NewMarshaller() gocoding.Marshaller {
	return gocoding.NewMarshaller(text.Encoding)
}

func Marshal(writer io.Writer, obj interface{}) error {
	return NewMarshaller().Marshal(Render(writer), obj)
}

func Render(writer io.Writer) gocoding.Renderer {
	return &htmlRenderer{Writer: writer}
}

type htmlRenderer struct {
	gocoding.BasicErrorable
	io.Writer
}

func (r *htmlRenderer) Write(data []byte) (int, error) {
	n, err := r.Writer.Write(data)

	if err != nil {
		r.Error(&gocoding.Error{"Writer", err})
	}

	return n, nil
}

func (r *htmlRenderer) Print(args ...interface{}) int {
	n, _ := fmt.Fprint(r, args...)
	return n
}

func (r *htmlRenderer) Printf(format string, args ...interface{}) int {
	n, _ := fmt.Fprintf(r, format, args...)
	return n
}

func (r *htmlRenderer) WriteNil() int {
	n, _ := r.Write([]byte("null"))
	return n
}

func (r *htmlRenderer) PrintString(str string) int {
	n, _ := r.Write([]byte(str))
	return n
}

func (r *htmlRenderer) StartStruct() int {
	return r.Printf(`<ul class="struct">`)
}
func (r *htmlRenderer) StopStruct() int {
	return r.Printf(`</ul>`)
}

func (r *htmlRenderer) StartMap() int {
	return r.Printf(`<ul class="map">`)
}
func (r *htmlRenderer) StopMap() int {
	return r.Printf(`</ul>`)
}

func (r *htmlRenderer) StartArray() int {
	return r.Printf(`<ol class="array" start="0">`)
}
func (r *htmlRenderer) StopArray() int {
	return r.Printf(`</ol>`)
}

func (r *htmlRenderer) StartElement(id string) int {
	return r.Printf(`<li class="element"><span>%s: </span>`, id)
}
func (r *htmlRenderer) StopElement(id string) int {
	return r.Printf(`</li>`)
}
