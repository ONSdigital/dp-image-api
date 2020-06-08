package schema

import (
	"github.com/ONSdigital/go-ns/avro"
)

var imageUploadedEvent = `{
  "type": "record",
  "name": "image-uploaded",
  "fields": [
    {"name": "image_id", "type": "string", "default": ""},
    {"name": "path", "type": "string", "default": ""}
  ]
}`

// ImageUploadedEvent the Avro schema for Image uploaded messages.
var ImageUploadedEvent = &avro.Schema{
	Definition: imageUploadedEvent,
}
