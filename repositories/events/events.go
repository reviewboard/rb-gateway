package events

import (
	"bytes"
	"encoding/json"
)

const (
	PushEvent string = "push"
)

var (
	exists = struct{}{}

	validEvents = map[string]struct{}{
		PushEvent: exists,
	}
)

func IsValidEvent(event string) bool {
	_, ok := validEvents[event]
	return ok
}

// The payload type.
type Payload interface {
	// The event the payload is for.
	//
	// This will be included in the marshalled payload.
	GetEvent() string

	// The repository name this payload is for.
	//
	// This will be included in the marshalled payload.
	GetRepository() string

	// Extra content to be included in the payload.
	//
	// This returns a tuple of `(fieldName, fieldContents)`.
	//
	// The contents must be `json.Marshal`-able.
	GetContent() (string, interface{})
}

// Marshal a payload into a JSON blob.
func MarshalPayload(p Payload) ([]byte, error) {
	buffer := bytes.NewBufferString("{\n")

	b, err := json.Marshal(p.GetEvent())
	if err != nil {
		return nil, err
	}

	buffer.WriteString("\t\"event\": ")
	buffer.Write(b)
	buffer.WriteString(",\n")

	b, err = json.Marshal(p.GetRepository())
	if err != nil {
		return nil, err
	}

	buffer.WriteString("\t\"repository\": ")
	buffer.Write(b)

	fieldName, content := p.GetContent()
	if fieldName != "" {
		buffer.WriteString(",\n")

		b, err = json.Marshal(fieldName)
		if err != nil {
			return nil, err
		}

		buffer.WriteString("\t")
		buffer.Write(b)
		buffer.WriteString(": ")

		b, err = json.MarshalIndent(content, "\t", "\t")
		if err != nil {
			return nil, err
		}

		buffer.Write(b)
	}

	buffer.WriteString("\n}\n")

	return buffer.Bytes(), nil
}
