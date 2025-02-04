package models_test

import (
	"testing"

	"github.com/ONSdigital/dp-image-api/models"
	"github.com/smartystreets/goconvey/convey"
)

func TestDownloadStateValidation(t *testing.T) {
	convey.Convey("Given a Pending download state, then only transitions to pending and importing are allowed", t, func() {
		convey.So(models.StateDownloadPending.TransitionAllowed(models.StateDownloadPending), convey.ShouldBeFalse)
		convey.So(models.StateDownloadPending.TransitionAllowed(models.StateDownloadImporting), convey.ShouldBeTrue)
		convey.So(models.StateDownloadPending.TransitionAllowed(models.StateDownloadImported), convey.ShouldBeFalse)
		convey.So(models.StateDownloadPending.TransitionAllowed(models.StateDownloadPublished), convey.ShouldBeFalse)
		convey.So(models.StateDownloadPending.TransitionAllowed(models.StateDownloadCompleted), convey.ShouldBeFalse)
		convey.So(models.StateDownloadPending.TransitionAllowed(models.StateDownloadFailed), convey.ShouldBeFalse)
	})

	convey.Convey("Given a Importing download state, then only transitions to importing, imported and failed are allowed", t, func() {
		convey.So(models.StateDownloadImporting.TransitionAllowed(models.StateDownloadPending), convey.ShouldBeFalse)
		convey.So(models.StateDownloadImporting.TransitionAllowed(models.StateDownloadImporting), convey.ShouldBeFalse)
		convey.So(models.StateDownloadImporting.TransitionAllowed(models.StateDownloadImported), convey.ShouldBeTrue)
		convey.So(models.StateDownloadImporting.TransitionAllowed(models.StateDownloadPublished), convey.ShouldBeFalse)
		convey.So(models.StateDownloadImporting.TransitionAllowed(models.StateDownloadCompleted), convey.ShouldBeFalse)
		convey.So(models.StateDownloadImporting.TransitionAllowed(models.StateDownloadFailed), convey.ShouldBeTrue)
	})

	convey.Convey("Given a Imported download state, then only transitions to imported and published are allowed", t, func() {
		convey.So(models.StateDownloadImported.TransitionAllowed(models.StateDownloadPending), convey.ShouldBeFalse)
		convey.So(models.StateDownloadImported.TransitionAllowed(models.StateDownloadImporting), convey.ShouldBeFalse)
		convey.So(models.StateDownloadImported.TransitionAllowed(models.StateDownloadImported), convey.ShouldBeFalse)
		convey.So(models.StateDownloadImported.TransitionAllowed(models.StateDownloadPublished), convey.ShouldBeTrue)
		convey.So(models.StateDownloadImported.TransitionAllowed(models.StateDownloadCompleted), convey.ShouldBeFalse)
		convey.So(models.StateDownloadImported.TransitionAllowed(models.StateDownloadFailed), convey.ShouldBeFalse)
	})

	convey.Convey("Given a Published download state, then only transitions to published and completed are allowed", t, func() {
		convey.So(models.StateDownloadPublished.TransitionAllowed(models.StateDownloadPending), convey.ShouldBeFalse)
		convey.So(models.StateDownloadPublished.TransitionAllowed(models.StateDownloadImporting), convey.ShouldBeFalse)
		convey.So(models.StateDownloadPublished.TransitionAllowed(models.StateDownloadImported), convey.ShouldBeFalse)
		convey.So(models.StateDownloadPublished.TransitionAllowed(models.StateDownloadPublished), convey.ShouldBeFalse)
		convey.So(models.StateDownloadPublished.TransitionAllowed(models.StateDownloadCompleted), convey.ShouldBeTrue)
		convey.So(models.StateDownloadPublished.TransitionAllowed(models.StateDownloadFailed), convey.ShouldBeFalse)
	})

	convey.Convey("Given a Completed download state, then only transitions to completed are allowed", t, func() {
		convey.So(models.StateDownloadCompleted.TransitionAllowed(models.StateDownloadPending), convey.ShouldBeFalse)
		convey.So(models.StateDownloadCompleted.TransitionAllowed(models.StateDownloadImporting), convey.ShouldBeFalse)
		convey.So(models.StateDownloadCompleted.TransitionAllowed(models.StateDownloadImported), convey.ShouldBeFalse)
		convey.So(models.StateDownloadCompleted.TransitionAllowed(models.StateDownloadPublished), convey.ShouldBeFalse)
		convey.So(models.StateDownloadCompleted.TransitionAllowed(models.StateDownloadCompleted), convey.ShouldBeFalse)
		convey.So(models.StateDownloadCompleted.TransitionAllowed(models.StateDownloadFailed), convey.ShouldBeFalse)
	})

	convey.Convey("Given a Failed download state, then only transitions to failed are allowed", t, func() {
		convey.So(models.StateDownloadFailed.TransitionAllowed(models.StateDownloadPending), convey.ShouldBeFalse)
		convey.So(models.StateDownloadFailed.TransitionAllowed(models.StateDownloadImporting), convey.ShouldBeFalse)
		convey.So(models.StateDownloadFailed.TransitionAllowed(models.StateDownloadImported), convey.ShouldBeFalse)
		convey.So(models.StateDownloadFailed.TransitionAllowed(models.StateDownloadPublished), convey.ShouldBeFalse)
		convey.So(models.StateDownloadFailed.TransitionAllowed(models.StateDownloadCompleted), convey.ShouldBeFalse)
		convey.So(models.StateDownloadFailed.TransitionAllowed(models.StateDownloadFailed), convey.ShouldBeFalse)
	})
}
