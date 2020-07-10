package event

// ImageUploaded provides an avro structure for an image uploaded event
type ImageUploaded struct {
	Path    string `avro:"path"`
	ImageID string `avro:"image_id"`
}

// ImagePublished provides an avro structure for an image published event
type ImagePublished struct {
	SrcPath string `avro:"src_path"`
	DstPath string `avro:"dst_path"`
}
