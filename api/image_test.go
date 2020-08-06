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
	"sync"
	"testing"
	"time"

	dpauth "github.com/ONSdigital/dp-authorisation/auth"
	"github.com/ONSdigital/dp-image-api/api"
	"github.com/ONSdigital/dp-image-api/api/mock"
	"github.com/ONSdigital/dp-image-api/apierrors"
	"github.com/ONSdigital/dp-image-api/config"
	"github.com/ONSdigital/dp-image-api/event"
	"github.com/ONSdigital/dp-image-api/models"
	"github.com/ONSdigital/dp-image-api/schema"
	kafka "github.com/ONSdigital/dp-kafka"
	"github.com/ONSdigital/dp-kafka/kafkatest"
	"github.com/ONSdigital/dp-net/handlers"
	dphttp "github.com/ONSdigital/dp-net/http"

	. "github.com/smartystreets/goconvey/convey"
)

// Contants for testing
const (
	testUserAuthToken      = "UserToken"
	testImageID1           = "imageImageID1"
	testImageID2           = "imageImageID2"
	testImageCreatedID     = "imageCreatedID"
	testImageUploadedID    = "imageUploadedID"
	testImageImportingID   = "imageImportingID"
	testImagePublishedID   = "imagePublishedID"
	testCollectionID1      = "1234"
	testVariantOriginal    = "original"
	testVariantAlternative = "bw1024"
	testCollectionID2      = "4321"
	testUploadPath         = "s3://images/newimage.png"
	longName               = "Llanfairpwllgwyngyllgogerychwyrndrobwllllantysiliogogogoch"
	testLockID             = "image-myID-123456789"
	testDownloadType       = "originally uploaded file"
	testPrivateHref        = "http://download.ons.gov.uk/images/imageImageID2/original/some-image-name"
)

var (
	testSize             = 1024
	testWidth            = 123
	testHeight           = 321
	testImportStarted    = time.Date(2020, time.April, 26, 8, 5, 52, 0, time.UTC)
	testImportCompleted  = time.Date(2020, time.April, 26, 8, 7, 32, 0, time.UTC)
	testPublishStarted   = time.Date(2020, time.April, 26, 9, 51, 3, 0, time.UTC)
	testPublishCompleted = time.Date(2020, time.April, 26, 10, 1, 28, 0, time.UTC)
)

var errMongoDB = errors.New("MongoDB generic error")

// Empty JSON Payload
var emptyJsonPayload = `{}`

// New Image Payload without any extra field.
var newImagePayloadFmt = `{
	"collection_id": "%s",
	"filename": "%s",
	"license": {
		"title": "Open Government Licence v3.0",
		"href": "https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/"
	},
	"type": "chart"
}`

// New Image Payload with extra state field.
var newImageWithStatePayloadFmt = `{
	"collection_id": "%s",
	"filename": "some-image-name",
	"state": "%s",
	"license": {
		"title": "Open Government Licence v3.0",
		"href": "https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/"
	},
	"type": "chart"
}`

// Image Upload Payload without any extra field.
var imageUploadPayloadFmt = `{
	"path": "%s"
}`

// Image Upload Payload with extra state field.
var imageUploadWithStatePayloadFmt = `{
	"upload": {
		"path": "%s"
	},
	"state": "%s"
}`

// New Image Download Payload without any extra field.
var newImageDownloadPayloadFmt = `{
	"id": "%s",
	"type": "%s",
	"state": "%s",
    "import_started" : "2020-04-26T08:05:52Z"
}`

// Image Download Payload with all possible fields.
var (
	imageDownloadPayloadFmt = `{
		"size": 1024,
		"palette": "bw",
		"type": "%s",
		"width": 123,
		"height": 321,
		"public": false,
		"href": "http://someURL.com",
		"private": "my-private-bucket",
		"state": "published",
		"error": "someError",
		"import_started": "2020-04-26T08:05:52Z",
		"import_completed": "2020-04-26T08:07:32+00:00",
		"publish_started": "2020-04-26T09:51:03-00:00",
		"publish_completed": "2020-04-26T10:01:28Z"
	}`
	imageDownloadPayload = fmt.Sprintf(imageDownloadPayloadFmt, testDownloadType)
)

// Full Image payload, containing all possible fields
var (
	fullImagePayloadFmt = `{
		"id": "%s",
		"collection_id": "%s",
		"filename": "some-image-name",
		"license": {
			"title": "Open Government Licence v3.0",
			"href": "https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/"
		},
		"type": "chart",
		"state": "%s",
		"upload": {
			"path": "images/025a789c-533f-4ecf-a83b-65412b96b2b7/image-name.png"
		},
		"downloads": {
			"original": {
				"size": 1024,
				"palette": "bw",
				"type": "originally uploaded file",
				"width": 123,
				"height": 321,
				"public": true,
				"href": "http://download.ons.gov.uk/images/042e216a-7822-4fa0-a3d6-e3f5248ffc35/image-name.png",
				"private": "my-private-bucket",
				"state": "published",
				"error": "",
				"import_started": "2020-04-26T08:05:52Z",
				"import_completed": "2020-04-26T08:07:32+00:00",
				"publish_started": "2020-04-26T09:51:03-00:00",
				"publish_completed": "2020-04-26T10:01:28Z"
			}
		}
	}`
	fullImagePayload = fmt.Sprintf(fullImagePayloadFmt, testImageID2, testCollectionID1, models.StatePublished.String())
)

// DB model corresponding to an image in the provided state, without any download variant
func dbImage(state models.State) *models.Image {
	return &models.Image{
		ID:           testImageID1,
		CollectionID: testCollectionID1,
		Filename:     "some-image-name",
		License: &models.License{
			Title: "Open Government Licence v3.0",
			Href:  "https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/",
		},
		Type:  "chart",
		State: state.String(),
	}
}

// dbDownloadUpdate is the model corresponding to an image download variant update
func dbDownloadUpdate(variantState models.DownloadState) models.Download {
	return models.Download{
		Size:    &testSize,
		Type:    "originally uploaded file",
		Width:   &testWidth,
		Height:  &testHeight,
		Private: "my-private-bucket",
		State:   variantState.String(),
	}
}

func dbDownload(variantState models.DownloadState) models.Download {
	return dbDownloadWithID("", testVariantOriginal, variantState)
}

func dbDownloadWithID(id, variant string, state models.DownloadState) models.Download {
	download := models.Download{
		ID:    variant,
		Type:  "originally uploaded file",
		State: state.String(),
		Links: &models.DownloadLinks{
			Self:  fmt.Sprintf("/images/%s/downloads/%s", id, variant),
			Image: fmt.Sprintf("/images/%s", id),
		},
	}
	if state != models.StateDownloadImporting {
		download.Size = &testSize
		download.Width = &testWidth
		download.Height = &testHeight
		download.Href = dbDownloadHRef(state)
		download.Private = "my-private-bucket"
	}
	if state == models.StateDownloadImporting {
		download.ImportStarted = &testImportStarted
	}
	if state == models.StateDownloadImported {
		download.ImportStarted = &testImportStarted
		download.ImportCompleted = &testImportCompleted
	}
	if state == models.StateDownloadPublished {
		download.ImportStarted = &testImportStarted
		download.ImportCompleted = &testImportCompleted
		download.PublishStarted = &testPublishStarted
	}
	if state == models.StateDownloadPublished {
		download.ImportStarted = &testImportStarted
		download.ImportCompleted = &testImportCompleted
		download.PublishStarted = &testPublishStarted
		download.PublishCompleted = &testPublishCompleted
	}
	return download
}

func dbDownloadHRef(variantState models.DownloadState) string {
	switch variantState {
	case models.StateDownloadPublished:
		return testPrivateHref
	default:
		return ""
	}

}

// API model with state removed
func updateImage() *models.Image {
	image := dbImage(models.StateCreated)
	image.State = ""
	return image
}

// DB model corresponding to an image in the provided state, with a download variant in the provided state.
func dbFullImage(state models.State) *models.Image {
	return &models.Image{
		ID:           testImageID2,
		CollectionID: testCollectionID1,
		Filename:     "some-image-name",
		License: &models.License{
			Title: "Open Government Licence v3.0",
			Href:  "https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/",
		},
		Type:  "chart",
		State: state.String(),
		Upload: &models.Upload{
			Path: "images/025a789c-533f-4ecf-a83b-65412b96b2b7/image-name.png",
		},
	}
}

func dbFullImageWithDownloads(state models.State, downloads ...models.Download) *models.Image {
	image := dbFullImage(state)
	image.Downloads = map[string]models.Download{}
	for _, d := range downloads {
		image.Downloads[d.ID] = d
	}
	return image
}

// API model corresponding to an image in the provided state
func apiFullImage(state models.State) *models.Image {
	return &models.Image{
		ID:           testImageID2,
		CollectionID: testCollectionID1,
		Filename:     "some-image-name",
		License: &models.License{
			Title: "Open Government Licence v3.0",
			Href:  "https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/",
		},
		Type:  "chart",
		State: state.String(),
		Upload: &models.Upload{
			Path: "images/025a789c-533f-4ecf-a83b-65412b96b2b7/image-name.png",
		},
	}
}

// DB model corresponding to an image in uploaded state
func dbUploadedImage() *models.Image {
	return dbImage(models.StateUploaded)
}

// API model corresponding to dbCreatedImage
func createdImage() *models.Image {
	return dbImage(models.StateCreated)
}

// API model corresponding to dbImage with StateImported
func importedImage() *models.Image {
	return dbImage(models.StateImported)
}

// DB model corresponding to an image in created state without collectionID
func dbCreatedImageNoCollectionID() *models.Image {
	return &models.Image{
		ID:       testImageID1,
		Filename: "some-image-name",
		License: &models.License{
			Title: "Open Government Licence v3.0",
			Href:  "https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/",
		},
		Type:  "chart",
		State: models.StateCreated.String(),
	}
}

