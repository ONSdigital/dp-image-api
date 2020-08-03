package url

import "fmt"

// Builder encapsulates the building of urls in a central place, with knowledge of the url structures and base host names.
type Builder struct {
	apiURL string
}

// NewBuilder returns a new instance of url.Builder
func NewBuilder(apiURL string) *Builder {
	return &Builder{
		apiURL: apiURL,
	}
}

// BuildImageURL returns the website URL for a specific image
func (builder Builder) BuildImageURL(imageID string) string {
	return fmt.Sprintf("%s/images/%s",
		builder.apiURL, imageID)
}

// BuildImageDownloadURL returns the website URL for a specific image dowload variant
func (builder Builder) BuildImageDownloadURL(imageID, variant string) string {
	return fmt.Sprintf("%s/images/%s/downloads/%s",
		builder.apiURL, imageID, variant)
}
