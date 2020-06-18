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
		So(models.StateCreated.TransitionAllowed(models.StatePublishing), ShouldBeFalse)
		So(models.StateCreated.TransitionAllowed(models.StatePublished), ShouldBeFalse)
		So(models.StateCreated.TransitionAllowed(models.StateDeleted), ShouldBeTrue)
	})

	Convey("Given a Uploaded State, then only transitions to uploaded, publishing and deleted are allowed", t, func() {
		So(models.StateUploaded.TransitionAllowed(models.StateCreated), ShouldBeFalse)
		So(models.StateUploaded.TransitionAllowed(models.StateUploaded), ShouldBeTrue)
		So(models.StateUploaded.TransitionAllowed(models.StatePublishing), ShouldBeTrue)
		So(models.StateUploaded.TransitionAllowed(models.StatePublished), ShouldBeFalse)
		So(models.StateUploaded.TransitionAllowed(models.StateDeleted), ShouldBeTrue)
	})

	Convey("Given a Publishing State, then only transitions to publishing, uploaded, published and deleted are allowed", t, func() {
		So(models.StatePublishing.TransitionAllowed(models.StateCreated), ShouldBeFalse)
		So(models.StatePublishing.TransitionAllowed(models.StateUploaded), ShouldBeTrue)
		So(models.StatePublishing.TransitionAllowed(models.StatePublishing), ShouldBeTrue)
		So(models.StatePublishing.TransitionAllowed(models.StatePublished), ShouldBeTrue)
		So(models.StatePublishing.TransitionAllowed(models.StateDeleted), ShouldBeTrue)
	})

	Convey("Given a Published State, then only transitions to published and deleted are allowed", t, func() {
		So(models.StatePublished.TransitionAllowed(models.StateCreated), ShouldBeFalse)
		So(models.StatePublished.TransitionAllowed(models.StateUploaded), ShouldBeFalse)
		So(models.StatePublished.TransitionAllowed(models.StatePublishing), ShouldBeFalse)
		So(models.StatePublished.TransitionAllowed(models.StatePublished), ShouldBeTrue)
		So(models.StatePublished.TransitionAllowed(models.StateDeleted), ShouldBeTrue)
	})

	Convey("Given a Deleted State, then no transitions are allowed", t, func() {
		So(models.StateDeleted.TransitionAllowed(models.StateCreated), ShouldBeFalse)
		So(models.StateDeleted.TransitionAllowed(models.StateUploaded), ShouldBeFalse)
		So(models.StateDeleted.TransitionAllowed(models.StatePublishing), ShouldBeFalse)
		So(models.StateDeleted.TransitionAllowed(models.StatePublished), ShouldBeFalse)
		So(models.StateDeleted.TransitionAllowed(models.StateDeleted), ShouldBeFalse)
	})
}