// API model corresponding to dbCreatedImageNoCollectionID
func createdImageNoCollectionID() *models.Image {
	return dbCreatedImageNoCollectionID()
}

// API model corresponding to dbCreatedImageNoCollectionID, without a state value
func updateImageNoCollectionID() *models.Image {
	image := *dbCreatedImageNoCollectionID()
	image.State = ""
	return &image
}

var imagesWithCollectionID1 = models.Images{
	Items:      []models.Image{*createdImage(), *importedImage(), *apiFullImage(models.StatePublished)},
	Count:      3,
	Limit:      3,
	TotalCount: 3,
	Offset:     0,
}

var allImages = models.Images{
	Items:      []models.Image{*createdImage(), *createdImageNoCollectionID(), *importedImage(), *apiFullImage(models.StatePublished)},
	Count:      4,
	Limit:      4,
	TotalCount: 4,
	Offset:     0,
}

var emptyImages = models.Images{
	Items:      []models.Image{},
	Count:      0,
	Limit:      0,
	TotalCount: 0,
	Offset:     0,
}

// kafkaProducer mock which exposes Channels function returning empty channels
// to be used on tests that are not supposed to send any kafka message
var kafkaStubProducer = &kafkatest.IProducerMock{
	ChannelsFunc: func() *kafka.ProducerChannels {
		return &kafka.ProducerChannels{}
	},
}

func TestGetImagesHandler(t *testing.T) {

	Convey("Given an image API in publishing mode", t, func() {
		cfg, err := config.Get()
		cfg.IsPublishing = true
		So(err, ShouldBeNil)
		doTestGetImagesHandler(cfg)
	})

	Convey("Given an image API in web mode", t, func() {
		cfg, err := config.Get()
		cfg.IsPublishing = false
		So(err, ShouldBeNil)
		doTestGetImagesHandler(cfg)
	})
}

func doTestGetImagesHandler(cfg *config.Config) {

	Convey("And an image API with mongoDB returning the images as expected according to the collectionID filter", func() {
		cfg, err := config.Get()
		So(err, ShouldBeNil)
		mongoDBMock := &mock.MongoServerMock{
			GetImagesFunc: func(ctx context.Context, collectionID string) ([]models.Image, error) {
				if collectionID == testCollectionID1 {
					return []models.Image{*dbImage(models.StateCreated), *dbImage(models.StateImported), *dbFullImageWithDownloads(models.StatePublished, dbDownload(models.StateDownloadPublished))}, nil
				} else if collectionID == "" {
					return []models.Image{*dbImage(models.StateCreated), *dbCreatedImageNoCollectionID(), *dbImage(models.StateImported), *dbFullImageWithDownloads(models.StatePublished, dbDownload(models.StateDownloadPublished))}, nil
				} else {
					return []models.Image{}, nil
				}
			},
		}
		authHandlerMock := &mock.AuthHandlerMock{
			RequireFunc: func(required dpauth.Permissions, handler http.HandlerFunc) http.HandlerFunc {
				return handler
			},
		}
		imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

		Convey("When existing images are requested with a valid Collection-Id context and query parameter value", func() {
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:24700/images?collection_id=%s", testCollectionID1), nil)
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)

			Convey("Then the expected images are returned with status code 200", func() {
				So(w.Code, ShouldEqual, http.StatusOK)
				payload, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				retImages := models.Images{}
				err = json.Unmarshal(payload, &retImages)
				So(err, ShouldBeNil)
				So(retImages, ShouldResemble, imagesWithCollectionID1)
			})
		})

		Convey("When inexistent images are requested with a valid Collection-Id context and query parameter value", func() {
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:24700/images?collection_id=%s", "otherCollectionId"), nil)
			r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), "otherCollectionId"))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
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

		Convey("When no collection_id query parameter value is provided", func() {
			r := httptest.NewRequest(http.MethodGet, "http://localhost:24700/images", nil)
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)

			Convey("Then the full list of images is returned with status code 200", func() {
				So(w.Code, ShouldEqual, http.StatusOK)
				payload, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				retImages := models.Images{}
				err = json.Unmarshal(payload, &retImages)
				So(err, ShouldBeNil)
				So(retImages, ShouldResemble, allImages)
			})
		})

		Convey("Providing different values for Collection-Id header and query parameter is allowed", func() {
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:24700/images?collection_id=%s", testCollectionID1), nil)
			r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), "otherCollectionId"))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusOK)
		})

	})

	Convey("And an image API with mongoDB returning an error", func() {
		cfg, err := config.Get()
		So(err, ShouldBeNil)
		mongoDBMock := &mock.MongoServerMock{
			GetImagesFunc: func(ctx context.Context, collectionID string) ([]models.Image, error) {
				return []models.Image{}, errMongoDB
			},
		}
		authHandlerMock := &mock.AuthHandlerMock{
			RequireFunc: func(required dpauth.Permissions, handler http.HandlerFunc) http.HandlerFunc {
				return handler
			},
		}
		imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

		Convey("Then when images are requested, a 500 error is returned", func() {
			r := httptest.NewRequest(http.MethodGet, "http://localhost:24700/images", nil)
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusInternalServerError)
		})
	})
}

