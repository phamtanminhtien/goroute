package logging

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
)

const serviceName = "goroute"

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
