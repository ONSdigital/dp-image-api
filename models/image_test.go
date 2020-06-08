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
			Downloads: map[string]map[string]models.Download{
				"png": {"resolution": {}},
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
		validateTransitionsFromCreated(image)
	})

	Convey("Given an image with a wrong state value, then no transition is allowed", t, func() {
		image := models.Image{State: "wrong"}
		validateTransitionsFromCreated(image)
	})

	Convey("Given an image without state, then created state is assumed when checking for transitions", t, func() {
		image := models.Image{}
		validateTransitionsFromCreated(image)
	})
}

func validateTransitionsFromCreated(image models.Image) {
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
