// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package internal

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

type Event struct {
	Type string
	Data []byte
}

type EventReader struct {
	reader *bufio.Reader
}

func NewEventReader(reader io.Reader) EventReader {
	return EventReader{reader: bufio.NewReader(reader)}
}

func (stream EventReader) Read() (Event, error) {
	var event Event

	for {
		rawLine, err := stream.reader.ReadBytes('\n')
		if err != nil {
			return event, fmt.Errorf("read line: %w", err)
		}
		rawLine = bytes.TrimRight(rawLine, "\r\n")
		rawLine = bytes.TrimSpace(rawLine)
		if len(rawLine) == 0 { // Delimiter of events.
			return event, nil
		}

		key, value, _ := bytes.Cut(rawLine, []byte(":"))
		value = bytes.TrimSpace(value)
		switch string(key) {
		case "event":
			event.Type = string(value)
		case "data":
			if event.Data != nil {
				event.Data = append(event.Data, '\n')
			}
			event.Data = append(event.Data, value...)
		default:
			// Ignore unknown fields and comments.
		}
	}
}
