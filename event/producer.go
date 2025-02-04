package event

import (
	"github.com/pkg/errors"
)

//go:generate moq -out mock/marshaller.go -pkg mock . Marshaller

// AvroProducer of output events.
type AvroProducer struct {
	out        chan []byte
	marshaller Marshaller
}

// Marshaller marshals events into messages.
type Marshaller interface {
	Marshal(s interface{}) ([]byte, error)
}

// NewAvroProducer returns a new instance of AvroProducer.
func NewAvroProducer(outputChannel chan []byte, marshaller Marshaller) *AvroProducer {
	return &AvroProducer{
		out:        outputChannel,
		marshaller: marshaller,
	}
}

// ImageUploaded produces a new ImageUploaded event.
func (producer *AvroProducer) ImageUploaded(event *ImageUploaded) error {
	if event == nil {
		return errors.New("event required but was nil")
	}
	return producer.marshalAndSendEvent(event)
}

// ImagePublished produces a new ImagePublished event.
func (producer *AvroProducer) ImagePublished(event *ImagePublished) error {
	if event == nil {
		return errors.New("event required but was nil")
	}
	return producer.marshalAndSendEvent(event)
}

// marshalAndSendEvent is a generic function that marshals avro events and sends them to the output channel of the producer
func (producer *AvroProducer) marshalAndSendEvent(event interface{}) error {
	bytes, err := producer.marshaller.Marshal(event)
	if err != nil {
		return err
	}
	producer.out <- bytes
	return nil
}
