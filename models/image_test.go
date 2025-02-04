package models_test

import (
	"strings"
	"testing"
	"time"

	"github.com/ONSdigital/dp-image-api/apierrors"
	"github.com/ONSdigital/dp-image-api/models"
	"github.com/smartystreets/goconvey/convey"
)

// Constants for testing
const (
	testVariantOriginal = "original"
	testDownloadType    = "originally uploaded file"
)

var (
	testImportStarted   = time.Date(2020, time.April, 26, 8, 5, 52, 0, time.UTC)
	testImportCompleted = time.Date(2020, time.April, 26, 8, 7, 32, 0, time.UTC)
)

func TestImageAllDownloadsOfState(t *testing.T) {
	convey.Convey("Given an image with all download variants in completed state", t, func() {
		image := models.Image{
			Downloads: map[string]models.Download{
				"original": {State: models.StateDownloadCompleted.String()},
				"var1":     {State: models.StateDownloadCompleted.String()},
				"var2":     {State: models.StateDownloadCompleted.String()},
			},
		}
		convey.Convey("Then AllDownloadsOfStateconvey.Should return expected values", func() {
			convey.So(image.AllDownloadsOfState(models.StateDownloadCompleted), convey.ShouldBeTrue)
			convey.So(image.AllDownloadsOfState(models.StateDownloadImporting), convey.ShouldBeFalse)
		})
	})

	convey.Convey("Given an image with all download variants in completed state except the original", t, func() {
		image := models.Image{
			Downloads: map[string]models.Download{
				"original": {State: models.StateDownloadPublished.String()},
				"var1":     {State: models.StateDownloadCompleted.String()},
				"var2":     {State: models.StateDownloadCompleted.String()},
			},
		}
		convey.Convey("Then AllDownloadsOfStateconvey.Should return expected values", func() {
			convey.So(image.AllDownloadsOfState(models.StateDownloadPublished), convey.ShouldBeFalse)
			convey.So(image.AllDownloadsOfState(models.StateDownloadCompleted), convey.ShouldBeFalse)
			convey.So(image.AllDownloadsOfState(models.StateDownloadImporting), convey.ShouldBeFalse)
		})
	})

	convey.Convey("Given an image with no download variants", t, func() {
		image := models.Image{}
		convey.Convey("Then AllDownloadsOfStateconvey.Should return expected values", func() {
			convey.So(image.AllDownloadsOfState(models.StateDownloadPublished), convey.ShouldBeFalse)
		})
	})
}

func TestImageAnyDownloadsOfState(t *testing.T) {
	convey.Convey("Given an image with all download variants in completed state", t, func() {
		image := models.Image{
			Downloads: map[string]models.Download{
				"original": {State: models.StateDownloadCompleted.String()},
				"var1":     {State: models.StateDownloadCompleted.String()},
				"var2":     {State: models.StateDownloadCompleted.String()},
			},
		}
		convey.Convey("Then AnyDownloadsOfStateconvey.Should return expected values", func() {
			convey.So(image.AnyDownloadsOfState(models.StateDownloadCompleted), convey.ShouldBeTrue)
			convey.So(image.AnyDownloadsOfState(models.StateDownloadImporting), convey.ShouldBeFalse)
		})
	})

	convey.Convey("Given an image with all download variants in completed state except the original", t, func() {
		image := models.Image{
			Downloads: map[string]models.Download{
				"original": {State: models.StateDownloadPublished.String()},
				"var1":     {State: models.StateDownloadCompleted.String()},
				"var2":     {State: models.StateDownloadCompleted.String()},
			},
		}
		convey.Convey("Then AnyDownloadsOfStateconvey.Should return expected values", func() {
			convey.So(image.AnyDownloadsOfState(models.StateDownloadPublished), convey.ShouldBeTrue)
			convey.So(image.AnyDownloadsOfState(models.StateDownloadCompleted), convey.ShouldBeTrue)
			convey.So(image.AnyDownloadsOfState(models.StateDownloadImporting), convey.ShouldBeFalse)
		})
	})

	convey.Convey("Given an image with no download variants", t, func() {
		image := models.Image{}
		convey.Convey("Then AnyDownloadsOfStateconvey.Should return expected values", func() {
			convey.So(image.AnyDownloadsOfState(models.StateDownloadPublished), convey.ShouldBeFalse)
		})
	})
}

