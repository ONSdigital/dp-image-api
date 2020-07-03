package models_test

import (
	"testing"

	"github.com/ONSdigital/dp-image-api/apierrors"
	"github.com/ONSdigital/dp-image-api/models"
	. "github.com/smartystreets/goconvey/convey"
)

func TestImagesRefresh(t *testing.T) {
	Convey("Given an images struct with an image that contains a download variant", t, func() {
		images := models.Images{
			Count:  1,
			Offset: 0,
			Limit:  1,
			Items: []models.Image{
				{
					ID:       "imageID",
					Filename: "myImage.png",
					State:    models.StateCompleted.String(),
					Downloads: map[string]models.Download{
						"original": {
							State: models.StateDownloadCompleted.String(),
						},
					},
				},
			},
		}
		Convey("Then, refreshing the 'images' refreshes the image and the download variant with the expected values", func() {
			images.Refresh()
			So(images, ShouldResemble, models.Images{
				Count:  1,
				Offset: 0,
				Limit:  1,
				Items: []models.Image{
					{
						ID:       "imageID",
						Filename: "myImage.png",
						State:    models.StateCompleted.String(),
						Downloads: map[string]models.Download{
							"original": {
								State:  models.StateDownloadCompleted.String(),
								Href:   "http://static.ons.gov.uk/images/imageID/original/myImage.png",
								Public: true,
							},
						},
					},
				},
			})
		})
	})
}

func TestImageValidation(t *testing.T) {

	Convey("Given an empty image, it is successfully validated", t, func() {
		image := models.Image{}
		err := image.Validate()
		So(err, ShouldBeNil)
	})

	Convey("Given an image with a filename that is longer than the maximum allowed, it fails to validate with the expected error", t, func() {
		image := models.Image{
			Filename: "Llanfairpwllgwyngyllgogerychwyrndrobwllllantysiliogogogoch",
		}
		err := image.Validate()
		So(err, ShouldResemble, apierrors.ErrImageFilenameTooLong)
	})

	Convey("Given an image with a state that does not correspond to any expected state, it fails to validate with the expected error", t, func() {
		image := models.Image{
			State: "wrong",
		}
		err := image.Validate()
		So(err, ShouldResemble, apierrors.ErrImageInvalidState)
	})

	Convey("Given an image with an download in an invalid state, it fails to validate with the expected error", t, func() {
		image := models.Image{
			State: models.StatePublished.String(),
			Downloads: map[string]models.Download{
				"original": {
					State: "wrong",
				},
			},
		}
		err := image.Validate()
		So(err, ShouldResemble, apierrors.ErrImageDownloadInvalidState)
	})

	Convey("Given a fully populated valid image with a valid download variant, it is successfully validated", t, func() {
		image := models.Image{
			ID:           "123",
			CollectionID: "456",
			State:        models.StatePublished.String(),
			Filename:     "image-file-name",
			License: &models.License{
				Title: "testLicense",
				Href:  "http://testLicense.co.uk",
			},
			Upload: &models.Upload{
				Path: "image-upload-path",
			},
			Type: "icon",
			Downloads: map[string]models.Download{
				"original": {
					State: models.StateDownloadCompleted.String(),
				},
			},
		}
		err := image.Validate()
		So(err, ShouldBeNil)
	})
}

func TestImageRefresh(t *testing.T) {
	Convey("Given a fully populated valid image with a valid download variant, then refreshing the image results in the download variant being refreshed", t, func() {

		image := models.Image{
			ID:       "imageID",
			Filename: "myImage.png",
			State:    models.StateCompleted.String(),
			Downloads: map[string]models.Download{
				"original": {
					State: models.StateDownloadCompleted.String(),
				},
			},
		}
		Convey("Then, refreshing the image refreshes the download variant with the expected values", func() {
			image.Refresh()
			So(image, ShouldResemble, models.Image{
				ID:       "imageID",
				Filename: "myImage.png",
				State:    models.StateCompleted.String(),
				Downloads: map[string]models.Download{
					"original": {
						State:  models.StateDownloadCompleted.String(),
						Href:   "http://static.ons.gov.uk/images/imageID/original/myImage.png",
						Public: true,
					},
				},
			})
		})
	})
}

