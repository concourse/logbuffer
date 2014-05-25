package logbuffer

import (
	"errors"
	"io"
	"sync"
)

type LogBuffer struct {
	content      []byte
	contentMutex *sync.RWMutex

	sinks []io.WriteCloser

	closed        bool
	waitForClosed chan struct{}
}

func NewLogBuffer() *LogBuffer {
	return &LogBuffer{
		contentMutex:  new(sync.RWMutex),
		waitForClosed: make(chan struct{}),
	}
}

func (buffer *LogBuffer) Write(data []byte) (int, error) {
	buffer.contentMutex.Lock()

	buffer.content = append(buffer.content, data...)

	newSinks := []io.WriteCloser{}
	for _, sink := range buffer.sinks {
		_, err := sink.Write(data)
		if err != nil {
			continue
		}

		newSinks = append(newSinks, sink)
	}

	buffer.sinks = newSinks

	buffer.contentMutex.Unlock()

	return len(data), nil
}

func (buffer *LogBuffer) Attach(sink io.WriteCloser) {
	buffer.contentMutex.Lock()

	sink.Write(buffer.content)

	if buffer.closed {
		sink.Close()
	} else {
		buffer.sinks = append(buffer.sinks, sink)
	}

	buffer.contentMutex.Unlock()

	<-buffer.waitForClosed
}

func (buffer *LogBuffer) Close() error {
	buffer.contentMutex.Lock()

	if buffer.closed {
		return errors.New("close twice")
	}

	for _, sink := range buffer.sinks {
		sink.Close()
	}

	buffer.closed = true
	buffer.sinks = nil

	close(buffer.waitForClosed)

	buffer.contentMutex.Unlock()

	return nil
}

func (buffer *LogBuffer) Content() []byte {
	buffer.contentMutex.Lock()
	content := make([]byte, len(buffer.content))
	copy(content, buffer.content)
	buffer.contentMutex.Unlock()

	return content
}