func TestCreateImageHandler(t *testing.T) {

	api.NewID = func() string { return testImageID1 }

	Convey("Given an image API in publishing mode that can successfully store valid images in mongoDB", t, func() {
		cfg, err := config.Get()
		So(err, ShouldBeNil)
		cfg.IsPublishing = true

		mongoDBMock := &mock.MongoServerMock{
			UpsertImageFunc: func(ctx context.Context, id string, image *models.Image) error { return nil },
		}

		authHandlerMock := &mock.AuthHandlerMock{
			RequireFunc: func(required dpauth.Permissions, handler http.HandlerFunc) http.HandlerFunc {
				return handler
			},
		}

		imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

		Convey("When a valid new image is posted", func() {
			r := httptest.NewRequest(http.MethodPost, "http://localhost:24700/images", bytes.NewBufferString(
				fmt.Sprintf(newImagePayloadFmt, testCollectionID1, "some-image-name")))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)

			Convey("Then a newly created image with the new id and provided details is returned with status code 201", func() {
				So(w.Code, ShouldEqual, http.StatusCreated)
				payload, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				retImage := models.Image{}
				err = json.Unmarshal(payload, &retImage)
				So(err, ShouldBeNil)
				So(retImage, ShouldResemble, *createdImage())
			})
		})

		Convey("When a valid new image with extra fields is posted", func() {
			r := httptest.NewRequest(http.MethodPost, "http://localhost:24700/images", bytes.NewBufferString(fullImagePayload))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			Convey("Then a newly created image with the new id and provided details is returned with status code 201, ignoring any field that is not supposed to be provided at creation time", func() {
				So(w.Code, ShouldEqual, http.StatusCreated)
				payload, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				retImage := models.Image{}
				err = json.Unmarshal(payload, &retImage)
				So(err, ShouldBeNil)
				So(retImage, ShouldResemble, *createdImage())
			})
		})

		Convey("When a new image with an invalid state fields is posted", func() {
			r := httptest.NewRequest(http.MethodPost, "http://localhost:24700/images", bytes.NewBufferString(
				fmt.Sprintf(newImageWithStatePayloadFmt, testCollectionID1, "invalidState")))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			Convey("Then a newly created image with the new id and provided details is returned with status code 201, ignoring the state and any field that is not supposed to be provided at creation time", func() {
				So(w.Code, ShouldEqual, http.StatusCreated)
				payload, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				retImage := models.Image{}
				err = json.Unmarshal(payload, &retImage)
				So(err, ShouldBeNil)
				So(retImage, ShouldResemble, *createdImage())
			})
		})

		Convey("Posting an image with a filename longer than the maximum allowed results in BadRequest response", func() {
			r := httptest.NewRequest(http.MethodPost, "http://localhost:24700/images", bytes.NewBufferString(
				fmt.Sprintf(newImagePayloadFmt, testCollectionID1, longName)))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("Posting an empty image (without collection id) results in BadRequest response", func() {
			r := httptest.NewRequest(http.MethodPost, "http://localhost:24700/images", bytes.NewBufferString(emptyJsonPayload))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("An empty request body results in a BadRequest response", func() {
			r := httptest.NewRequest(http.MethodPost, "http://localhost:24700/images", nil)
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("An invalid request body results in a BadRequest response", func() {
			r := httptest.NewRequest(http.MethodPost, "http://localhost:24700/images", bytes.NewBufferString("invalidJson"))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})
	})

	Convey("Given an image API with mongoDB that fails to insert images", t, func() {
		cfg, err := config.Get()
		So(err, ShouldBeNil)
		cfg.IsPublishing = true

		mongoDBMock := &mock.MongoServerMock{
			UpsertImageFunc: func(ctx context.Context, id string, image *models.Image) error { return errMongoDB },
		}

		authHandlerMock := &mock.AuthHandlerMock{
			RequireFunc: func(required dpauth.Permissions, handler http.HandlerFunc) http.HandlerFunc {
				return handler
			},
		}
		imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

		Convey("When a new image is posted a 500 InternalServerError status code is returned", func() {
			r := httptest.NewRequest(http.MethodPost, "http://localhost:24700/images", bytes.NewBufferString(
				fmt.Sprintf(newImagePayloadFmt, testCollectionID1, "some-image-name")))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusInternalServerError)
		})
	})
}

func TestGetImageHandler(t *testing.T) {

	Convey("Given an image API in publishing mode", t, func() {
		cfg, err := config.Get()
		cfg.IsPublishing = true
		So(err, ShouldBeNil)
		doTestGetImageHandler(cfg)
	})

	Convey("Given an image API in web mode", t, func() {
		cfg, err := config.Get()
		cfg.IsPublishing = false
		So(err, ShouldBeNil)
		doTestGetImageHandler(cfg)
	})
}

func doTestGetImageHandler(cfg *config.Config) {

	Convey("And an image API with mongoDB returning 'created' and 'published' images", func() {

		mongoDBMock := &mock.MongoServerMock{
			GetImageFunc: func(ctx context.Context, id string) (*models.Image, error) {
				switch id {
				case testImageID1:
					return dbImage(models.StateCreated), nil
				case testImageID2:
					return dbFullImageWithDownloads(models.StatePublished, dbDownload(models.StateDownloadPublished)), nil
				default:
					return nil, apierrors.ErrImageNotFound
				}
			},
		}
		authHandlerMock := &mock.AuthHandlerMock{
			RequireFunc: func(required dpauth.Permissions, handler http.HandlerFunc) http.HandlerFunc {
				return handler
			},
		}
		imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

		Convey("When an existing 'created' image is requested with the valid Collection-Id context value", func() {
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:24700/images/%s", testImageID1), nil)
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			Convey("Then the expected image is returned with status code 200", func() {
				So(w.Code, ShouldEqual, http.StatusOK)
				payload, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				retImage := models.Image{}
				err = json.Unmarshal(payload, &retImage)
				So(err, ShouldBeNil)
				So(retImage, ShouldResemble, *createdImage())
			})
		})

		Convey("When an existing 'published' image is requested without a Collection-Id context value", func() {
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:24700/images/%s", testImageID2), nil)
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			Convey("Then the published image is returned with status code 200", func() {
				So(w.Code, ShouldEqual, http.StatusOK)
				payload, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				retImage := models.Image{}
				err = json.Unmarshal(payload, &retImage)
				So(err, ShouldBeNil)
				So(retImage, ShouldResemble, *apiFullImage(models.StatePublished))
			})
		})

		Convey("Requesting an inexistent image ID results in a NotFound response", func() {
			r := httptest.NewRequest(http.MethodGet, "http://localhost:24700/images/inexistent", nil)
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusNotFound)
		})
	})
}

func TestUpdateImageHandler(t *testing.T) {

	Convey("Given a valid config, auth handler, kafka producer", t, func() {
		cfg, err := config.Get()
		So(err, ShouldBeNil)
		authHandlerMock := &mock.AuthHandlerMock{
			RequireFunc: func(required dpauth.Permissions, handler http.HandlerFunc) http.HandlerFunc {
				return handler
			},
		}

		Convey("And an empty MongoDB mock", func() {
			mongoMock := &mock.MongoServerMock{}
			imageApi := GetAPIWithMocks(cfg, mongoMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

			Convey("Calling update image with an invalid body results in 400 response", func() {
				r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s", testImageID1), bytes.NewBufferString("wrong"))
				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
				w := httptest.NewRecorder()
				imageApi.Router.ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})

			Convey("Calling update image with an image that has a different id results in 400 response", func() {
				r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s", testImageID1), bytes.NewBufferString(fullImagePayload))
				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
				w := httptest.NewRecorder()
				imageApi.Router.ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})

			Convey("Calling update image with an image that has a filename that is too long results in 400 response", func() {
				r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s", testImageID1), bytes.NewBufferString(
					fmt.Sprintf(`{"filename": "%s"}`, longName)))
				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
				w := httptest.NewRecorder()
				imageApi.Router.ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})
		})

		Convey("And an image in created state in MongoDB", func() {
			mongoDBMock := &mock.MongoServerMock{
				GetImageFunc: func(ctx context.Context, id string) (*models.Image, error) {
					return dbImage(models.StateCreated), nil
				},
				UpdateImageFunc: func(ctx context.Context, id string, image *models.Image) (bool, error) {
					return true, nil
				},
				AcquireImageLockFunc: func(ctx context.Context, id string) (string, error) { return testLockID, nil },
				UnlockImageFunc:      func(id string) error { return nil },
			}
			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

			Convey("Calling update image providing a valid image results in 200 OK response with the expected image updated to mongoDB, and the full image returned in the response body", func() {
				r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s", testImageID1), bytes.NewBufferString(
					fmt.Sprintf(newImageWithStatePayloadFmt, testCollectionID1, models.StateCreated.String())))
				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
				w := httptest.NewRecorder()
				imageApi.Router.ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusOK)
				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 2)
				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(mongoDBMock.GetImageCalls()[1].ID, ShouldEqual, testImageID1)
				So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.UpdateImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(*mongoDBMock.UpdateImageCalls()[0].Image, ShouldResemble, *updateImage()) // Image without state
				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.UnlockImageCalls()[0].LockID, ShouldEqual, testLockID)

				retImage := &models.Image{}
				err := json.Unmarshal(w.Body.Bytes(), retImage)
				So(err, ShouldBeNil)
				So(*retImage, ShouldResemble, *dbImage(models.StateCreated)) // full image with state
			})

			Convey("Calling update image with an image without collectionID results in 200 OK response, only the provided fields are updated, and the full image is", func() {
				r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s", testImageID1), bytes.NewBufferString(
					fmt.Sprintf(newImageWithStatePayloadFmt, "", models.StateCreated.String())))
				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
				w := httptest.NewRecorder()
				imageApi.Router.ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusOK)
				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 2)
				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(mongoDBMock.GetImageCalls()[1].ID, ShouldEqual, testImageID1)
				So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.UpdateImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(*mongoDBMock.UpdateImageCalls()[0].Image, ShouldResemble, *updateImageNoCollectionID())
				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)

				retImage := &models.Image{}
				err := json.Unmarshal(w.Body.Bytes(), retImage)
				So(err, ShouldBeNil)
				So(*retImage, ShouldResemble, *dbImage(models.StateCreated)) // full image with state and collectionID
			})
		})

		Convey("And MongoDB returning imageNotFound error", func() {
			mongoDBMock := &mock.MongoServerMock{
				GetImageFunc: func(ctx context.Context, id string) (*models.Image, error) {
					return nil, apierrors.ErrImageNotFound
				},
				AcquireImageLockFunc: func(ctx context.Context, id string) (string, error) { return testLockID, nil },
				UnlockImageFunc:      func(id string) error { return nil },
			}
			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

			Convey("Calling update image with an inexistent image id results in 404 response", func() {
				r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s", "inexistent"), bytes.NewBufferString(
					fmt.Sprintf(newImageWithStatePayloadFmt, testCollectionID1, models.StateCreated.String())))
				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
				w := httptest.NewRecorder()
				imageApi.Router.ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusNotFound)
				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, "inexistent")
				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
			})
		})

		Convey("And MongoDB failing to get an image", func() {
			mongoDBMock := &mock.MongoServerMock{
				GetImageFunc: func(ctx context.Context, id string) (*models.Image, error) {
					return nil, errors.New("internal mongoDB error")
				},
				AcquireImageLockFunc: func(ctx context.Context, id string) (string, error) { return testLockID, nil },
				UnlockImageFunc:      func(id string) error { return nil },
			}
			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

			Convey("Calling update image results in 500 response", func() {
				r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s", testImageID1), bytes.NewBufferString(
					fmt.Sprintf(newImageWithStatePayloadFmt, testCollectionID1, models.StateCreated.String())))
				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
				w := httptest.NewRecorder()
				imageApi.Router.ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusInternalServerError)
				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
			})
		})

		Convey("And an API with a mongoDB containing an image that does not change on update", func() {
			mongoDBMock := &mock.MongoServerMock{
				GetImageFunc: func(ctx context.Context, id string) (*models.Image, error) {
					return dbImage(models.StateCreated), nil
				},
				UpdateImageFunc: func(ctx context.Context, id string, image *models.Image) (bool, error) {
					return false, nil
				},
				AcquireImageLockFunc: func(ctx context.Context, id string) (string, error) { return testLockID, nil },
				UnlockImageFunc:      func(id string) error { return nil },
			}
			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

			Convey("Calling update image results in 200 OK but getImage is called only once", func() {
				r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s", testImageID1), bytes.NewBufferString(
					`{}`))
				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
				w := httptest.NewRecorder()
				imageApi.Router.ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusOK)
				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.UpdateImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(*mongoDBMock.UpdateImageCalls()[0].Image, ShouldResemble, models.Image{ID: testImageID1})
				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
			})
		})

		Convey("And an API with a mongoDB containing an image that fails to update", func() {
			mongoDBMock := &mock.MongoServerMock{
				GetImageFunc: func(ctx context.Context, id string) (*models.Image, error) {
					return dbImage(models.StateCreated), nil
				},
				UpdateImageFunc: func(ctx context.Context, id string, image *models.Image) (bool, error) {
					return false, errors.New("internal mongoDB error")
				},
				AcquireImageLockFunc: func(ctx context.Context, id string) (string, error) { return testLockID, nil },
				UnlockImageFunc:      func(id string) error { return nil },
			}
			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

			Convey("Calling update image results in 500 result", func() {
				r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s", testImageID1), bytes.NewBufferString(
					fmt.Sprintf(newImageWithStatePayloadFmt, testCollectionID1, models.StateCreated.String())))
				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
				w := httptest.NewRecorder()
				imageApi.Router.ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusInternalServerError)
				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.UpdateImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
			})
		})

		Convey("And an image in published state in MongoDB", func() {
			mongoDBMock := &mock.MongoServerMock{
				GetImageFunc: func(ctx context.Context, id string) (*models.Image, error) {
					return dbImage(models.StatePublished), nil
				},
				AcquireImageLockFunc: func(ctx context.Context, id string) (string, error) { return testLockID, nil },
				UnlockImageFunc:      func(id string) error { return nil },
			}
			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

			Convey("Calling update image results in 403 Forbidden response and it is not updated", func() {
				r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s", testImageID2), bytes.NewBufferString(
					fmt.Sprintf(newImageWithStatePayloadFmt, testImageID2, models.StateDeleted.String())))
				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
				w := httptest.NewRecorder()
				imageApi.Router.ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusForbidden)
				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID2)
				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
			})
		})
	})
}

