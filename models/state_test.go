package models_test

import (
	"testing"

	"github.com/ONSdigital/dp-image-api/models"
	. "github.com/smartystreets/goconvey/convey"
)

func TestStateValidation(t *testing.T) {

	Convey("Given a Created State, then only transitions to created, uploaded and deleted are allowed", t, func() {
		So(models.StateCreated.TransitionAllowed(models.StateCreated), ShouldBeTrue)
		So(models.StateCreated.TransitionAllowed(models.StateUploaded), ShouldBeTrue)
		So(models.StateCreated.TransitionAllowed(models.StateImporting), ShouldBeFalse)
		So(models.StateCreated.TransitionAllowed(models.StateImported), ShouldBeFalse)
		So(models.StateCreated.TransitionAllowed(models.StatePublished), ShouldBeFalse)
		So(models.StateCreated.TransitionAllowed(models.StateDeleted), ShouldBeTrue)
	})

	Convey("Given a Uploaded State, then only transitions to uploaded, importing and deleted are allowed", t, func() {
		So(models.StateUploaded.TransitionAllowed(models.StateCreated), ShouldBeFalse)
		So(models.StateUploaded.TransitionAllowed(models.StateUploaded), ShouldBeTrue)
		So(models.StateUploaded.TransitionAllowed(models.StateImporting), ShouldBeTrue)
		So(models.StateUploaded.TransitionAllowed(models.StateImported), ShouldBeFalse)
		So(models.StateUploaded.TransitionAllowed(models.StatePublished), ShouldBeFalse)
		So(models.StateUploaded.TransitionAllowed(models.StateDeleted), ShouldBeTrue)
	})

	Convey("Given an Importing State, then only transitions to importing, imported and deleted are allowed", t, func() {
		So(models.StateImporting.TransitionAllowed(models.StateCreated), ShouldBeFalse)
		So(models.StateImporting.TransitionAllowed(models.StateUploaded), ShouldBeFalse)
		So(models.StateImporting.TransitionAllowed(models.StateImporting), ShouldBeTrue)
		So(models.StateImporting.TransitionAllowed(models.StateImported), ShouldBeTrue)
		So(models.StateImporting.TransitionAllowed(models.StatePublished), ShouldBeFalse)
		So(models.StateImporting.TransitionAllowed(models.StateDeleted), ShouldBeTrue)
	})

	Convey("Given an Imported State, then only transitions to imported, published and deleted are allowed", t, func() {
		So(models.StateImported.TransitionAllowed(models.StateCreated), ShouldBeFalse)
		So(models.StateImported.TransitionAllowed(models.StateUploaded), ShouldBeFalse)
		So(models.StateImported.TransitionAllowed(models.StateImporting), ShouldBeFalse)
		So(models.StateImported.TransitionAllowed(models.StateImported), ShouldBeTrue)
		So(models.StateImported.TransitionAllowed(models.StatePublished), ShouldBeTrue)
		So(models.StateImported.TransitionAllowed(models.StateDeleted), ShouldBeTrue)
	})

	Convey("Given a Published State, then only transitions to published and deleted are allowed", t, func() {
		So(models.StatePublished.TransitionAllowed(models.StateCreated), ShouldBeFalse)
		So(models.StatePublished.TransitionAllowed(models.StateUploaded), ShouldBeFalse)
		So(models.StatePublished.TransitionAllowed(models.StateImporting), ShouldBeFalse)
		So(models.StatePublished.TransitionAllowed(models.StateImported), ShouldBeFalse)
		So(models.StatePublished.TransitionAllowed(models.StatePublished), ShouldBeTrue)
		So(models.StatePublished.TransitionAllowed(models.StateDeleted), ShouldBeTrue)
	})

	Convey("Given a Deleted State, then no transitions are allowed", t, func() {
		So(models.StateDeleted.TransitionAllowed(models.StateCreated), ShouldBeFalse)
		So(models.StateDeleted.TransitionAllowed(models.StateUploaded), ShouldBeFalse)
		So(models.StateDeleted.TransitionAllowed(models.StateImporting), ShouldBeFalse)
		So(models.StateDeleted.TransitionAllowed(models.StateImported), ShouldBeFalse)
		So(models.StateDeleted.TransitionAllowed(models.StatePublished), ShouldBeFalse)
		So(models.StateDeleted.TransitionAllowed(models.StateDeleted), ShouldBeFalse)
	})
}
