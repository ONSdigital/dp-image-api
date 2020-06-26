package models_test

import (
	"testing"

	"github.com/ONSdigital/dp-image-api/models"
	. "github.com/smartystreets/goconvey/convey"
)

func TestDownloadStateValidation(t *testing.T) {

	Convey("Given a Pending download state, then only transitions to pending and importing are allowed", t, func() {
		So(models.StateDownloadPending.TransitionAllowed(models.StateDownloadPending), ShouldBeTrue)
		So(models.StateDownloadPending.TransitionAllowed(models.StateDownloadImporting), ShouldBeTrue)
		So(models.StateDownloadPending.TransitionAllowed(models.StateDownloadImported), ShouldBeFalse)
		So(models.StateDownloadPending.TransitionAllowed(models.StateDownloadPublished), ShouldBeFalse)
		So(models.StateDownloadPending.TransitionAllowed(models.StateDownloadCompleted), ShouldBeFalse)
		So(models.StateDownloadPending.TransitionAllowed(models.StateDownloadFailed), ShouldBeFalse)
	})

	Convey("Given a Importing download state, then only transitions to importing, imported and failed are allowed", t, func() {
		So(models.StateDownloadImporting.TransitionAllowed(models.StateDownloadPending), ShouldBeFalse)
		So(models.StateDownloadImporting.TransitionAllowed(models.StateDownloadImporting), ShouldBeTrue)
		So(models.StateDownloadImporting.TransitionAllowed(models.StateDownloadImported), ShouldBeTrue)
		So(models.StateDownloadImporting.TransitionAllowed(models.StateDownloadPublished), ShouldBeFalse)
		So(models.StateDownloadImporting.TransitionAllowed(models.StateDownloadCompleted), ShouldBeFalse)
		So(models.StateDownloadImporting.TransitionAllowed(models.StateDownloadFailed), ShouldBeTrue)
	})

	Convey("Given a Imported download state, then only transitions to imported and published are allowed", t, func() {
		So(models.StateDownloadImported.TransitionAllowed(models.StateDownloadPending), ShouldBeFalse)
		So(models.StateDownloadImported.TransitionAllowed(models.StateDownloadImporting), ShouldBeFalse)
		So(models.StateDownloadImported.TransitionAllowed(models.StateDownloadImported), ShouldBeTrue)
		So(models.StateDownloadImported.TransitionAllowed(models.StateDownloadPublished), ShouldBeTrue)
		So(models.StateDownloadImported.TransitionAllowed(models.StateDownloadCompleted), ShouldBeFalse)
		So(models.StateDownloadImported.TransitionAllowed(models.StateDownloadFailed), ShouldBeFalse)
	})

	Convey("Given a Published download state, then only transitions to published and completed are allowed", t, func() {
		So(models.StateDownloadPublished.TransitionAllowed(models.StateDownloadPending), ShouldBeFalse)
		So(models.StateDownloadPublished.TransitionAllowed(models.StateDownloadImporting), ShouldBeFalse)
		So(models.StateDownloadPublished.TransitionAllowed(models.StateDownloadImported), ShouldBeFalse)
		So(models.StateDownloadPublished.TransitionAllowed(models.StateDownloadPublished), ShouldBeTrue)
		So(models.StateDownloadPublished.TransitionAllowed(models.StateDownloadCompleted), ShouldBeTrue)
		So(models.StateDownloadPublished.TransitionAllowed(models.StateDownloadFailed), ShouldBeFalse)
	})

	Convey("Given a Completed download state, then only transitions to completed are allowed", t, func() {
		So(models.StateDownloadCompleted.TransitionAllowed(models.StateDownloadPending), ShouldBeFalse)
		So(models.StateDownloadCompleted.TransitionAllowed(models.StateDownloadImporting), ShouldBeFalse)
		So(models.StateDownloadCompleted.TransitionAllowed(models.StateDownloadImported), ShouldBeFalse)
		So(models.StateDownloadCompleted.TransitionAllowed(models.StateDownloadPublished), ShouldBeFalse)
		So(models.StateDownloadCompleted.TransitionAllowed(models.StateDownloadCompleted), ShouldBeTrue)
		So(models.StateDownloadCompleted.TransitionAllowed(models.StateDownloadFailed), ShouldBeFalse)
	})

	Convey("Given a Failed download state, then only transitions to failed are allowed", t, func() {
		So(models.StateDownloadFailed.TransitionAllowed(models.StateDownloadPending), ShouldBeFalse)
		So(models.StateDownloadFailed.TransitionAllowed(models.StateDownloadImporting), ShouldBeFalse)
		So(models.StateDownloadFailed.TransitionAllowed(models.StateDownloadImported), ShouldBeFalse)
		So(models.StateDownloadFailed.TransitionAllowed(models.StateDownloadPublished), ShouldBeFalse)
		So(models.StateDownloadFailed.TransitionAllowed(models.StateDownloadCompleted), ShouldBeFalse)
		So(models.StateDownloadFailed.TransitionAllowed(models.StateDownloadFailed), ShouldBeTrue)
	})
}
