package models_test

import (
	"testing"

	"github.com/ONSdigital/dp-image-api/models"
	"github.com/smartystreets/goconvey/convey"
)

func TestStateValidation(t *testing.T) {
	convey.Convey("Given a Created State, then only transitions to uploaded and deleted are allowed", t, func() {
		convey.So(models.StateCreated.TransitionAllowed(models.StateCreated), convey.ShouldBeFalse)
		convey.So(models.StateCreated.TransitionAllowed(models.StateUploaded), convey.ShouldBeTrue)
		convey.So(models.StateCreated.TransitionAllowed(models.StateImporting), convey.ShouldBeFalse)
		convey.So(models.StateCreated.TransitionAllowed(models.StateImported), convey.ShouldBeFalse)
		convey.So(models.StateCreated.TransitionAllowed(models.StatePublished), convey.ShouldBeFalse)
		convey.So(models.StateCreated.TransitionAllowed(models.StateCompleted), convey.ShouldBeFalse)
		convey.So(models.StateCreated.TransitionAllowed(models.StateDeleted), convey.ShouldBeTrue)
		convey.So(models.StateCreated.TransitionAllowed(models.StateFailedImport), convey.ShouldBeFalse)
		convey.So(models.StateCreated.TransitionAllowed(models.StateFailedPublish), convey.ShouldBeFalse)
	})

	convey.Convey("Given a Uploaded State, then only transitions to importing and deleted are allowed", t, func() {
		convey.So(models.StateUploaded.TransitionAllowed(models.StateCreated), convey.ShouldBeFalse)
		convey.So(models.StateUploaded.TransitionAllowed(models.StateUploaded), convey.ShouldBeFalse)
		convey.So(models.StateUploaded.TransitionAllowed(models.StateImporting), convey.ShouldBeTrue)
		convey.So(models.StateUploaded.TransitionAllowed(models.StateImported), convey.ShouldBeFalse)
		convey.So(models.StateUploaded.TransitionAllowed(models.StatePublished), convey.ShouldBeFalse)
		convey.So(models.StateUploaded.TransitionAllowed(models.StateCompleted), convey.ShouldBeFalse)
		convey.So(models.StateUploaded.TransitionAllowed(models.StateDeleted), convey.ShouldBeTrue)
		convey.So(models.StateUploaded.TransitionAllowed(models.StateFailedImport), convey.ShouldBeTrue)
		convey.So(models.StateUploaded.TransitionAllowed(models.StateFailedPublish), convey.ShouldBeFalse)
	})

	convey.Convey("Given an Importing State, then only transitions to imported, failedImport and deleted are allowed", t, func() {
		convey.So(models.StateImporting.TransitionAllowed(models.StateCreated), convey.ShouldBeFalse)
		convey.So(models.StateImporting.TransitionAllowed(models.StateUploaded), convey.ShouldBeFalse)
		convey.So(models.StateImporting.TransitionAllowed(models.StateImporting), convey.ShouldBeFalse)
		convey.So(models.StateImporting.TransitionAllowed(models.StateImported), convey.ShouldBeTrue)
		convey.So(models.StateImporting.TransitionAllowed(models.StatePublished), convey.ShouldBeFalse)
		convey.So(models.StateImporting.TransitionAllowed(models.StateCompleted), convey.ShouldBeFalse)
		convey.So(models.StateImporting.TransitionAllowed(models.StateDeleted), convey.ShouldBeTrue)
		convey.So(models.StateImporting.TransitionAllowed(models.StateFailedImport), convey.ShouldBeTrue)
		convey.So(models.StateImporting.TransitionAllowed(models.StateFailedPublish), convey.ShouldBeFalse)
	})

	convey.Convey("Given an Imported State, then only transitions to published and deleted are allowed", t, func() {
		convey.So(models.StateImported.TransitionAllowed(models.StateCreated), convey.ShouldBeFalse)
		convey.So(models.StateImported.TransitionAllowed(models.StateUploaded), convey.ShouldBeFalse)
		convey.So(models.StateImported.TransitionAllowed(models.StateImporting), convey.ShouldBeFalse)
		convey.So(models.StateImported.TransitionAllowed(models.StateImported), convey.ShouldBeFalse)
		convey.So(models.StateImported.TransitionAllowed(models.StatePublished), convey.ShouldBeTrue)
		convey.So(models.StateImported.TransitionAllowed(models.StateCompleted), convey.ShouldBeFalse)
		convey.So(models.StateImported.TransitionAllowed(models.StateDeleted), convey.ShouldBeTrue)
		convey.So(models.StateImported.TransitionAllowed(models.StateFailedImport), convey.ShouldBeFalse)
		convey.So(models.StateImported.TransitionAllowed(models.StateFailedPublish), convey.ShouldBeFalse)
	})

	convey.Convey("Given a Published State, then only transitions to failedPublish, completed and deleted are allowed", t, func() {
		convey.So(models.StatePublished.TransitionAllowed(models.StateCreated), convey.ShouldBeFalse)
		convey.So(models.StatePublished.TransitionAllowed(models.StateUploaded), convey.ShouldBeFalse)
		convey.So(models.StatePublished.TransitionAllowed(models.StateImporting), convey.ShouldBeFalse)
		convey.So(models.StatePublished.TransitionAllowed(models.StateImported), convey.ShouldBeFalse)
		convey.So(models.StatePublished.TransitionAllowed(models.StatePublished), convey.ShouldBeFalse)
		convey.So(models.StatePublished.TransitionAllowed(models.StateCompleted), convey.ShouldBeTrue)
		convey.So(models.StatePublished.TransitionAllowed(models.StateDeleted), convey.ShouldBeTrue)
		convey.So(models.StatePublished.TransitionAllowed(models.StateFailedImport), convey.ShouldBeFalse)
		convey.So(models.StatePublished.TransitionAllowed(models.StateFailedPublish), convey.ShouldBeTrue)
	})

	convey.Convey("Given a Completed State, then only transitions to deleted are allowed", t, func() {
		convey.So(models.StateCompleted.TransitionAllowed(models.StateCreated), convey.ShouldBeFalse)
		convey.So(models.StateCompleted.TransitionAllowed(models.StateUploaded), convey.ShouldBeFalse)
		convey.So(models.StateCompleted.TransitionAllowed(models.StateImporting), convey.ShouldBeFalse)
		convey.So(models.StateCompleted.TransitionAllowed(models.StateImported), convey.ShouldBeFalse)
		convey.So(models.StateCompleted.TransitionAllowed(models.StatePublished), convey.ShouldBeFalse)
		convey.So(models.StateCompleted.TransitionAllowed(models.StateCompleted), convey.ShouldBeFalse)
		convey.So(models.StateCompleted.TransitionAllowed(models.StateDeleted), convey.ShouldBeTrue)
		convey.So(models.StateCompleted.TransitionAllowed(models.StateFailedImport), convey.ShouldBeFalse)
		convey.So(models.StateCompleted.TransitionAllowed(models.StateFailedPublish), convey.ShouldBeFalse)
	})

	convey.Convey("Given a Deleted State, then no transitions are allowed", t, func() {
		convey.So(models.StateDeleted.TransitionAllowed(models.StateCreated), convey.ShouldBeFalse)
		convey.So(models.StateDeleted.TransitionAllowed(models.StateUploaded), convey.ShouldBeFalse)
		convey.So(models.StateDeleted.TransitionAllowed(models.StateImporting), convey.ShouldBeFalse)
		convey.So(models.StateDeleted.TransitionAllowed(models.StateImported), convey.ShouldBeFalse)
		convey.So(models.StateDeleted.TransitionAllowed(models.StatePublished), convey.ShouldBeFalse)
		convey.So(models.StateDeleted.TransitionAllowed(models.StateCompleted), convey.ShouldBeFalse)
		convey.So(models.StateDeleted.TransitionAllowed(models.StateDeleted), convey.ShouldBeFalse)
		convey.So(models.StateDeleted.TransitionAllowed(models.StateFailedImport), convey.ShouldBeFalse)
		convey.So(models.StateDeleted.TransitionAllowed(models.StateFailedPublish), convey.ShouldBeFalse)
	})

	convey.Convey("Given a FailedImport State, then only transitions to deleted are allowed", t, func() {
		convey.So(models.StateFailedImport.TransitionAllowed(models.StateCreated), convey.ShouldBeFalse)
		convey.So(models.StateFailedImport.TransitionAllowed(models.StateUploaded), convey.ShouldBeFalse)
		convey.So(models.StateFailedImport.TransitionAllowed(models.StateImporting), convey.ShouldBeFalse)
		convey.So(models.StateFailedImport.TransitionAllowed(models.StateImported), convey.ShouldBeFalse)
		convey.So(models.StateFailedImport.TransitionAllowed(models.StatePublished), convey.ShouldBeFalse)
		convey.So(models.StateFailedImport.TransitionAllowed(models.StateCompleted), convey.ShouldBeFalse)
		convey.So(models.StateFailedImport.TransitionAllowed(models.StateDeleted), convey.ShouldBeTrue)
		convey.So(models.StateFailedImport.TransitionAllowed(models.StateFailedImport), convey.ShouldBeFalse)
		convey.So(models.StateFailedImport.TransitionAllowed(models.StateFailedPublish), convey.ShouldBeFalse)
	})

	convey.Convey("Given a FailedPublish State, then only transitions to deleted are allowed", t, func() {
		convey.So(models.StateFailedPublish.TransitionAllowed(models.StateCreated), convey.ShouldBeFalse)
		convey.So(models.StateFailedPublish.TransitionAllowed(models.StateUploaded), convey.ShouldBeFalse)
		convey.So(models.StateFailedPublish.TransitionAllowed(models.StateImporting), convey.ShouldBeFalse)
		convey.So(models.StateFailedPublish.TransitionAllowed(models.StateImported), convey.ShouldBeFalse)
		convey.So(models.StateFailedPublish.TransitionAllowed(models.StatePublished), convey.ShouldBeFalse)
		convey.So(models.StateFailedPublish.TransitionAllowed(models.StateCompleted), convey.ShouldBeFalse)
		convey.So(models.StateFailedPublish.TransitionAllowed(models.StateDeleted), convey.ShouldBeTrue)
		convey.So(models.StateFailedPublish.TransitionAllowed(models.StateFailedImport), convey.ShouldBeFalse)
		convey.So(models.StateFailedPublish.TransitionAllowed(models.StateFailedPublish), convey.ShouldBeFalse)
	})
}
