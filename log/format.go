package log

import (
	"fmt"
	"time"

	"github.com/logrusorgru/aurora"
)

func CommonLevelFormatter(l Level) string {
	return l.String()
}

func EqualLengthLevelFormatter(l Level) string {
	str := func(base string, val Level) string {
		if val == 0 {
			return base
		}
		return fmt.Sprintf("%s+%d", base, val)
	}

	switch {
	case l < LevelInfo:
		return str("DEBUG", l-LevelDebug)
	case l < LevelWarn:
		return str("INFO ", l-LevelInfo)
	case l < LevelError:
		return str("WARN ", l-LevelWarn)
	default:
		return str("ERROR", l-LevelError)
	}
}

func ColorLevelFormatter(l Level) string {
	str := func(base string, val Level) string {
		if val == 0 {
			return base
		}
		return fmt.Sprintf("%s+%d", base, aurora.Red(val))
	}

	switch {
	case l < LevelInfo:
		return str(aurora.White("DEBUG").String(), l-LevelDebug)
	case l < LevelWarn:
		return str(aurora.Cyan("INFO ").String(), l-LevelInfo)
	case l < LevelError:
		return str(aurora.Yellow("WARN ").String(), l-LevelWarn)
	default:
		return str(aurora.Red("ERROR").String(), l-LevelError)
	}
}

func RFC3339TimeFormatter(t time.Time) string {
	return t.Format(time.RFC3339)
}
