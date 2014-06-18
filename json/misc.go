package json

import (
	"io"
	
	"github.com/firelizzard18/gocoding"
)

func NewMarshaller(writer io.Writer) gocoding.Marshaller {
	return gocoding.NewMarshaller(gocoding.TextEncoding, RenderJSON(writer))
}