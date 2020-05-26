package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ONSdigital/dp-image-api/api"
	"github.com/ONSdigital/dp-image-api/api/mock"
	"github.com/ONSdigital/dp-image-api/apierrors"
	"github.com/ONSdigital/dp-image-api/config"
	"github.com/ONSdigital/dp-image-api/models"
	"github.com/ONSdigital/dp-net/handlers"
	. "github.com/smartystreets/goconvey/convey"
)

const testCreatedImageID = "createdImageID"
const testPublishedImageID = "publishedImageID"
const testCollectionID = "1234"

var errMongoDB = errors.New("MongoDB generic error")

var newImagePayload = fmt.Sprintf(`
{
	"collection_id": "%s",
	"filename": "some-image-name",
	"license": {
		"title": "Open Government Licence v3.0",
		"href": "https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/"
	},
	"type": "chart"
}
`, testCollectionID)

var newImagePayloadTooLong = fmt.Sprintf(`
{
	"collection_id": "%s",
	"filename": "Llanfairpwllgwyngyllgogerychwyrndrobwllllantysiliogogogoch",
	"type": "Filename is too long"
}
`, testCollectionID)

var newImageInvalidState = fmt.Sprintf(`
{
	"collection_id": "%s",
	"filename": "some-image-name",
	"state": "invalidState",
	"license": {
		"title": "Open Government Licence v3.0",
		"href": "https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/"
	},
	"type": "chart"
}
`, testCollectionID)

var fullImagePayload = fmt.Sprintf(`
{
	"id": "%s",
	"collection_id": "%s",
	"filename": "some-image-name",
	"license": {
		"title": "Open Government Licence v3.0",
		"href": "https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/"
	},
	"type": "chart",
	"state": "published",
	"upload": {
		"path": "images/025a789c-533f-4ecf-a83b-65412b96b2b7/image-name.png"
	},
	"download": {
		"png": {
			"1920x1080": {
				"size": 1024,
				"href": "http://download.ons.gov.uk/images/042e216a-7822-4fa0-a3d6-e3f5248ffc35/image-name.png",
				"public": "my-public-bucket",
				"private": "my-private-bucket"
			}
		}
	}
}
`, testPublishedImageID, testCollectionID)

var createdImage = models.Image{
	ID:           testCreatedImageID,
	CollectionID: testCollectionID,
	Filename:     "some-image-name",
	License: &models.License{
		Title: "Open Government Licence v3.0",
		Href:  "https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/",
	},
	Type:  "chart",
	State: models.StateCreated.String(),
}

var publishedImage = models.Image{
	ID:           testPublishedImageID,
	CollectionID: testCollectionID,
	Filename:     "some-image-name",
	License: &models.License{
		Title: "Open Government Licence v3.0",
		Href:  "https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/",
	},
	Type: "chart",
	Upload: &models.Upload{
		Path: "images/025a789c-533f-4ecf-a83b-65412b96b2b7/image-name.png",
	},
	Downloads: map[string]map[string]models.Download{
		"png": {
			"1920x1080": models.Download{
				Size:    1024,
				Href:    "http://download.ons.gov.uk/images/042e216a-7822-4fa0-a3d6-e3f5248ffc35/image-name.png",
				Public:  "my-public-bucket",
				Private: "my-private-bucket",
			},
		},
	},
	State: models.StatePublished.String(),
}

var images = models.Images{
	Items:      []models.Image{createdImage, publishedImage},
	Count:      2,
	Limit:      2,
	TotalCount: 2,
	Offset:     0,
}

var emptyImages = models.Images{
	Items:      []models.Image{},
	Count:      0,
	Limit:      0,
	TotalCount: 0,
	Offset:     0,
}

