package loghandler

import (
	"io"
	"log/slog"
	"os"

	"github.com/qtraffics/qtfra/ex"
	"github.com/qtraffics/qtfra/log"
)

type BuildOption struct {
	Disabled bool      `json:"disabled"`
	Output   string    `json:"output"`
	Level    log.Level `json:"level"`
	Time     bool      `json:"time"`
	Debug    bool      `json:"debug"`

	OutputWriter io.Writer `json:"-"`
}

// New
// Deprecated: you should build handler by yourself
func New(opt BuildOption) (log.Handler, error) {
	var (
		h   = slog.DiscardHandler
		err error
	)
	if !opt.Disabled {
		var (
			file           = opt.OutputWriter
			sourceLevel    = log.LevelDisable
			timeFormatter  = log.RFC3339TimeFormatter
			levelFormatter = log.EqualLengthLevelFormatter
		)
		if opt.Debug {
			sourceLevel = log.LevelError
		}

		if file == nil {
			switch opt.Output {
			case "", "stdout":
				file = os.Stdout
				levelFormatter = log.ColorLevelFormatter
			case "stderr":
				file = os.Stderr
				levelFormatter = log.ColorLevelFormatter
			default:
				file, err = os.OpenFile(opt.Output, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
				if err != nil {
					return nil, ex.Cause(err, "openfile")
				}
			}
		}

		h = NewConsoleHandler(file, ConsoleHandlerOption{
			Level:      opt.Level,
			EnableTime: opt.Time,

			SourceLevel:    sourceLevel,
			TimeFormatter:  timeFormatter,
			LevelFormatter: levelFormatter,
		})
	}
	return h, nil
}
