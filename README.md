# gocoding

v1.2.1

Gocoding is designed to provide a framework for creating flexible marshallers
and unmarshallers. It is designed for flexibility and modularity, not
scalability. It has not (yet) been optimized for speed and it can only handle
string lengths up to 1022 characters (not including quotes).

## Change log

### v1.2.1

Fixed bugs

Fixed some decoding bugs

### v1.2

HTML

Added an HTML renderer such that Golang variables can be marshalled into HTML.
The project has no interest in supporting a conversion from HTML into Golang.

### v1.1

Hotfix

Fixed bugs

### v1.0

Initial version

Marshaller & unmarshaller framework for encoder caching
Text encoders & decoders
JSON renderer & scanner

Complete Golang <=> JSON conversion.