func TestCreateImageHandler(t *testing.T) {

	api.NewID = func() string { return testCreatedImageID }

	Convey("Given an image API that can successfully store valid images in mongoDB", t, func() {
		cfg, err := config.Get()
		So(err, ShouldBeNil)
		mongoDbMock := &mock.MongoServerMock{
			UpsertImageFunc: func(ctx context.Context, id string, image *models.Image) error { return nil },
		}
		imageApi := GetAPIWithMocks(mongoDbMock, cfg)

		Convey("When a valid new image is posted", func() {
			r := httptest.NewRequest(http.MethodPost, "http://localhost:24700/images", bytes.NewBufferString(newImagePayload))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			Convey("Then a newly created image with the new id and provided details is returned with status code 201", func() {
				So(w.Code, ShouldEqual, http.StatusCreated)
				payload, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				retImage := models.Image{}
				err = json.Unmarshal(payload, &retImage)
				So(err, ShouldBeNil)
				So(retImage, ShouldResemble, createdImage)
			})
		})

		Convey("When a valid new image with extra fields is posted", func() {
			r := httptest.NewRequest(http.MethodPost, "http://localhost:24700/images", bytes.NewBufferString(fullImagePayload))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			Convey("Then a newly created image with the new id and provided details is returned with status code 201, ignoring any field that is not supposed to be provided at creation time", func() {
				So(w.Code, ShouldEqual, http.StatusCreated)
				payload, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				retImage := models.Image{}
				err = json.Unmarshal(payload, &retImage)
				So(err, ShouldBeNil)
				So(retImage, ShouldResemble, createdImage)
			})
		})

		Convey("When a new image with an invalid state fields is posted", func() {
			r := httptest.NewRequest(http.MethodPost, "http://localhost:24700/images", bytes.NewBufferString(newImageInvalidState))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			Convey("Then a newly created image with the new id and provided details is returned with status code 201, ignoring the state and any field that is not supposed to be provided at creation time", func() {
				So(w.Code, ShouldEqual, http.StatusCreated)
				payload, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				retImage := models.Image{}
				err = json.Unmarshal(payload, &retImage)
				So(err, ShouldBeNil)
				So(retImage, ShouldResemble, createdImage)
			})
		})

		Convey("Posting an image with a filename longer than the maximum allowed results in BadRequest response", func() {
			r := httptest.NewRequest(http.MethodPost, "http://localhost:24700/images", bytes.NewBufferString(newImagePayloadTooLong))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusBadRequest)
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

	Convey("Given an image API with mongoDB that fails to insert images", t, func() {
		cfg, err := config.Get()
		So(err, ShouldBeNil)
		mongoDbMock := &mock.MongoServerMock{
			UpsertImageFunc: func(ctx context.Context, id string, image *models.Image) error { return errMongoDB },
		}
		imageApi := GetAPIWithMocks(mongoDbMock, cfg)

		Convey("When a new image is posted a 501 InternalServerError status code is returned", func() {
			r := httptest.NewRequest(http.MethodPost, "http://localhost:24700/images", bytes.NewBufferString(newImagePayload))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusInternalServerError)
		})
	})
}

func TestGetImageHandler(t *testing.T) {

	Convey("Given an image API with mongoDB returning 'created' and 'published' images", t, func() {
		cfg, err := config.Get()
		So(err, ShouldBeNil)
		mongoDbMock := &mock.MongoServerMock{
			GetImageFunc: func(ctx context.Context, id string) (*models.Image, error) {
				switch id {
				case testCreatedImageID:
					return &createdImage, nil
				case testPublishedImageID:
					return &publishedImage, nil
				default:
					return nil, apierrors.ErrImageNotFound
				}
			},
		}
		imageApi := GetAPIWithMocks(mongoDbMock, cfg)

		Convey("When an existing 'created' image is requested with the valid Collection-Id context value", func() {
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:24700/images/%s", testCreatedImageID), nil)
			r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			Convey("Then the expected image is returned with status code 200", func() {
				So(w.Code, ShouldEqual, http.StatusOK)
				payload, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				retImage := models.Image{}
				err = json.Unmarshal(payload, &retImage)
				So(err, ShouldBeNil)
				So(retImage, ShouldResemble, createdImage)
			})
		})

		Convey("When an existing 'published' image is requested without a Collection-Id context value", func() {
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:24700/images/%s", testPublishedImageID), nil)
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			Convey("Then the published image is returned with status code 200", func() {
				So(w.Code, ShouldEqual, http.StatusOK)
				payload, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				retImage := models.Image{}
				err = json.Unmarshal(payload, &retImage)
				So(err, ShouldBeNil)
				So(retImage, ShouldResemble, publishedImage)
			})
		})

		Convey("Missing a valid Collection-Id when requesting a non-published image results in a BadRequest response", func() {
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:24700/images/%s", testCreatedImageID), nil)
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("Requesting an inexistent image ID results in a NotFound response", func() {
			r := httptest.NewRequest(http.MethodGet, "http://localhost:24700/images/inexistent", nil)
			r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusNotFound)
		})
	})

}

func TestGetImagesHandler(t *testing.T) {

	Convey("Given an image API with mongoDB returning the images for the testing Collection-Id", t, func() {
		cfg, err := config.Get()
		So(err, ShouldBeNil)
		mongoDbMock := &mock.MongoServerMock{
			GetImagesFunc: func(ctx context.Context, collectionID string) ([]models.Image, error) {
				if collectionID == testCollectionID {
					return []models.Image{createdImage, publishedImage}, nil
				}
				return []models.Image{}, nil
			},
		}
		imageApi := GetAPIWithMocks(mongoDbMock, cfg)

		Convey("When existing images are requested with a valid Collection-Id context and query value", func() {
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:24700/images?collection_id=%s", testCollectionID), nil)
			r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			Convey("Then the expected images are returned with status code 200", func() {
				So(w.Code, ShouldEqual, http.StatusOK)
				payload, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				retImages := models.Images{}
				err = json.Unmarshal(payload, &retImages)
				So(err, ShouldBeNil)
				So(retImages, ShouldResemble, images)
			})
		})

		Convey("When inexistent images are requested with a valid Collection-Id context and query value", func() {
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:24700/images?collection_id=%s", "otherCollectionId"), nil)
			r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), "otherCollectionId"))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			Convey("Then an empty list of images is returned with status code 200", func() {
				So(w.Code, ShouldEqual, http.StatusOK)
				payload, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				retImages := models.Images{}
				err = json.Unmarshal(payload, &retImages)
				So(err, ShouldBeNil)
				So(retImages, ShouldResemble, emptyImages)
			})
		})

		Convey("Missing a valid Collection-Id header when requesting images results in a BadRequest response", func() {
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:24700/images?collection_id=%s", testCollectionID), nil)
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("Missing a valid Collection-Id query parmeter when requesting images results in a BadRequest response", func() {
			r := httptest.NewRequest(http.MethodGet, "http://localhost:24700/images", nil)
			r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("Providing different values for Collection-Id header and query parameter when requesting images results in a BadRequest response", func() {
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:24700/images?collection_id=%s", testCollectionID), nil)
			r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), "otherCollectionId"))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

	})
}

func TestUpdateImageHandler(t *testing.T) {
	Convey("Given an image API with empty mongoDB", t, func() {
		cfg, err := config.Get()
		So(err, ShouldBeNil)
		imageApi := GetAPIWithMocks(&mock.MongoServerMock{}, cfg)

		Convey("Calling update image results in 501 NotImplemented", func() {
			r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s", testCreatedImageID), nil)
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusNotImplemented)
		})
	})
}

func TestPublishImageHandler(t *testing.T) {
	Convey("Given an image API with empty mongoDB", t, func() {
		cfg, err := config.Get()
		So(err, ShouldBeNil)
		imageApi := GetAPIWithMocks(&mock.MongoServerMock{}, cfg)

		Convey("Calling publish image results in 501 NotImplemented", func() {
			r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/publish", testCreatedImageID), nil)
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusNotImplemented)
		})
	})
}