func TestImageValidation(t *testing.T) {
	convey.Convey("Given an empty image, it is successfully validated", t, func() {
		image := models.Image{
			State: models.StateCreated.String(),
		}
		err := image.Validate()
		convey.So(err, convey.ShouldBeNil)
	})

	convey.Convey("Given an image with no state supplied, it fails to validate  with the expected error", t, func() {
		image := models.Image{}
		err := image.Validate()
		convey.So(err, convey.ShouldResemble, apierrors.ErrImageInvalidState)
	})

	convey.Convey("Given an image with a filename that is longer than the maximum allowed, it fails to validate with the expected error", t, func() {
		image := models.Image{
			Filename: strings.Repeat("Llanfairpwllgwyngyllgogerychwyrndrobwllllantysiliogogogoch", 10),
		}
		err := image.Validate()
		convey.So(err, convey.ShouldResemble, apierrors.ErrImageFilenameTooLong)
	})

	convey.Convey("Given an image with a state that does not correspond to any expected state, it fails to validate with the expected error", t, func() {
		image := models.Image{
			State: "wrong",
		}
		err := image.Validate()
		convey.So(err, convey.ShouldResemble, apierrors.ErrImageInvalidState)
	})

	convey.Convey("Given an image with a state of uploaded has no upload section it fails to validate with the expected error", t, func() {
		image := models.Image{
			State: "uploaded",
		}
		err := image.Validate()
		convey.So(err, convey.ShouldResemble, apierrors.ErrImageUploadEmpty)
	})

	convey.Convey("Given an image with a state of uploaded has no path in its upload section it fails to validate with the expected error", t, func() {
		image := models.Image{
			State:  "uploaded",
			Upload: &models.Upload{},
		}
		err := image.Validate()
		convey.So(err, convey.ShouldResemble, apierrors.ErrImageUploadPathEmpty)
	})

	convey.Convey("Given a fully populated valid image with a valid download variant, it is successfully validated", t, func() {
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
		convey.So(err, convey.ShouldBeNil)
	})
}

func TestImageValidateTransitionFrom(t *testing.T) {
	convey.Convey("Given an existing image in an uploaded state", t, func() {
		existing := &models.Image{
			State: models.StateUploaded.String(),
		}

		convey.Convey("When we try to transition to an Importing state", func() {
			image := &models.Image{
				State: models.StateImporting.String(),
			}
			err := image.ValidateTransitionFrom(existing)
			convey.So(err, convey.ShouldBeNil)
		})

		convey.Convey("When we try to transition to a Created state", func() {
			image := &models.Image{
				State: models.StateCreated.String(),
			}
			err := image.ValidateTransitionFrom(existing)
			convey.So(err, convey.ShouldResemble, apierrors.ErrImageStateTransitionNotAllowed)
		})
	})

	convey.Convey("Given an existing image in a Completed state", t, func() {
		existing := &models.Image{
			State: models.StateCompleted.String(),
		}

		convey.Convey("When we try to transition to an Importing state", func() {
			image := &models.Image{
				State: models.StateImporting.String(),
			}
			err := image.ValidateTransitionFrom(existing)
			convey.So(err, convey.ShouldResemble, apierrors.ErrImageStateTransitionNotAllowed)
		})

		convey.Convey("When we try to transition to a Deleted state", func() {
			image := &models.Image{
				State: models.StateDeleted.String(),
			}
			err := image.ValidateTransitionFrom(existing)
			convey.So(err, convey.ShouldResemble, apierrors.ErrImageAlreadyCompleted)
		})
	})
}

