package api_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ONSdigital/dp-image-api/api/mock"
	"github.com/ONSdigital/dp-image-api/config"
	"github.com/ONSdigital/dp-image-api/models"
	. "github.com/smartystreets/goconvey/convey"
)

var newImagePayload = `
{
	"collection_id": "1234",
	"filename": "some-image-name",
	"license": {
		"title": "Open Government Licence v3.0",
		"href": "https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/"
	},
	"type": "chart"
}
`

var expectedImage = models.Image{
	CollectionID: "1234",
	Filename:     "some-image-name",
	License: &models.License{
		Title: "Open Government Licence v3.0",
		Href:  "https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/",
	},
	Type: "chart",
}

func TestCreateImageHandler(t *testing.T) {

	Convey("Given an image API that can successfully store valid images in mongoDB", t, func() {
		cfg, err := config.Get()
		So(err, ShouldBeNil)
		mongoDbMock := &mock.MongoServerMock{}
		imageApi := GetAPIWithMocks(mongoDbMock, cfg)
		var b string
		b = newImagePayload

		Convey("When a valid new image is posted", func() {
			r := httptest.NewRequest(http.MethodPost, "http://localhost:24700/images", bytes.NewBufferString(b))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			Convey("Then the created image with the new id is returned with status code 201, expected body and requestID header", func() {
				So(w.Code, ShouldEqual, http.StatusCreated)
				payload, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				retImage := models.Image{}
				err = json.Unmarshal(payload, &retImage)
				So(err, ShouldBeNil)
				So(retImage, ShouldResemble, expectedImage)
			})
		})

		Convey("An empty request body results in a BadRequest response", func() {
			r := httptest.NewRequest(http.MethodPost, "http://localhost:24700/images", nil)
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("An invalid request body results in a BadRequest response", func() {
			r := httptest.NewRequest(http.MethodPost, "http://localhost:24700/images", bytes.NewBufferString("invalidJson"))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})
	})
}