// TODO Replace with test for PUT /images/{image_id}
//func TestUploadImageHandler(t *testing.T) {
//
//	Convey("Given a valid config, auth handler, kafka producer", t, func() {
//		cfg, err := config.Get()
//		So(err, ShouldBeNil)
//		authHandlerMock := &mock.AuthHandlerMock{
//			RequireFunc: func(required dpauth.Permissions, handler http.HandlerFunc) http.HandlerFunc {
//				return handler
//			},
//		}
//		uploadedProducer := kafkatest.NewMessageProducer(true)
//
//		Convey("And an image in created state in MongoDB", func() {
//			mongoDBMock := &mock.MongoServerMock{
//				GetImageFunc: func(ctx context.Context, id string) (*models.Image, error) {
//					return dbImage(models.StateCreated), nil
//				},
//				UpdateImageFunc: func(ctx context.Context, id string, image *models.Image) (bool, error) {
//					return true, nil
//				},
//				AcquireImageLockFunc: func(ctx context.Context, id string) (string, error) { return testLockID, nil },
//				UnlockImageFunc:      func(id string) error { return nil },
//			}
//			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, uploadedProducer, kafkaStubProducer)
//
//			Convey("Calling image upload with a valid image upload results in 200 OK response with the expected image provided to mongoDB and the message sent to kafka producer", func() {
//				r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/upload", testImageID1), bytes.NewBufferString(
//					fmt.Sprintf(imageUploadPayloadFmt, testUploadPath)))
//				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
//				w := httptest.NewRecorder()
//
//				sentBytes := serveHTTPAndReadKafka(w, r, imageApi, uploadedProducer, 1)
//				So(w.Code, ShouldEqual, http.StatusOK)
//				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
//				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
//				So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 1)
//				So(mongoDBMock.UpdateImageCalls()[0].ID, ShouldEqual, testImageID1)
//				So(*mongoDBMock.UpdateImageCalls()[0].Image, ShouldResemble, models.Image{
//					Upload: &models.Upload{
//						Path: testUploadPath,
//					},
//					State: "uploaded",
//				})
//				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
//				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
//
//				Convey("And the expected avro event is sent to the corresponding kafka output channel", func() {
//					expectedBytes, err := schema.ImageUploadedEvent.Marshal(&event.ImageUploaded{
//						ImageID: testImageID1,
//						Path:    testUploadPath,
//					})
//					So(err, ShouldBeNil)
//					So(expectedBytes, ShouldResemble, sentBytes[0])
//				})
//			})
//
//			Convey("Calling image upload results in a 500 InternalError response when an invalid image uploaded event is generated, and the image is not updated in mongoDB", func() {
//				api.ImageUploadedEvent = func(imageID, uploadPath string) *event.ImageUploaded {
//					return nil
//				}
//				imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)
//				r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/upload", testImageID1), bytes.NewBufferString(
//					fmt.Sprintf(imageUploadPayloadFmt, testUploadPath)))
//				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
//				r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
//				w := httptest.NewRecorder()
//				imageApi.Router.ServeHTTP(w, r)
//				So(w.Code, ShouldEqual, http.StatusInternalServerError)
//				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
//				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
//				So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 0)
//			})
//
//			Convey("Calling image upload on an image that is already uploaded returns a 403 response.", func() {
//				mongoDBMock := &mock.MongoServerMock{
//					GetImageFunc: func(ctx context.Context, id string) (*models.Image, error) {
//						return dbUploadedImage(), nil
//					},
//					AcquireImageLockFunc: func(ctx context.Context, id string) (string, error) { return testLockID, nil },
//					UnlockImageFunc:      func(id string) error { return nil },
//				}
//				imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)
//
//				Convey("Calling 'publish image' results in 403 Forbidden response", func() {
//					r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/upload", testImageID1), bytes.NewBufferString(
//						fmt.Sprintf(imageUploadPayloadFmt, testUploadPath)))
//					r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
//					r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
//					w := httptest.NewRecorder()
//					imageApi.Router.ServeHTTP(w, r)
//					So(w.Code, ShouldEqual, http.StatusForbidden)
//					So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
//					So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
//					So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
//					So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
//				})
//			})
//		})
//	})
//}

func TestGetDownloadsHandler(t *testing.T) {
	//TODO Test Get downloads
	// TEST no image exists
	// TEST no downloads exist
}

func TestCreateDownloadHandler(t *testing.T) {

	// Convey - Given an API in publishing mode with an existing image in publishing

	Convey("Given an image API in publishing mode with existing valid images stored in a mongoDB mock", t, func() {
		cfg, err := config.Get()
		So(err, ShouldBeNil)
		cfg.IsPublishing = true

		mongoDBMock := &mock.MongoServerMock{
			GetImageFunc: func(ctx context.Context, id string) (*models.Image, error) {
				switch id {
				case testImageCreatedID:
					return dbImage(models.StateCreated), nil
				case testImageUploadedID:
					return dbFullImage(models.StateUploaded), nil
				case testImageImportingID:
					return dbFullImageWithDownloads(models.StateImporting, dbDownload(models.StateDownloadImported)), nil
				case testImagePublishedID:
					return dbFullImageWithDownloads(models.StatePublished, dbDownload(models.StateDownloadPublished)), nil
				default:
					return nil, apierrors.ErrImageNotFound
				}
			},
			UpsertImageFunc:      func(ctx context.Context, id string, image *models.Image) error { return nil },
			AcquireImageLockFunc: func(ctx context.Context, id string) (string, error) { return testLockID, nil },
			UnlockImageFunc:      func(id string) error { return nil },
		}

		authHandlerMock := &mock.AuthHandlerMock{
			RequireFunc: func(required dpauth.Permissions, handler http.HandlerFunc) http.HandlerFunc {
				return handler
			},
		}

		imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

		Convey("When a valid new download is posted to an image in an 'uploaded' state", func() {
			r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/downloads", testImageUploadedID), bytes.NewBufferString(
				fmt.Sprintf(newImageDownloadPayloadFmt, testVariantOriginal, testDownloadType, models.StateDownloadImporting.String())))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)

			Convey("Then a newly created download is returned with status code 201", func() {
				So(w.Code, ShouldEqual, http.StatusCreated)
				payload, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				retDownload := models.Download{}
				err = json.Unmarshal(payload, &retDownload)
				So(err, ShouldBeNil)
				So(retDownload, ShouldResemble, dbDownloadWithID(testImageUploadedID, testVariantOriginal, models.StateDownloadImporting))
			})
		})

		Convey("When a new download with an imported state is posted to an image in an 'uploaded' state", func() {
			r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/downloads", testImageUploadedID), bytes.NewBufferString(
				fmt.Sprintf(newImageDownloadPayloadFmt, testVariantOriginal, testDownloadType, models.StateDownloadImported.String())))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)

			Convey("Then a status code 403 (Forbidden) is returned", func() {
				So(w.Code, ShouldEqual, http.StatusForbidden)
			})
		})

		Convey("When a new download with an invalid state is posted to an image in an 'uploaded' state", func() {
			r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/downloads", testImageUploadedID), bytes.NewBufferString(
				fmt.Sprintf(newImageDownloadPayloadFmt, testVariantOriginal, testDownloadType, "stateWibbling")))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)

			Convey("Then a status code 400 (BadRequest) is returned", func() {
				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})
		})

		Convey("When a valid new download is posted to an image in an 'importing' state", func() {
			r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/downloads", testImageImportingID), bytes.NewBufferString(
				fmt.Sprintf(newImageDownloadPayloadFmt, testVariantAlternative, testDownloadType, models.StateDownloadImporting.String())))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)

			Convey("Then a newly created download is returned with status code 201", func() {
				So(w.Code, ShouldEqual, http.StatusCreated)
				payload, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				retDownload := models.Download{}
				err = json.Unmarshal(payload, &retDownload)
				So(err, ShouldBeNil)
				So(retDownload, ShouldResemble, dbDownloadWithID(testImageImportingID, testVariantAlternative, models.StateDownloadImporting))
			})
		})

		Convey("When a download with an already existing ID is posted to an image in an 'importing' state", func() {
			r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/downloads", testImageImportingID), bytes.NewBufferString(
				fmt.Sprintf(newImageDownloadPayloadFmt, testVariantOriginal, testDownloadType, models.StateDownloadImporting.String())))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)

			Convey("Then a status code 403 (Forbidden) is returned", func() {
				So(w.Code, ShouldEqual, http.StatusForbidden)
			})
		})

		Convey("When a valid new download is posted to an image in a 'published' state", func() {
			r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/downloads", testImagePublishedID), bytes.NewBufferString(
				fmt.Sprintf(newImageDownloadPayloadFmt, testVariantAlternative, testDownloadType, models.StateDownloadImporting.String())))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)

			Convey("Then a status code 403 (Forbidden) is returned", func() {
				So(w.Code, ShouldEqual, http.StatusForbidden)
			})
		})

		Convey("When a valid new download is posted to an image with an ID not found in mongodb", func() {
			r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/downloads", "nonExistantID"), bytes.NewBufferString(
				fmt.Sprintf(newImageDownloadPayloadFmt, testVariantAlternative, testDownloadType, models.StateDownloadImporting.String())))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)

			Convey("Then a status code 404 (NotFound) is returned", func() {
				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})

	})

	Convey("Given an image API in publishing mode with a mongoDB that fails to update", t, func() {
		cfg, err := config.Get()
		So(err, ShouldBeNil)
		cfg.IsPublishing = true

		mongoDBMock := &mock.MongoServerMock{
			GetImageFunc:         func(ctx context.Context, id string) (*models.Image, error) { return dbImage(models.StateUploaded), nil },
			UpsertImageFunc:      func(ctx context.Context, id string, image *models.Image) error { return errMongoDB },
			AcquireImageLockFunc: func(ctx context.Context, id string) (string, error) { return testLockID, nil },
			UnlockImageFunc:      func(id string) error { return nil },
		}

		authHandlerMock := &mock.AuthHandlerMock{
			RequireFunc: func(required dpauth.Permissions, handler http.HandlerFunc) http.HandlerFunc {
				return handler
			},
		}

		imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

		Convey("When a valid new download is posted to an image in a 'uploaded' state", func() {
			r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/downloads", testImageUploadedID), bytes.NewBufferString(
				fmt.Sprintf(newImageDownloadPayloadFmt, testVariantOriginal, testDownloadType, models.StateDownloadImporting.String())))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)

			Convey("Then a status code 500 (InternalServerError) is returned", func() {
				So(w.Code, ShouldEqual, http.StatusInternalServerError)
			})
		})

	})

	Convey("Given an image API in publishing mode with a mongoDB that fails to lock", t, func() {
		cfg, err := config.Get()
		So(err, ShouldBeNil)
		cfg.IsPublishing = true

		mongoDBMock := &mock.MongoServerMock{
			GetImageFunc:         func(ctx context.Context, id string) (*models.Image, error) { return dbImage(models.StateUploaded), nil },
			UpsertImageFunc:      func(ctx context.Context, id string, image *models.Image) error { return nil },
			AcquireImageLockFunc: func(ctx context.Context, id string) (string, error) { return "", errors.New("fail to lock") },
			UnlockImageFunc:      func(id string) error { return nil },
		}

		authHandlerMock := &mock.AuthHandlerMock{
			RequireFunc: func(required dpauth.Permissions, handler http.HandlerFunc) http.HandlerFunc {
				return handler
			},
		}

		imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

		Convey("When a valid new download is posted to an image in a 'uploaded' state", func() {
			r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/downloads", testImageUploadedID), bytes.NewBufferString(
				fmt.Sprintf(newImageDownloadPayloadFmt, testVariantOriginal, testDownloadType, models.StateDownloadImporting.String())))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)

			Convey("Then a status code 500 (InternalServerError) is returned", func() {
				So(w.Code, ShouldEqual, http.StatusInternalServerError)
			})
		})

	})
}