func TestImageStateTransitionAllowed(t *testing.T) {
	convey.Convey("Given an image in created state", t, func() {
		image := models.Image{
			State: models.StateCreated.String(),
		}
		validateTransitionsToCreated(image)
	})

	convey.Convey("Given an image with a wrong state value, then no transition is allowed", t, func() {
		image := models.Image{State: "wrong"}
		validateTransitionsToCreated(image)
	})

	convey.Convey("Given an image without state, then created state is assumed when checking for transitions", t, func() {
		image := models.Image{}
		validateTransitionsToCreated(image)
	})
}

func TestImageUpdatedState(t *testing.T) {
	convey.Convey("Given an image in FailedImport State", t, func() {
		image := models.Image{
			State: models.StateFailedImport.String(),
		}
		convey.Convey("Then UpdatedState should be the same", func() {
			convey.So(image.UpdatedState(), convey.ShouldResemble, models.StateFailedImport.String())
		})
	})

	convey.Convey("Given an image in FailedPublish State", t, func() {
		image := models.Image{
			State: models.StateFailedPublish.String(),
		}
		convey.Convey("Then UpdatedState should be the same", func() {
			convey.So(image.UpdatedState(), convey.ShouldResemble, models.StateFailedPublish.String())
		})
	})

	convey.Convey("Given an image in Importing State with any downloads in Failed state", t, func() {
		image := models.Image{
			State: models.StateImporting.String(),
			Downloads: map[string]models.Download{
				"original": {State: models.StateDownloadImporting.String()},
				"var1":     {State: models.StateDownloadImported.String()},
				"var2":     {State: models.StateDownloadFailed.String()},
			},
		}
		convey.Convey("Then UpdatedState should be FailedImport", func() {
			convey.So(image.UpdatedState(), convey.ShouldResemble, models.StateFailedImport.String())
		})
	})

	convey.Convey("Given an image in Importing State with all downloads in Imported state", t, func() {
		image := models.Image{
			State: models.StateImporting.String(),
			Downloads: map[string]models.Download{
				"original": {State: models.StateDownloadImported.String()},
				"var1":     {State: models.StateDownloadImported.String()},
			},
		}
		convey.Convey("Then UpdatedState should be Impoprted", func() {
			convey.So(image.UpdatedState(), convey.ShouldResemble, models.StateImported.String())
		})
	})

	convey.Convey("Given an image in Importing State with downloads in Imported and Importing state", t, func() {
		image := models.Image{
			State: models.StateImporting.String(),
			Downloads: map[string]models.Download{
				"original": {State: models.StateDownloadImporting.String()},
				"var1":     {State: models.StateDownloadImported.String()},
			},
		}
		convey.Convey("Then UpdatedState should remain Importing", func() {
			convey.So(image.UpdatedState(), convey.ShouldResemble, models.StateImporting.String())
		})
	})

	convey.Convey("Given an image in Published State with any downloads in Failed state", t, func() {
		image := models.Image{
			State: models.StatePublished.String(),
			Downloads: map[string]models.Download{
				"original": {State: models.StateDownloadCompleted.String()},
				"var1":     {State: models.StateDownloadImported.String()},
				"var2":     {State: models.StateDownloadFailed.String()},
			},
		}
		convey.Convey("Then UpdatedState should be FailedPublish", func() {
			convey.So(image.UpdatedState(), convey.ShouldResemble, models.StateFailedPublish.String())
		})
	})

	convey.Convey("Given an image in Published State with all downloads in Completed state", t, func() {
		image := models.Image{
			State: models.StatePublished.String(),
			Downloads: map[string]models.Download{
				"original": {State: models.StateDownloadCompleted.String()},
				"var1":     {State: models.StateDownloadCompleted.String()},
			},
		}
		convey.Convey("Then UpdatedState should be Completed", func() {
			convey.So(image.UpdatedState(), convey.ShouldResemble, models.StateCompleted.String())
		})
	})

	convey.Convey("Given an image in Published State with downloads in Published and Completed state", t, func() {
		image := models.Image{
			State: models.StatePublished.String(),
			Downloads: map[string]models.Download{
				"original": {State: models.StateCompleted.String()},
				"var1":     {State: models.StatePublished.String()},
			},
		}
		convey.Convey("Then UpdatedState should remain Published", func() {
			convey.So(image.UpdatedState(), convey.ShouldResemble, models.StatePublished.String())
		})
	})
}

