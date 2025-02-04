package url_test

import (
	"fmt"
	"testing"

	"github.com/ONSdigital/dp-image-api/url"
	"github.com/smartystreets/goconvey/convey"
)

const (
	websiteURL      = "localhost:20000"
	imageID         = "123"
	downloadVariant = "640bw"
)

func TestBuilder_BuildWebsiteDatasetVersionURL(t *testing.T) {
	convey.Convey("Given a URL builder", t, func() {
		urlBuilder := url.NewBuilder(websiteURL)

		convey.Convey("When BuildImageURL is called", func() {
			url := urlBuilder.BuildImageURL(imageID)

			expectedURL := fmt.Sprintf("%s/images/%s",
				websiteURL, imageID)

			convey.Convey("Then the expected URL is returned", func() {
				convey.So(url, convey.ShouldEqual, expectedURL)
			})
		})

		convey.Convey("When BuildImageDownloadsURL is called", func() {
			url := urlBuilder.BuildImageDownloadsURL(imageID)

			expectedURL := fmt.Sprintf("%s/images/%s/downloads",
				websiteURL, imageID)

			convey.Convey("Then the expected URL is returned", func() {
				convey.So(url, convey.ShouldEqual, expectedURL)
			})
		})

		convey.Convey("When BuildImageDownloadURL is called", func() {
			url := urlBuilder.BuildImageDownloadURL(imageID, downloadVariant)

			expectedURL := fmt.Sprintf("%s/images/%s/downloads/%s",
				websiteURL, imageID, downloadVariant)

			convey.Convey("Then the expected URL is returned", func() {
				convey.So(url, convey.ShouldEqual, expectedURL)
			})
		})
	})
}