func TestUpdateDownloadHandler(t *testing.T) {
	// TODO Update test to use PUT /images/{id}/downloads/{variant}
	//
	//	Convey("Given a valid config, auth handler", t, func() {
	//		cfg, err := config.Get()
	//		So(err, ShouldBeNil)
	//		authHandlerMock := &mock.AuthHandlerMock{
	//			RequireFunc: func(required dpauth.Permissions, handler http.HandlerFunc) http.HandlerFunc {
	//				return handler
	//			},
	//		}
	//
	//		Convey("And an image in 'uploaded' state in MongoDB, with a download variant in 'pending' state", func() {
	//			mongoDBMock := &mock.MongoServerMock{
	//				GetImageFunc: func(ctx context.Context, id string) (*models.Image, error) {
	//					return dbFullImage(models.StateUploaded, models.StateDownloadPending), nil
	//				},
	//				UpdateImageFunc: func(ctx context.Context, id string, image *models.Image) (bool, error) {
	//					return true, nil
	//				},
	//				AcquireImageLockFunc: func(ctx context.Context, id string) (string, error) { return testLockID, nil },
	//				UnlockImageFunc:      func(id string) error { return nil },
	//			}
	//			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)
	//
	//			Convey("Calling 'import variant' for the image results in 200 OK response and the image is updated as expected", func() {
	//				t0 := time.Now()
	//				r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/downloads/%s/import", testImageID2, testVariantOriginal), nil)
	//				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
	//				r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
	//				w := httptest.NewRecorder()
	//				imageApi.Router.ServeHTTP(w, r)
	//				So(w.Code, ShouldEqual, http.StatusOK)
	//				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
	//				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID2)
	//				So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 1)
	//				So(mongoDBMock.UpdateImageCalls()[0].ID, ShouldResemble, testImageID2)
	//				update := mongoDBMock.UpdateImageCalls()[0].Image
	//				So(update.State, ShouldResemble, models.StateImporting.String())
	//				So(update.Downloads[testVariantOriginal].State, ShouldResemble, models.StateDownloadImporting.String())
	//				So(*update.Downloads[testVariantOriginal].ImportStarted, ShouldHappenOnOrBetween, t0, time.Now())
	//				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
	//				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
	//			})
	//		})
	//
	//		Convey("And an image in 'uploaded' state in MongoDB, without any download variants", func() {
	//			mongoDBMock := &mock.MongoServerMock{
	//				GetImageFunc: func(ctx context.Context, id string) (*models.Image, error) {
	//					return dbImage(models.StateUploaded), nil
	//				},
	//				AcquireImageLockFunc: func(ctx context.Context, id string) (string, error) { return testLockID, nil },
	//				UnlockImageFunc:      func(id string) error { return nil },
	//			}
	//			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)
	//
	//			Convey("Calling 'import variant' for the existing image without variants results in 404 Not found response", func() {
	//				r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/downloads/%s/import", testImageID1, testVariantOriginal), nil)
	//				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
	//				r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
	//				w := httptest.NewRecorder()
	//				imageApi.Router.ServeHTTP(w, r)
	//				So(w.Code, ShouldEqual, http.StatusNotFound)
	//				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
	//				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
	//				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
	//				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
	//			})
	//		})
	//
	//		Convey("And a MongoDB returning error on GetImage", func() {
	//			mongoDBMock := &mock.MongoServerMock{
	//				GetImageFunc: func(ctx context.Context, id string) (*models.Image, error) {
	//					return nil, errMongoDB
	//				},
	//				AcquireImageLockFunc: func(ctx context.Context, id string) (string, error) { return testLockID, nil },
	//				UnlockImageFunc:      func(id string) error { return nil },
	//			}
	//			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)
	//
	//			Convey("Calling 'import variant' for an inexistent image results in 500 InternalServerError response", func() {
	//				r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/downloads/%s/import", testImageID1, testVariantOriginal), nil)
	//				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
	//				r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
	//				w := httptest.NewRecorder()
	//				imageApi.Router.ServeHTTP(w, r)
	//				So(w.Code, ShouldEqual, http.StatusInternalServerError)
	//				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
	//				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
	//				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
	//				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
	//			})
	//		})
	//
	//		Convey("And a MongoDB returning error on UploadImage", func() {
	//			mongoDBMock := &mock.MongoServerMock{
	//				GetImageFunc: func(ctx context.Context, id string) (*models.Image, error) {
	//					return dbFullImage(models.StateUploaded, models.StateDownloadPending), nil
	//				},
	//				UpdateImageFunc: func(ctx context.Context, id string, image *models.Image) (bool, error) {
	//					return false, errMongoDB
	//				},
	//				AcquireImageLockFunc: func(ctx context.Context, id string) (string, error) { return testLockID, nil },
	//				UnlockImageFunc:      func(id string) error { return nil },
	//			}
	//			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)
	//
	//			Convey("Calling 'import variant' results in 500 InternalServerError response", func() {
	//				r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/downloads/%s/import", testImageID1, testVariantOriginal), nil)
	//				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
	//				r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
	//				w := httptest.NewRecorder()
	//				imageApi.Router.ServeHTTP(w, r)
	//				So(w.Code, ShouldEqual, http.StatusInternalServerError)
	//				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
	//				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
	//				So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 1)
	//				So(mongoDBMock.UpdateImageCalls()[0].ID, ShouldEqual, testImageID1)
	//				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
	//				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
	//			})
	//		})
	//
	//		Convey("And an image in 'uploaded' state in MongoDB, with a download variant in 'published' state", func() {
	//			mongoDBMock := &mock.MongoServerMock{
	//				GetImageFunc: func(ctx context.Context, id string) (*models.Image, error) {
	//					return dbFullImage(models.StateUploaded, models.StateDownloadPublished), nil
	//				},
	//				AcquireImageLockFunc: func(ctx context.Context, id string) (string, error) { return testLockID, nil },
	//				UnlockImageFunc:      func(id string) error { return nil },
	//			}
	//			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)
	//
	//			Convey("Calling 'import variant' for the image results in 403 Forbidden response because the download variant state transition is not allowed", func() {
	//				r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/downloads/%s/import", testImageID2, testVariantOriginal), nil)
	//				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
	//				r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
	//				w := httptest.NewRecorder()
	//				imageApi.Router.ServeHTTP(w, r)
	//				So(w.Code, ShouldEqual, http.StatusForbidden)
	//				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
	//				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID2)
	//				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
	//				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
	//			})
	//		})
	//
	//		Convey("And an image in 'created' state in MongoDB, with a download variant in 'pending' state", func() {
	//			mongoDBMock := &mock.MongoServerMock{
	//				GetImageFunc: func(ctx context.Context, id string) (*models.Image, error) {
	//					return dbFullImage(models.StateCreated, models.StateDownloadPending), nil
	//				},
	//				AcquireImageLockFunc: func(ctx context.Context, id string) (string, error) { return testLockID, nil },
	//				UnlockImageFunc:      func(id string) error { return nil },
	//			}
	//			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)
	//
	//			Convey("Calling 'import variant' for the image results in 403 Forbidden response because the image state transition is not allowed", func() {
	//				r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/downloads/%s/import", testImageID2, testVariantOriginal), nil)
	//				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
	//				r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
	//				w := httptest.NewRecorder()
	//				imageApi.Router.ServeHTTP(w, r)
	//				So(w.Code, ShouldEqual, http.StatusForbidden)
	//				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
	//				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID2)
	//				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
	//				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
	//			})
	//		})
	//	})
}

