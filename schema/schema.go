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

var imagePublishedEvent = `{
  "type": "record",
  "name": "image-published",
  "fields": [
    {
      "name": "downloads",
      "type": {
        "type": "array",
        "items": {
          "type": "record",
          "name": "download-variant",
          "fields": [
            {"name": "src_path", "type": "string", "default": ""},
            {"name": "dst_path", "type": "string", "default": ""}
          ]
        }
      }
    }
  ]
}`

// ImageUploadedEvent is the Avro schema for Image uploaded messages.
var ImageUploadedEvent = &avro.Schema{
	Definition: imageUploadedEvent,
}

// ImagePublishedEvent is the Avro schema for Image published messages.
var ImagePublishedEvent = &avro.Schema{
	Definition: imagePublishedEvent,
}
