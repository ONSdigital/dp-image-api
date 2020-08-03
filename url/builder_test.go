package url_test

import (
	"fmt"
	"github.com/ONSdigital/dp-image-api/url"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

const (
	websiteURL      = "localhost:20000"
	imageID         = "123"
	downloadVariant = "640bw"
)

func TestBuilder_BuildWebsiteDatasetVersionURL(t *testing.T) {

	Convey("Given a URL builder", t, func() {

		urlBuilder := url.NewBuilder(websiteURL)

		Convey("When BuildImageURL is called", func() {

			url := urlBuilder.BuildImageURL(imageID)

			expectedURL := fmt.Sprintf("%s/images/%s",
				websiteURL, imageID)

			Convey("Then the expected URL is returned", func() {
				So(url, ShouldEqual, expectedURL)
			})
		})

		Convey("When BuildImageDownloadURL is called", func() {

			url := urlBuilder.BuildImageDownloadURL(imageID, downloadVariant)

			expectedURL := fmt.Sprintf("%s/images/%s/downloads/%s",
				websiteURL, imageID, downloadVariant)

			Convey("Then the expected URL is returned", func() {
				So(url, ShouldEqual, expectedURL)
			})
		})
	})
}