func TestPublishImageHandler(t *testing.T) {

	Convey("Given a valid config, auth handler, kafka producer", t, func() {
		cfg, err := config.Get()
		So(err, ShouldBeNil)
		authHandlerMock := &mock.AuthHandlerMock{
			RequireFunc: func(required dpauth.Permissions, handler http.HandlerFunc) http.HandlerFunc {
				return handler
			},
		}

		Convey("And an image in 'imported' state in MongoDB", func() {
			expectedPathOriginal := fmt.Sprintf("/images/%s/original/some-image-name", testImageID1)
			expectedPathPngW500 := fmt.Sprintf("/images/%s/png_w500/some-image-name", testImageID1)
			mongoDBMock := &mock.MongoServerMock{
				GetImageFunc: func(ctx context.Context, id string) (*models.Image, error) {
					image := dbImage(models.StateImported)
					image.Downloads = map[string]models.Download{"original": {Href: expectedPathOriginal}, "png_w500": {Href: expectedPathPngW500}}
					return image, nil
				},
				UpdateImageFunc: func(ctx context.Context, id string, image *models.Image) (bool, error) {
					return true, nil
				},
				AcquireImageLockFunc: func(ctx context.Context, id string) (string, error) { return testLockID, nil },
				UnlockImageFunc:      func(id string) error { return nil },
			}

			Convey("Calling 'publish image' results in 200 OK response with the expected image state update to mongoDB and the message sent to kafka producer", func() {
				publishedProducer := kafkatest.NewMessageProducer(true)
				imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, publishedProducer)
				r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/publish", testImageID1), nil)
				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
				r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
				w := httptest.NewRecorder()
				sentBytes := serveHTTPAndReadKafka(w, r, imageApi, publishedProducer, 2)
				So(w.Code, ShouldEqual, http.StatusOK)
				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.UpdateImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(mongoDBMock.UpdateImageCalls()[0].Image.State, ShouldEqual, models.StatePublished.String())
				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)

				Convey("And the expected avro event is sent to the corresponding kafka output channel, with the expected source and dest paths", func() {
					// Note: the paths correspond to the path part of DownloadHrefFmt format "http://<host>/images/<imageID>/<variantName>/<fileName>"
					expectedBytesOriginal, err := schema.ImagePublishedEvent.Marshal(&event.ImagePublished{
						SrcPath: expectedPathOriginal,
						DstPath: expectedPathOriginal,
					})
					So(err, ShouldBeNil)
					expectedBytesPngW500, err := schema.ImagePublishedEvent.Marshal(&event.ImagePublished{
						SrcPath: expectedPathPngW500,
						DstPath: expectedPathPngW500,
					})
					So(err, ShouldBeNil)
					validateExpectedBytes(sentBytes, [][]byte{expectedBytesOriginal, expectedBytesPngW500})
				})
			})

			Convey("Calling 'publish image' with a 500 InternalError response when an invalid image published event is generated", func() {
				api.ImagePublishedEvent = func(path string) *event.ImagePublished {
					return nil
				}
				publishedProducer := kafkatest.NewMessageProducer(true)
				imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, publishedProducer)
				r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/publish", testImageID1), nil)
				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
				r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
				w := httptest.NewRecorder()
				imageApi.Router.ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusInternalServerError)
				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 0)
			})
		})

		Convey("And an image with invalid filename, which results in an invalid href for the download variants", func() {
			mongoDBMock := &mock.MongoServerMock{
				GetImageFunc: func(ctx context.Context, id string) (*models.Image, error) {
					image := dbImage(models.StateImported)
					image.Filename = "a££$(50y4534%£$||}{}"
					image.Downloads = map[string]models.Download{"original": {}, "png_w500": {}}
					return image, nil
				},
				AcquireImageLockFunc: func(ctx context.Context, id string) (string, error) { return testLockID, nil },
				UnlockImageFunc:      func(id string) error { return nil },
			}
			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

			Convey("Calling 'publish image' results in 500 response", func() {
				r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/publish", testImageID1), nil)
				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
				r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
				w := httptest.NewRecorder()
				imageApi.Router.ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusInternalServerError)
				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
			})
		})

		Convey("And an image in 'created' state in MongoDB (non-publishable)", func() {
			mongoDBMock := &mock.MongoServerMock{
				GetImageFunc: func(ctx context.Context, id string) (*models.Image, error) {
					return dbImage(models.StateCreated), nil
				},
				AcquireImageLockFunc: func(ctx context.Context, id string) (string, error) { return testLockID, nil },
				UnlockImageFunc:      func(id string) error { return nil },
			}
			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

			Convey("Calling 'publish image' results in 403 Forbidden response", func() {
				r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/publish", testImageID1), nil)
				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
				r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
				w := httptest.NewRecorder()
				imageApi.Router.ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusForbidden)
				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
			})
		})

		Convey("And MongoDB failing to get an image", func() {
			mongoDBMock := &mock.MongoServerMock{
				GetImageFunc: func(ctx context.Context, id string) (*models.Image, error) {
					return nil, errors.New("internal mongoDB error")
				},
				AcquireImageLockFunc: func(ctx context.Context, id string) (string, error) { return testLockID, nil },
				UnlockImageFunc:      func(id string) error { return nil },
			}
			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

			Convey("Calling 'publish image' results in 500 response", func() {
				r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/publish", testImageID1), nil)
				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
				w := httptest.NewRecorder()
				imageApi.Router.ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusInternalServerError)
				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
			})
		})

		Convey("And an API with a mongoDB containing an imported image that fails to update", func() {
			mongoDBMock := &mock.MongoServerMock{
				GetImageFunc: func(ctx context.Context, id string) (*models.Image, error) {
					return dbImage(models.StateImported), nil
				},
				UpdateImageFunc: func(ctx context.Context, id string, image *models.Image) (bool, error) {
					return false, errors.New("internal mongoDB error")
				},
				AcquireImageLockFunc: func(ctx context.Context, id string) (string, error) { return testLockID, nil },
				UnlockImageFunc:      func(id string) error { return nil },
			}
			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

			Convey("Calling 'publish image' results in 500 response", func() {
				r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/publish", testImageID1), nil)
				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
				r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
				w := httptest.NewRecorder()
				imageApi.Router.ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusInternalServerError)
				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.UpdateImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(mongoDBMock.UpdateImageCalls()[0].Image.State, ShouldEqual, models.StatePublished.String())
				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
			})
		})
	})
}

// serveHTTPAndReadKafka performs the ServeHTTP with the provided responseRecorder and Request in a parallel go-routine, then reads the bytes
// from the kafka output channel for the provided number of messages, and waits for the ServeHTTP routine to finish.
// The bytes sent to kafka output channel are returned in an array corresponding to each call.
func serveHTTPAndReadKafka(w *httptest.ResponseRecorder, r *http.Request, imageApi *api.API, kafkaProducerMock kafka.IProducer, expectedNumMessages int) [][]byte {
	sentBytes := [][]byte{}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		imageApi.Router.ServeHTTP(w, r)
	}()

	for i := 0; i < expectedNumMessages; i++ {
		s := <-kafkaProducerMock.Channels().Output
		So(s, ShouldNotBeNil)
		sentBytes = append(sentBytes, s)
	}

	wg.Wait()
	return sentBytes
}

// validateExpectedBytes checks that all byte arrays from b1 resemble all byte arrayes from b2, ignoring order
func validateExpectedBytes(bytes1, bytes2 [][]byte) {

	So(len(bytes1), ShouldEqual, len(bytes2))

	// utility function to compare bytes arrays
	bytesEqual := func(b1, b2 []byte) bool {
		for i, b := range b2 {
			if b != b1[i] {
				return false
			}
		}
		return true
	}

	// utility function to find byte array in an array of byte arrays
	findByteArray := func(toFind []byte, b [][]byte) bool {
		for _, comparing := range b {
			if bytesEqual(toFind, comparing) {
				return true
			}
		}
		return false
	}

	for _, b1 := range bytes1 {
		So(findByteArray(b1, bytes2), ShouldBeTrue)
	}
}