func TestImageStateTransitionAllowed(t *testing.T) {
	Convey("Given an image in created state", t, func() {
		image := models.Image{
			State: models.StateCreated.String(),
		}
		validateTransitionsToCreated(image)
	})

	Convey("Given an image with a wrong state value, then no transition is allowed", t, func() {
		image := models.Image{State: "wrong"}
		validateTransitionsToCreated(image)
	})

	Convey("Given an image without state, then created state is assumed when checking for transitions", t, func() {
		image := models.Image{}
		validateTransitionsToCreated(image)
	})
}

// validateTransitionsToCreated validates that the provided image can transition to created state,
// and not to any forbidden of invalid state
func validateTransitionsToCreated(image models.Image) {
	Convey("Then an allowed transition is successfully checked", func() {
		So(image.StateTransitionAllowed(models.StateUploaded.String()), ShouldBeTrue)
	})
	Convey("Then a forbidden transition to a valid state is not allowed", func() {
		So(image.StateTransitionAllowed(models.StatePublished.String()), ShouldBeFalse)
	})
	Convey("Then a transition to an invalid state is not allowed", func() {
		So(image.StateTransitionAllowed("wrong"), ShouldBeFalse)
	})
}

func TestDownloadValidation(t *testing.T) {

	Convey("Given an empty download variant, it is successfully validated", t, func() {
		download := models.Download{}
		err := download.Validate()
		So(err, ShouldBeNil)
	})

	Convey("Given a download variant with an invalid state name, it fails to validate with the expected error", t, func() {
		download := models.Download{
			State: "wrong",
		}
		err := download.Validate()
		So(err, ShouldResemble, apierrors.ErrImageDownloadInvalidState)
	})

	Convey("Given a download variant with a valid state name, it is successfully validated", t, func() {
		download := models.Download{
			State: models.StateDownloadImported.String(),
		}
		err := download.Validate()
		So(err, ShouldBeNil)
	})
}

func TestDownloadRefresh(t *testing.T) {

	Convey("Given an array of download variants in any state except completed", t, func() {
		downloads := []models.Download{
			{State: models.StateDownloadPending.String()},
			{State: models.StateDownloadImporting.String()},
			{State: models.StateDownloadImported.String()},
			{State: models.StateDownloadPublished.String()},
			{State: models.StateDownloadFailed.String()},
		}

		Convey("Then, refreshing the models results in public being false, and the expected href", func() {
			for _, download := range downloads {
				state := download.State
				download.Refresh("imageID", "png_bw", "imageName.png")
				So(download, ShouldResemble, models.Download{
					State:  state,
					Href:   "http://download.ons.gov.uk/images/imageID/png_bw/imageName.png",
					Public: false,
				})
			}
		})
	})

	Convey("Given a download variant in completed state", t, func() {
		download := models.Download{State: models.StateCompleted.String()}
		Convey("Then, refreshing the model results in public being true, and the expected href", func() {
			download.Refresh("imageID", "png_bw", "imageName.png")
			So(download, ShouldResemble, models.Download{
				State:  models.StateCompleted.String(),
				Href:   "http://static.ons.gov.uk/images/imageID/png_bw/imageName.png",
				Public: true,
			})
		})
	})
}

func TestDownloadStateTransitionAllowed(t *testing.T) {
	Convey("Given an image download variant in pending state", t, func() {
		download := models.Download{
			State: models.StateDownloadPending.String(),
		}
		validateDownloadTransitionsToPending(download)
	})

	Convey("Given an image download varian with a wrong state value, then no transition is allowed", t, func() {
		download := models.Download{State: "wrong"}
		validateDownloadTransitionsToPending(download)
	})

	Convey("Given an image download variant without state, then pending state is assumed when checking for transitions", t, func() {
		download := models.Download{}
		validateDownloadTransitionsToPending(download)
	})
}

func validateDownloadTransitionsToPending(download models.Download) {
	Convey("Then an allowed transition is successfully checked", func() {
		So(download.StateTransitionAllowed(models.StateImporting.String()), ShouldBeTrue)
	})
	Convey("Then a forbidden transition to a valid state is not allowed", func() {
		So(download.StateTransitionAllowed(models.StateDownloadPublished.String()), ShouldBeFalse)
	})
	Convey("Then a transition to an invalid state is not allowed", func() {
		So(download.StateTransitionAllowed("wrong"), ShouldBeFalse)
	})
}
