package event_test

import (
	"testing"

	"github.com/ONSdigital/dp-image-api/event"
	"github.com/ONSdigital/dp-image-api/event/mock"
	"github.com/pkg/errors"
	"github.com/smartystreets/goconvey/convey"
)

var errMarshal = errors.New("Marshal error")

func TestAvroProducer(t *testing.T) {
	convey.Convey("Given a successful message producer mock", t, func() {
		// channel to capture messages sent.
		outputChannel := make(chan []byte, 1)

		// bytes to send
		avroBytes := []byte("hello world")

		// mock that represents a marshaller
		marshallerMock := &mock.MarshallerMock{
			MarshalFunc: func(_ interface{}) ([]byte, error) {
				return avroBytes, nil
			},
		}

		// eventProducer under test
		eventProducer := event.NewAvroProducer(outputChannel, marshallerMock)

		convey.Convey("when ImageUploaded is called with a nil event", func() {
			err := eventProducer.ImageUploaded(nil)

			convey.Convey("then the expected error is returned", func() {
				convey.So(err.Error(), convey.ShouldEqual, "event required but was nil")
			})

			convey.Convey("and marshaller is never called", func() {
				convey.So(marshallerMock.MarshalCalls(), convey.ShouldHaveLength, 0)
			})
		})

		convey.Convey("when ImagePublished is called with a nil event", func() {
			err := eventProducer.ImagePublished(nil)

			convey.Convey("then the expected error is returned", func() {
				convey.So(err.Error(), convey.ShouldEqual, "event required but was nil")
			})

			convey.Convey("and marshaller is never called", func() {
				convey.So(marshallerMock.MarshalCalls(), convey.ShouldHaveLength, 0)
			})
		})

		convey.Convey("When ImageUploaded is called on the event producer", func() {
			event := &event.ImageUploaded{
				ImageID:  "myImage",
				Path:     "myPath",
				Filename: "filename.png",
			}
			err := eventProducer.ImageUploaded(event)

			convey.Convey("The expected event is available on the output channel", func() {
				convey.So(err, convey.ShouldBeNil)

				messageBytes := <-outputChannel
				close(outputChannel)
				convey.So(messageBytes, convey.ShouldResemble, avroBytes)
			})
		})

		convey.Convey("When ImagePublished is called on the event producer", func() {
			event := &event.ImagePublished{
				SrcPath:      "path/private/image.png",
				DstPath:      "path/public/img.png",
				ImageID:      "123",
				ImageVariant: "original",
			}
			err := eventProducer.ImagePublished(event)

			convey.Convey("The expected event is available on the output channel", func() {
				convey.So(err, convey.ShouldBeNil)

				messageBytes := <-outputChannel
				close(outputChannel)
				convey.So(messageBytes, convey.ShouldResemble, avroBytes)
			})
		})
	})

	convey.Convey("Given a message producer mock that fails to marshall", t, func() {
		// mock that represents a marshaller
		marshallerMock := &mock.MarshallerMock{
			MarshalFunc: func(_ interface{}) ([]byte, error) {
				return nil, errMarshal
			},
		}

		// eventProducer under test, without out channel because nothing is expected to be sent
		eventProducer := event.NewAvroProducer(nil, marshallerMock)

		convey.Convey("When ImageUploaded is called on the event producer", func() {
			event := &event.ImageUploaded{
				ImageID:  "myImage",
				Path:     "myPath",
				Filename: "filename.png",
			}
			err := eventProducer.ImageUploaded(event)

			convey.Convey("The expected error is returned", func() {
				convey.So(err, convey.ShouldResemble, errMarshal)
			})
		})

		convey.Convey("When ImagePublished is called on the event producer", func() {
			event := &event.ImagePublished{
				SrcPath:      "path/private/image.png",
				DstPath:      "path/public/img.png",
				ImageID:      "123",
				ImageVariant: "original",
			}
			err := eventProducer.ImagePublished(event)

			convey.Convey("The expected error is returned", func() {
				convey.So(err, convey.ShouldResemble, errMarshal)
			})
		})
	})
}