// TODO Replace with test using PUT /images/{id}/downloads/{variant}
//func TestCompleteVariantHandler(t *testing.T) {
//
//	Convey("Given a valid config, auth handler", t, func() {
//		cfg, err := config.Get()
//		So(err, ShouldBeNil)
//		authHandlerMock := &mock.AuthHandlerMock{
//			RequireFunc: func(required dpauth.Permissions, handler http.HandlerFunc) http.HandlerFunc {
//				return handler
//			},
//		}
//		mongoDBMock := &mock.MongoServerMock{
//			AcquireImageLockFunc: func(ctx context.Context, id string) (string, error) { return testLockID, nil },
//			UnlockImageFunc:      func(id string) error { return nil },
//		}
//
//		Convey("And an image in 'published' state in MongoDB, with a download variant in 'published' state", func() {
//			mongoDBMock.GetImageFunc = func(ctx context.Context, id string) (*models.Image, error) {
//				return dbFullImage(models.StatePublished, models.StateDownloadPublished), nil
//			}
//			mongoDBMock.UpdateImageFunc = func(ctx context.Context, id string, image *models.Image) (bool, error) { return true, nil }
//			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)
//
//			Convey("Calling 'complete variant' for the image results in 200 OK response and the image is completed", func() {
//				t0 := time.Now()
//				r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/downloads/%s/complete", testImageID2, testVariantOriginal), nil)
//				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
//				r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
//				w := httptest.NewRecorder()
//				imageApi.Router.ServeHTTP(w, r)
//				So(w.Code, ShouldEqual, http.StatusOK)
//				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
//				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID2)
//				So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 1)
//				So(mongoDBMock.UpdateImageCalls()[0].ID, ShouldResemble, testImageID2)
//				update := mongoDBMock.UpdateImageCalls()[0].Image
//				So(update.State, ShouldResemble, models.StateCompleted.String())
//				So(update.Downloads[testVariantOriginal].State, ShouldResemble, models.StateDownloadCompleted.String())
//				So(*update.Downloads[testVariantOriginal].PublishCompleted, ShouldHappenOnOrBetween, t0, time.Now())
//				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
//				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
//			})
//		})
//
//		Convey("And an image in 'published' state in MongoDB, with 2 download variants in 'published' state", func() {
//			mongoDBMock.GetImageFunc = func(ctx context.Context, id string) (*models.Image, error) {
//				i := dbFullImage(models.StatePublished, models.StateDownloadPublished)
//				i.Downloads["png_w500"] = models.Download{State: models.StateDownloadPublished.String()}
//				return i, nil
//			}
//			mongoDBMock.UpdateImageFunc = func(ctx context.Context, id string, image *models.Image) (bool, error) { return true, nil }
//			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)
//
//			Convey("Calling 'complete variant' for the image results in 200 OK response, the variant is completed, but the image is not", func() {
//				t0 := time.Now()
//				r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/downloads/%s/complete", testImageID2, testVariantOriginal), nil)
//				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
//				r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
//				w := httptest.NewRecorder()
//				imageApi.Router.ServeHTTP(w, r)
//				So(w.Code, ShouldEqual, http.StatusOK)
//				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
//				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID2)
//				So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 1)
//				So(mongoDBMock.UpdateImageCalls()[0].ID, ShouldResemble, testImageID2)
//				update := mongoDBMock.UpdateImageCalls()[0].Image
//				So(update.State, ShouldResemble, "")
//				So(update.Downloads[testVariantOriginal].State, ShouldResemble, models.StateDownloadCompleted.String())
//				So(*update.Downloads[testVariantOriginal].PublishCompleted, ShouldHappenOnOrBetween, t0, time.Now())
//				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
//				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
//			})
//		})
//
//		Convey("And an image in 'published' state in MongoDB, with a download variants in 'published' state and another in 'failed' state", func() {
//			mongoDBMock.GetImageFunc = func(ctx context.Context, id string) (*models.Image, error) {
//				i := dbFullImage(models.StatePublished, models.StateDownloadPublished)
//				i.Downloads["png_w500"] = models.Download{State: models.StateDownloadFailed.String()}
//				return i, nil
//			}
//			mongoDBMock.UpdateImageFunc = func(ctx context.Context, id string, image *models.Image) (bool, error) { return true, nil }
//			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)
//
//			Convey("Calling 'complete variant' for the published variant results in 200 OK response, the variant is completed, but the image is set to 'import_failed' state", func() {
//				t0 := time.Now()
//				r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/downloads/%s/complete", testImageID2, testVariantOriginal), nil)
//				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
//				r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
//				w := httptest.NewRecorder()
//				imageApi.Router.ServeHTTP(w, r)
//				So(w.Code, ShouldEqual, http.StatusOK)
//				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
//				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID2)
//				So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 1)
//				So(mongoDBMock.UpdateImageCalls()[0].ID, ShouldResemble, testImageID2)
//				update := mongoDBMock.UpdateImageCalls()[0].Image
//				So(update.State, ShouldResemble, models.StateFailedPublish.String())
//				So(update.Downloads[testVariantOriginal].State, ShouldResemble, models.StateDownloadCompleted.String())
//				So(*update.Downloads[testVariantOriginal].PublishCompleted, ShouldHappenOnOrBetween, t0, time.Now())
//				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
//				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
//			})
//		})
//
//		Convey("And an image in a state in 'created' state", func() {
//			mongoDBMock.GetImageFunc = func(ctx context.Context, id string) (*models.Image, error) {
//				return dbFullImage(models.StateCreated, models.StateDownloadPending), nil
//			}
//			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)
//
//			Convey("Calling 'complete variant' for the image results in 403 Forbidden response and nothing is updated", func() {
//				r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/downloads/%s/complete", testImageID2, testVariantOriginal), nil)
//				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
//				r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
//				w := httptest.NewRecorder()
//				imageApi.Router.ServeHTTP(w, r)
//				So(w.Code, ShouldEqual, http.StatusForbidden)
//				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
//				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID2)
//				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
//				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
//			})
//		})
//
//		Convey("And an image in 'published' state in MongoDB, with a download variants in 'pending' state (unexpected)", func() {
//			mongoDBMock.GetImageFunc = func(ctx context.Context, id string) (*models.Image, error) {
//				return dbFullImage(models.StatePublished, models.StateDownloadPending), nil
//			}
//			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)
//
//			Convey("Calling 'complete variant' for the variant results in 403 Forbiden response and nothing is updated", func() {
//				r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/downloads/%s/complete", testImageID2, testVariantOriginal), nil)
//				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
//				r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
//				w := httptest.NewRecorder()
//				imageApi.Router.ServeHTTP(w, r)
//				So(w.Code, ShouldEqual, http.StatusForbidden)
//				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
//				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID2)
//				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
//				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
//			})
//		})
//
//		Convey("And a MongoDB mock that fails to lock", func() {
//			mongoDBMock := &mock.MongoServerMock{
//				AcquireImageLockFunc: func(ctx context.Context, id string) (string, error) { return "", errMongoDB },
//			}
//			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)
//
//			Convey("Calling 'complete variant' for the variant results in 500 StatusInternalServerError response", func() {
//				r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/downloads/%s/complete", testImageID2, testVariantOriginal), nil)
//				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
//				r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
//				w := httptest.NewRecorder()
//				imageApi.Router.ServeHTTP(w, r)
//				So(w.Code, ShouldEqual, http.StatusInternalServerError)
//				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
//			})
//		})
//
//		Convey("And a MongoDB mock that returns an error on GetImage", func() {
//			mongoDBMock.GetImageFunc = func(ctx context.Context, id string) (*models.Image, error) {
//				return nil, errMongoDB
//			}
//			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)
//
//			Convey("Calling 'complete variant' for the variant results in 500 StatusInternalServerError response", func() {
//				r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/downloads/%s/complete", testImageID2, testVariantOriginal), nil)
//				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
//				r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
//				w := httptest.NewRecorder()
//				imageApi.Router.ServeHTTP(w, r)
//				So(w.Code, ShouldEqual, http.StatusInternalServerError)
//				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
//				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID2)
//				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
//				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
//			})
//		})
//
//		Convey("And a MongoDB mock that returns an error on UpdateImage", func() {
//			mongoDBMock.GetImageFunc = func(ctx context.Context, id string) (*models.Image, error) {
//				return dbFullImage(models.StatePublished, models.StateDownloadPublished), nil
//			}
//			mongoDBMock.UpdateImageFunc = func(ctx context.Context, id string, image *models.Image) (bool, error) { return false, errMongoDB }
//			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)
//
//			Convey("Calling 'complete variant' for the variant results in 500 StatusInternalServerError response", func() {
//				r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/downloads/%s/complete", testImageID2, testVariantOriginal), nil)
//				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
//				r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
//				w := httptest.NewRecorder()
//				imageApi.Router.ServeHTTP(w, r)
//				So(w.Code, ShouldEqual, http.StatusInternalServerError)
//				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
//				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID2)
//				So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 1)
//				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
//				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
//			})
//		})
//	})
//}

