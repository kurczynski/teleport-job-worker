package logging

import (
	"log/slog"
)

var (
	Log *slog.Logger
)

func Setup(handler slog.Handler) {
	Log = slog.New(handler)
}