// validateTransitionsToCreated validates that the provided image can transition to created state,
// and not to any forbidden of invalid state
func validateTransitionsToCreated(image models.Image) {
	convey.Convey("Then an allowed transition is successfully checked", func() {
		convey.So(image.StateTransitionAllowed(models.StateUploaded.String()), convey.ShouldBeTrue)
	})
	convey.Convey("Then a forbidden transition to a valid state is not allowed", func() {
		convey.So(image.StateTransitionAllowed(models.StatePublished.String()), convey.ShouldBeFalse)
	})
	convey.Convey("Then a transition to an invalid state is not allowed", func() {
		convey.So(image.StateTransitionAllowed("wrong"), convey.ShouldBeFalse)
	})
}

func TestDownloadValidation(t *testing.T) {
	convey.Convey("Given an empty download variant, it is successfully validated", t, func() {
		download := models.Download{}
		err := download.Validate()
		convey.So(err, convey.ShouldBeNil)
	})

	convey.Convey("Given a download variant with an invalid state name, it fails to validate with the expected error", t, func() {
		download := models.Download{
			State: "wrong",
		}
		err := download.Validate()
		convey.So(err, convey.ShouldResemble, apierrors.ErrImageDownloadInvalidState)
	})

	convey.Convey("Given a download variant with a valid state name, it is successfully validated", t, func() {
		download := models.Download{
			State: models.StateDownloadImported.String(),
		}
		err := download.Validate()
		convey.So(err, convey.ShouldBeNil)
	})
}

func TestDownloadValidateTransitionFrom(t *testing.T) {
	convey.Convey("And an existing download variant with valid importing state", t, func() {
		existing := &models.Download{
			ID:            testVariantOriginal,
			Type:          testDownloadType,
			State:         models.StateDownloadImporting.String(),
			ImportStarted: &testImportStarted,
		}

		convey.Convey("Then with a new download variant with valid state, the transition successfully validates", func() {
			download := models.Download{
				ID:              testVariantOriginal,
				Type:            testDownloadType,
				State:           models.StateDownloadImported.String(),
				ImportStarted:   &testImportStarted,
				ImportCompleted: &testImportCompleted,
			}
			err := download.ValidateTransitionFrom(existing)
			convey.So(err, convey.ShouldBeNil)
		})

		convey.Convey("Then with a new download variant with a changed type, the transition fails validation", func() {
			download := models.Download{
				ID:              testVariantOriginal,
				Type:            "some other type",
				State:           models.StateDownloadImported.String(),
				ImportStarted:   &testImportStarted,
				ImportCompleted: &testImportCompleted,
			}
			err := download.ValidateTransitionFrom(existing)
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(err, convey.ShouldResemble, apierrors.ErrImageDownloadTypeMismatch)
		})
	})

	convey.Convey("And an existing download variant with valid imported state", t, func() {
		existing := &models.Download{
			ID:              testVariantOriginal,
			Type:            testDownloadType,
			State:           models.StateDownloadImported.String(),
			ImportStarted:   &testImportStarted,
			ImportCompleted: &testImportCompleted,
		}

		convey.Convey("Then with a new download variant with importing state, the transition fails validation", func() {
			download := models.Download{
				ID:            testVariantOriginal,
				Type:          testDownloadType,
				State:         models.StateDownloadImporting.String(),
				ImportStarted: &testImportStarted,
			}
			err := download.ValidateTransitionFrom(existing)
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(err, convey.ShouldResemble, apierrors.ErrVariantStateTransitionNotAllowed)
		})
	})
}