func TestUpdateDownloadVariantHandler(t *testing.T) {

	Convey("Given a valid config, auth handler", t, func() {
		cfg, err := config.Get()
		So(err, ShouldBeNil)
		authHandlerMock := &mock.AuthHandlerMock{
			RequireFunc: func(required dpauth.Permissions, handler http.HandlerFunc) http.HandlerFunc {
				return handler
			},
		}
		mongoDBMock := &mock.MongoServerMock{
			AcquireImageLockFunc: func(ctx context.Context, id string) (string, error) { return testLockID, nil },
			UnlockImageFunc:      func(id string) error { return nil },
		}

		Convey("And an empty MongoDB mock", func() {
			mongoMock := &mock.MongoServerMock{}
			imageApi := GetAPIWithMocks(cfg, mongoMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

			Convey("Calling update image with an invalid body results in 400 response", func() {
				r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s/downloads/%s", testImageID1, testVariantOriginal),
					bytes.NewBufferString("wrong"))
				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
				w := httptest.NewRecorder()
				imageApi.Router.ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})
		})

		Convey("And an image in 'importing' state in MongoDB with a download variant in 'importing' state", func() {
			firstCall := true
			mongoDBMock.GetImageFunc = func(ctx context.Context, id string) (*models.Image, error) {
				if firstCall {
					// return before update
					firstCall = false
					return dbFullImageWithDownloads(models.StateImporting, dbDownload(models.StateDownloadImporting)), nil
				}
				// returned after update
				return dbFullImageWithDownloads(models.StateImported, dbDownload(models.StateDownloadImported)), nil
			}
			mongoDBMock.UpdateImageFunc = func(ctx context.Context, id string, image *models.Image) (bool, error) { return true, nil }
			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

			Convey("Calling update variant providing a valid download results in 200 OK response with the expected image update to mongoDB and the full updated image returned", func() {
				t0 := time.Now()
				r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s/downloads/%s", testImageID1, testVariantOriginal),
					bytes.NewBufferString(imageDownloadPayload))
				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
				w := httptest.NewRecorder()
				imageApi.Router.ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusOK)
				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 2)
				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(mongoDBMock.GetImageCalls()[1].ID, ShouldEqual, testImageID1)
				So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.UpdateImageCalls()[0].ID, ShouldEqual, testImageID1)
				update := mongoDBMock.UpdateImageCalls()[0].Image
				So(update.State, ShouldResemble, models.StateImported.String())
				validateImageDownloadImported(update.Downloads[testVariantOriginal], dbDownloadUpdate(models.StateDownloadImported), t0)
				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.UnlockImageCalls()[0].LockID, ShouldEqual, testLockID)
				retDownload := &models.Download{}
				err := json.Unmarshal(w.Body.Bytes(), retDownload)
				So(err, ShouldBeNil)
				expected := dbDownload(models.StateDownloadImported)
				expected.Public = false
				So(*retDownload, ShouldResemble, expected)
			})

			Convey("Calling update variant providing an empty download results in 200 OK response with only the state being updated to Imported", func() {
				t0 := time.Now()
				r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s/downloads/%s", testImageID1, testVariantOriginal),
					bytes.NewBufferString("{}"))
				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
				w := httptest.NewRecorder()
				imageApi.Router.ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusOK)
				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 2)
				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(mongoDBMock.GetImageCalls()[1].ID, ShouldEqual, testImageID1)
				So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.UpdateImageCalls()[0].ID, ShouldEqual, testImageID1)
				update := mongoDBMock.UpdateImageCalls()[0].Image
				So(update.State, ShouldResemble, models.StateImported.String())
				validateImageDownloadImported(update.Downloads[testVariantOriginal],
					models.Download{State: models.StateImported.String()}, t0)
				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.UnlockImageCalls()[0].LockID, ShouldEqual, testLockID)
				retDownload := &models.Download{}
				err := json.Unmarshal(w.Body.Bytes(), retDownload)
				So(err, ShouldBeNil)
				expected := dbDownload(models.StateDownloadImported)
				expected.Public = false
				So(*retDownload, ShouldResemble, expected)
			})

			Convey("Calling update variant with a download variant with a type that does not match the existing type in mongoDB results in 400 response being returned", func() {
				r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s/downloads/%s", testImageID1, testVariantOriginal),
					bytes.NewBufferString(fmt.Sprintf(imageDownloadPayloadFmt, "differentType")))
				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
				w := httptest.NewRecorder()
				imageApi.Router.ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusBadRequest)
				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 0)
			})
		})

		Convey("And an images in 'importing' state in MongoDB with two download variants in 'importing' state", func() {
			firstCall := true
			mongoDBMock.GetImageFunc = func(ctx context.Context, id string) (*models.Image, error) {
				if firstCall {
					// return before update
					firstCall = false
					i := dbFullImageWithDownloads(models.StateImporting, dbDownload(models.StateDownloadImporting))
					i.Downloads["png_w500"] = models.Download{State: models.StateDownloadImporting.String()}
					return i, nil
				}
				// return after update
				i := dbFullImageWithDownloads(models.StateImporting, dbDownload(models.StateDownloadImported))
				i.Downloads["png_w500"] = models.Download{State: models.StateDownloadImporting.String()}
				return i, nil
			}
			mongoDBMock.UpdateImageFunc = func(ctx context.Context, id string, image *models.Image) (bool, error) { return true, nil }
			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

			Convey("Calling update variant providing a valid download results in 200 OK response with the expected image update to mongoDB and the full updated image returned in importing state", func() {
				t0 := time.Now()
				r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s/downloads/%s", testImageID1, testVariantOriginal),
					bytes.NewBufferString(imageDownloadPayload))
				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
				w := httptest.NewRecorder()
				imageApi.Router.ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusOK)
				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 2)
				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(mongoDBMock.GetImageCalls()[1].ID, ShouldEqual, testImageID1)
				So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.UpdateImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.UnlockImageCalls()[0].LockID, ShouldEqual, testLockID)
				update := mongoDBMock.UpdateImageCalls()[0].Image
				So(update.State, ShouldResemble, "")
				validateImageDownloadImported(update.Downloads[testVariantOriginal], dbDownloadUpdate(models.StateDownloadImported), t0)
				retDownload := &models.Download{}
				err := json.Unmarshal(w.Body.Bytes(), retDownload)
				So(err, ShouldBeNil)
				expected := dbDownload(models.StateDownloadImported)
				expected.Public = false
				So(*retDownload, ShouldResemble, expected)
			})
		})

		Convey("And an images in 'importing' state in MongoDB with a download variants in 'importing' state and another one in 'failed' state", func() {
			firstCall := true
			mongoDBMock.GetImageFunc = func(ctx context.Context, id string) (*models.Image, error) {
				if firstCall {
					// return before update
					firstCall = false
					i := dbFullImageWithDownloads(models.StateImporting, dbDownload(models.StateDownloadImporting))
					i.Downloads["png_w500"] = models.Download{State: models.StateDownloadFailed.String()}
					return i, nil
				}
				// return after update
				i := dbFullImageWithDownloads(models.StateFailedImport, dbDownload(models.StateDownloadImported))
				i.Downloads["png_w500"] = models.Download{State: models.StateDownloadFailed.String()}
				return i, nil
			}
			mongoDBMock.UpdateImageFunc = func(ctx context.Context, id string, image *models.Image) (bool, error) { return true, nil }
			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

			Convey("Calling update variant providing a valid download results in 200 OK response with the expected image update to mongoDB and the full updated image returned in importing state", func() {
				t0 := time.Now()
				r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s/downloads/%s", testImageID1, testVariantOriginal),
					bytes.NewBufferString(imageDownloadPayload))
				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
				w := httptest.NewRecorder()
				imageApi.Router.ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusOK)
				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 2)
				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(mongoDBMock.GetImageCalls()[1].ID, ShouldEqual, testImageID1)
				So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.UpdateImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.UnlockImageCalls()[0].LockID, ShouldEqual, testLockID)
				update := mongoDBMock.UpdateImageCalls()[0].Image
				So(update.State, ShouldResemble, models.StateFailedImport.String())
				validateImageDownloadImported(update.Downloads[testVariantOriginal], dbDownloadUpdate(models.StateDownloadImported), t0)
				retDownload := &models.Download{}
				err := json.Unmarshal(w.Body.Bytes(), retDownload)
				So(err, ShouldBeNil)
				expected := dbDownload(models.StateDownloadImported)
				expected.Public = false
				So(*retDownload, ShouldResemble, expected)
			})
		})

		Convey("And a MongoDB mock that fails to lock", func() {
			mongoDBMock := &mock.MongoServerMock{
				AcquireImageLockFunc: func(ctx context.Context, id string) (string, error) { return "", errMongoDB },
			}
			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

			Convey("Calling 'update variant' for the variant results in 500 StatusInternalServerError response", func() {
				r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s/downloads/%s", testImageID1, testVariantOriginal),
					bytes.NewBufferString(imageDownloadPayload))
				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
				w := httptest.NewRecorder()
				imageApi.Router.ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusInternalServerError)
				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
			})
		})

		Convey("And a MongoDB mock that returns an error on GetImage", func() {
			mongoDBMock.GetImageFunc = func(ctx context.Context, id string) (*models.Image, error) {
				return nil, errMongoDB
			}
			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

			Convey("Calling 'import variant' for the variant results in 500 StatusInternalServerError response", func() {
				r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s/downloads/%s", testImageID1, testVariantOriginal),
					bytes.NewBufferString(imageDownloadPayload))
				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
				w := httptest.NewRecorder()
				imageApi.Router.ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusInternalServerError)
				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
			})
		})

		Convey("And a MongoDB mock that returns an error on UpdateImage", func() {
			mongoDBMock.GetImageFunc = func(ctx context.Context, id string) (*models.Image, error) {
				return dbFullImageWithDownloads(models.StateImporting, dbDownload(models.StateDownloadImporting)), nil
			}
			mongoDBMock.UpdateImageFunc = func(ctx context.Context, id string, image *models.Image) (bool, error) { return false, errMongoDB }
			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

			Convey("Calling 'import variant' for the variant results in 500 StatusInternalServerError response", func() {
				r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s/downloads/%s", testImageID1, testVariantOriginal),
					bytes.NewBufferString(imageDownloadPayload))
				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
				w := httptest.NewRecorder()
				imageApi.Router.ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusInternalServerError)
				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 1)
				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
			})
		})

		Convey("And a MongoDB mock that returns an image without the required download variant", func() {
			mongoDBMock.GetImageFunc = func(ctx context.Context, id string) (*models.Image, error) {
				return dbImage(models.StateImporting), nil
			}
			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

			Convey("Calling 'import variant' for the variant results in 404 Not Found response", func() {
				r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s/downloads/%s", testImageID1, testVariantOriginal),
					bytes.NewBufferString(imageDownloadPayload))
				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
				w := httptest.NewRecorder()
				imageApi.Router.ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusNotFound)
				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
			})
		})

		Convey("And a MongoDB mock that returns an image in created state", func() {
			mongoDBMock.GetImageFunc = func(ctx context.Context, id string) (*models.Image, error) {
				return dbFullImageWithDownloads(models.StateCreated, dbDownload(models.StateDownloadPending)), nil
			}
			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

			Convey("Calling 'import variant' for the variant results in 403 Forbiden response", func() {
				r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s/downloads/%s", testImageID1, testVariantOriginal),
					bytes.NewBufferString(imageDownloadPayload))
				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
				w := httptest.NewRecorder()
				imageApi.Router.ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusForbidden)
				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
			})
		})

		Convey("And a MongoDB mock that returns an image in 'importing' state with a download variant in 'pending' state", func() {
			mongoDBMock.GetImageFunc = func(ctx context.Context, id string) (*models.Image, error) {
				return dbFullImageWithDownloads(models.StateImporting, dbDownload(models.StateDownloadPending)), nil
			}
			imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock, kafkaStubProducer, kafkaStubProducer)

			Convey("Calling 'import variant' for the variant results in 403 Forbiden response", func() {
				r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s/downloads/%s", testImageID1, testVariantOriginal),
					bytes.NewBufferString(imageDownloadPayload))
				r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
				w := httptest.NewRecorder()
				imageApi.Router.ServeHTTP(w, r)
				So(w.Code, ShouldEqual, http.StatusForbidden)
				So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
				So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
				So(len(mongoDBMock.AcquireImageLockCalls()), ShouldEqual, 1)
				So(len(mongoDBMock.UnlockImageCalls()), ShouldEqual, 1)
			})
		})

	})
}

// validateImageDownload checks that the image contains the provided expected download variant and the ImportCompleted ws updated between t0 and now
func validateImageDownloadImported(variant, expected models.Download, t0 time.Time) {
	So(variant.State, ShouldResemble, models.StateDownloadImported.String())
	So(*variant.ImportCompleted, ShouldHappenOnOrBetween, t0, time.Now())
	if expected.Size != nil {
		So(*variant.Size, ShouldResemble, *expected.Size)
	} else {
		So(variant.Size, ShouldBeNil)
	}
	So(variant.Type, ShouldResemble, expected.Type)
	if expected.Width != nil {
		So(*variant.Width, ShouldResemble, *expected.Width)
	} else {
		So(variant.Width, ShouldBeNil)
	}
	if expected.Height != nil {
		So(*variant.Height, ShouldResemble, *expected.Height)
	} else {
		So(variant.Height, ShouldBeNil)
	}
	So(variant.Private, ShouldResemble, expected.Private)
}
