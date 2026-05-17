package logging

import (
	"bytes"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
)

const serviceName = "goroute"

const defaultSubscriberBufferSize = 256

var defaultBroadcaster = NewStreamBroadcaster(defaultSubscriberBufferSize)

type StreamBroadcaster struct {
	mu      sync.RWMutex
	nextID  uint64
	buffer  int
	streams map[uint64]chan string
}

type lineBroadcastWriter struct {
	mu          sync.Mutex
	broadcaster *StreamBroadcaster
	pending     []byte
}

func NewStreamBroadcaster(buffer int) *StreamBroadcaster {
	if buffer < 1 {
		buffer = 1
	}

	return &StreamBroadcaster{
		buffer:  buffer,
		streams: make(map[uint64]chan string),
	}
}

func DefaultBroadcaster() *StreamBroadcaster {
	return defaultBroadcaster
}

func (b *StreamBroadcaster) Subscribe() (<-chan string, func()) {
	if b == nil {
		ch := make(chan string)
		close(ch)
		return ch, func() {}
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	id := b.nextID
	b.nextID++

	ch := make(chan string, b.buffer)
	b.streams[id] = ch

	return ch, func() {
		b.mu.Lock()
		current, ok := b.streams[id]
		if ok {
			delete(b.streams, id)
			close(current)
		}
		b.mu.Unlock()
	}
}

func (b *StreamBroadcaster) Publish(line string) {
	if b == nil || line == "" {
		return
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.streams {
		select {
		case ch <- line:
		default:
		}
	}
}

func (b *StreamBroadcaster) Writer() io.Writer {
	if b == nil {
		return io.Discard
	}

	return &lineBroadcastWriter{broadcaster: b}
}

func (w *lineBroadcastWriter) Write(p []byte) (int, error) {
	if w == nil || w.broadcaster == nil {
		return len(p), nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	buffer := append(w.pending, p...)
	for {
		index := bytes.IndexByte(buffer, '\n')
		if index < 0 {
			break
		}

		line := strings.TrimRight(string(buffer[:index]), "\r")
		w.broadcaster.Publish(line)
		buffer = buffer[index+1:]
	}

	w.pending = append(w.pending[:0], buffer...)
	return len(p), nil
}

func New(env string) zerolog.Logger {
	return NewWithWriter(env, os.Stdout)
}

func NewWithWriter(env string, writer io.Writer) zerolog.Logger {
	zerolog.TimestampFunc = func() time.Time {
		return time.Now().UTC()
	}

	output := writer
	if !isProductionEnv(env) {
		output = zerolog.ConsoleWriter{
			Out:        writer,
			TimeFormat: time.RFC3339,
			NoColor:    !supportsColor(writer),
		}
	}

	output = io.MultiWriter(output, DefaultBroadcaster().Writer())

	return zerolog.New(output).With().Timestamp().Str("service", serviceName).Logger()
}

func isProductionEnv(env string) bool {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "prod", "production":
		return true
	default:
		return false
	}
}

func supportsColor(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}

	fd := file.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}
