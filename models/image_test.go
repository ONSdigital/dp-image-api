package models_test

import (
	"testing"

	"github.com/ONSdigital/dp-image-api/apierrors"
	"github.com/ONSdigital/dp-image-api/models"
	. "github.com/smartystreets/goconvey/convey"
)

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

	Convey("Given a fully populated valid image, it is successfully validated", t, func() {
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
				"original": {},
			},
		}
		err := image.Validate()
		So(err, ShouldBeNil)
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
		So(image.StateTransitionAllowed(models.StateCreated.String()), ShouldBeTrue)
	})
	Convey("Then a forbidden transition to a valid state is not allowed", func() {
		So(image.StateTransitionAllowed(models.StatePublished.String()), ShouldBeFalse)
	})
	Convey("Then a transition to an invalid state is not allowed", func() {
		So(image.StateTransitionAllowed("wrong"), ShouldBeFalse)
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
		So(download.StateTransitionAllowed(models.StateDownloadPending.String()), ShouldBeTrue)
	})
	Convey("Then a forbidden transition to a valid state is not allowed", func() {
		So(download.StateTransitionAllowed(models.StateDownloadPublished.String()), ShouldBeFalse)
	})
	Convey("Then a transition to an invalid state is not allowed", func() {
		So(download.StateTransitionAllowed("wrong"), ShouldBeFalse)
	})
}
