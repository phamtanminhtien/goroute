package httpapi

import (
	"bufio"
	"bytes"
	"io"
)

func writeSSEStream(w *bodyCaptureResponseWriter, body io.Reader) error {
	reader := bufio.NewReader(body)

	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			if _, writeErr := w.Write(line); writeErr != nil {
				return writeErr
			}
			if bytes.Equal(line, []byte("\n")) || bytes.Equal(line, []byte("\r\n")) {
				w.Flush()
			}
		}

		if err == nil {
			continue
		}
		if err == io.EOF {
			if len(line) > 0 {
				w.Flush()
			}
			return nil
		}
		return err
	}
}
