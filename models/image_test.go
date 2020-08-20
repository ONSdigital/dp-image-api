package models_test

import (
	"testing"

	"github.com/ONSdigital/dp-image-api/apierrors"
	"github.com/ONSdigital/dp-image-api/models"
	. "github.com/smartystreets/goconvey/convey"
)

func TestImageAllOtherDownloadsCompleted(t *testing.T) {
	Convey("Given an image with all download variants in completed state except the original", t, func() {
		image := models.Image{
			Downloads: map[string]models.Download{
				"original": {State: models.StatePublished.String()},
				"var1":     {State: models.StateDownloadCompleted.String()},
				"var2":     {State: models.StateDownloadCompleted.String()},
			},
		}
		Convey("Then AllOtherDownloadsCompleted for the original returns true", func() {
			So(image.AllOtherDownloadsCompleted("original"), ShouldBeTrue)
		})
		Convey("Then AllOtherDownloadsCompleted for var1 returns false", func() {
			So(image.AllOtherDownloadsCompleted("var1"), ShouldBeFalse)
		})
		Convey("Then AllOtherDownloadsCompleted for an inexistent variant returns false", func() {
			So(image.AllOtherDownloadsCompleted("wrong"), ShouldBeFalse)
		})
	})
}

func TestImageAllOtherDownloadsImported(t *testing.T) {
	Convey("Given an image with all download variants in imported state except the original", t, func() {
		image := models.Image{
			Downloads: map[string]models.Download{
				"original": {State: models.StateImporting.String()},
				"var1":     {State: models.StateImported.String()},
				"var2":     {State: models.StateImported.String()},
			},
		}
		Convey("Then AllOtherDownloadsCompleted for the original returns true", func() {
			So(image.AllOtherDownloadsImported("original"), ShouldBeTrue)
		})
		Convey("Then AllOtherDownloadsCompleted for var1 returns false", func() {
			So(image.AllOtherDownloadsImported("var1"), ShouldBeFalse)
		})
		Convey("Then AllOtherDownloadsCompleted for an inexistent variant returns false", func() {
			So(image.AllOtherDownloadsImported("wrong"), ShouldBeFalse)
		})
	})
}

func TestImageAnyDownloadFailed(t *testing.T) {
	Convey("Given an image with all download variants in non-failed state", t, func() {
		image := models.Image{
			Downloads: map[string]models.Download{
				"original": {State: models.StateDownloadCompleted.String()},
				"var1":     {State: models.StateDownloadPublished.String()},
				"var2":     {State: models.StateDownloadCompleted.String()},
			},
		}
		Convey("Then AnyDownloadFailed returns false", func() {
			So(image.AnyDownloadFailed(), ShouldBeFalse)
		})
	})

	Convey("Given an image with all download variants in valid state except the original, which is in failed state", t, func() {
		image := models.Image{
			Downloads: map[string]models.Download{
				"original": {State: models.StateDownloadFailed.String()},
				"var1":     {State: models.StatePublished.String()},
				"var2":     {State: models.StateDownloadCompleted.String()},
			},
		}
		Convey("Then AnyDownloadFailed returns true", func() {
			So(image.AnyDownloadFailed(), ShouldBeTrue)
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
		}
		err := image.Validate()
		So(err, ShouldBeNil)
	})
}

func TestImageValidateTransitionFrom(t *testing.T) {
	Convey("Given an existing image in an uploaded state", t, func() {
		existing := &models.Image{
			State: models.StateUploaded.String(),
		}

		Convey("When we try to transition to an Importing state", func() {
			image := &models.Image{
				State: models.StateImporting.String(),
			}
			err := image.ValidateTransitionFrom(existing)
			So(err, ShouldBeNil)
		})

		Convey("When we try to transition to a Created state", func() {
			image := &models.Image{
				State: models.StateCreated.String(),
			}
			err := image.ValidateTransitionFrom(existing)
			So(err, ShouldResemble, apierrors.ErrImageStateTransitionNotAllowed)
		})
	})

	Convey("Given an existing image in a Completed state", t, func() {
		existing := &models.Image{
			State: models.StateCompleted.String(),
		}

		Convey("When we try to transition to an Importing state", func() {
			image := &models.Image{
				State: models.StateImporting.String(),
			}
			err := image.ValidateTransitionFrom(existing)
			So(err, ShouldResemble, apierrors.ErrImageStateTransitionNotAllowed)
		})

		Convey("When we try to transition to a Deleted state", func() {
			image := &models.Image{
				State: models.StateDeleted.String(),
			}
			err := image.ValidateTransitionFrom(existing)
			So(err, ShouldResemble, apierrors.ErrImageAlreadyCompleted)
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
