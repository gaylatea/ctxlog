package ctxlog

import (
	"context"
	"fmt"

	"github.com/fatih/color"
)

// Sink implementers accept event data and store it for later analysis.
type Sink interface {
	Log(ctx context.Context, c *color.Color, levelname string, msg string, args ...interface{}) error
}

var (
	// Keep the ConsoleSink around as a backup in case other sinks fail.
	console = &ConsoleSink{}

	sinks = map[string]Sink{
		"console": console,
	}
)

// UseSink adds a sink which will receive all logs output by the application.
func UseSink(name string, s Sink) {
	sinks[name] = s
}

// ConsoleSink dumps out events to the console with colorized tags.
type ConsoleSink struct{}

// Log prints to the console with colorized tags.
func (cs *ConsoleSink) Log(ctx context.Context, c *color.Color, levelname string, msg string, args ...interface{}) error {
	// TODO(silversupreme): Implement some logging to like JSON here when not attached to a TTY.
	msg = fmt.Sprintf(msg, args...)
	s := fmt.Sprintf("[%s] %-40s", c.Sprintf("%-6s", levelname), msg)

	switch ctx.(type) {
	case LoggingContext:
		lc := ctx.(LoggingContext)
		// Ensure that tags are printed in the order that they were added,
		// which creates a nice nesting effect for logs.
		for _, k := range lc.order {
			val := lc.tags[k]

			// Special-case for single-item lists, to just print that single
			// item. Helps preserve the normal expected formatting.
			if len(val) == 1 {
				s = fmt.Sprintf("%s %s=%v", s, c.Sprint(k), lc.tags[k][0])
			} else {
				s = fmt.Sprintf("%s %s=%v", s, c.Sprint(k), lc.tags[k])
			}
		}
	default:
	}

	// Always include the global UUID in logs, at the end.
	s = fmt.Sprintf("%s %s=%s", s, c.Sprint("uuid"), globalUUID.String())
	fmt.Println(s)

	return nil
}
