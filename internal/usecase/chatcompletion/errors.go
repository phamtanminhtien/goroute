package chatcompletion

import "fmt"

type UpstreamError struct {
	StatusCode int
	Message    string
}

func (e UpstreamError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("upstream returned status %d", e.StatusCode)
	}

	return fmt.Sprintf("upstream returned status %d: %s", e.StatusCode, e.Message)
}