func TestDownloadValidateForImage(t *testing.T) {
	convey.Convey("Given an existing image in importing state", t, func() {
		image := &models.Image{
			State: models.StateImporting.String(),
		}
		convey.Convey("Then with a new download variant of state Importing is valid", func() {
			download := models.Download{
				ID:            testVariantOriginal,
				Type:          testDownloadType,
				State:         models.StateDownloadImporting.String(),
				ImportStarted: &testImportStarted,
			}
			err := download.ValidateForImage(image)
			convey.So(err, convey.ShouldBeNil)
		})
	})

	convey.Convey("Given an existing image in imported state", t, func() {
		image := &models.Image{
			State: models.StateImported.String(),
		}
		convey.Convey("Then a  download variant of state Importing is invalid", func() {
			download := models.Download{
				State: models.StateDownloadImporting.String(),
			}
			err := download.ValidateForImage(image)
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(err, convey.ShouldResemble, apierrors.ErrImageNotImporting)
		})
		convey.Convey("Then a download variant of state Completed is invalid", func() {
			download := models.Download{
				State: models.StateDownloadCompleted.String(),
			}
			err := download.ValidateForImage(image)
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(err, convey.ShouldResemble, apierrors.ErrImageNotPublished)
		})
	})

	convey.Convey("Given an existing image in published state", t, func() {
		image := &models.Image{
			State: models.StatePublished.String(),
		}
		convey.Convey("Then a download variant of state Completed is valid", func() {
			download := models.Download{
				State: models.StateDownloadCompleted.String(),
			}
			err := download.ValidateForImage(image)
			convey.So(err, convey.ShouldBeNil)
		})
		convey.Convey("Then with a new download variant of state Imported is invalid", func() {
			download := models.Download{
				State: models.StateDownloadImported.String(),
			}
			err := download.ValidateForImage(image)
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(err, convey.ShouldResemble, apierrors.ErrImageNotImporting)
		})
	})

	convey.Convey("Given an existing image in completed state", t, func() {
		image := &models.Image{
			State: models.StateCompleted.String(),
		}
		convey.Convey("Then a download variant of state Completed is invalid", func() {
			download := models.Download{
				State: models.StateDownloadCompleted.String(),
			}
			err := download.ValidateForImage(image)
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(err, convey.ShouldResemble, apierrors.ErrImageAlreadyCompleted)
		})
		convey.Convey("Then a download variant of state Importing is invalid", func() {
			download := models.Download{
				State: models.StateDownloadImporting.String(),
			}
			err := download.ValidateForImage(image)
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(err, convey.ShouldResemble, apierrors.ErrImageNotImporting)
		})
	})
}

func TestDownloadStateTransitionAllowed(t *testing.T) {
	convey.Convey("Given an image download variant in pending state", t, func() {
		download := models.Download{
			State: models.StateDownloadPending.String(),
		}
		validateDownloadTransitionsToPending(download)
	})

	convey.Convey("Given an image download variant with a wrong state value, then no transition is allowed", t, func() {
		download := models.Download{State: "wrong"}
		validateDownloadTransitionsToPending(download)
	})

	convey.Convey("Given an image download variant without state, then pending state is assumed when checking for transitions", t, func() {
		download := models.Download{}
		validateDownloadTransitionsToPending(download)
	})
}

func validateDownloadTransitionsToPending(download models.Download) {
	convey.Convey("Then an allowed transition is successfully checked", func() {
		convey.So(download.StateTransitionAllowed(models.StateImporting.String()), convey.ShouldBeTrue)
	})
	convey.Convey("Then a forbidden transition to a valid state is not allowed", func() {
		convey.So(download.StateTransitionAllowed(models.StateDownloadPublished.String()), convey.ShouldBeFalse)
	})
	convey.Convey("Then a transition to an invalid state is not allowed", func() {
		convey.So(download.StateTransitionAllowed("wrong"), convey.ShouldBeFalse)
	})
}
