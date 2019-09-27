package ctxlog

import (
	"context"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/google/uuid"
)

var (
	// The logging context will always include a random UUID which is tagged
	// to uniquely identify this particular version/invocation of this program.
	// Allows us to see when restarts happen/induce changes in behaviour.
	thisUUID string

	debug bool

	infoC  *color.Color = color.New(color.FgCyan, color.Bold)
	debugC *color.Color = color.New(color.FgMagenta, color.Bold)
	errC   *color.Color = color.New(color.FgRed, color.Bold)
)

func init() {
	id, err := uuid.NewRandom()
	if err != nil {
		// Think about how to handle this error in case others use it?
		fmt.Printf("[ERROR] Could not initialize a UUID for ctxlog!\n")
		os.Exit(1)
	}

	thisUUID = id.String()
}

// loggingContext allows structured logging information (in the form of "tags")
// to be carried across API boundaries in an application.
type loggingContext struct {
	context.Context

	tags map[string]interface{}
}

// With adds a tag to the context, which is carried into subsequent logging calls.
func With(ctx context.Context, k string, v interface{}) context.Context {
	var lc loggingContext
	switch ctx.(type) {
	case loggingContext:
		lc = ctx.(loggingContext)
	default:
		lc = loggingContext{Context: ctx, tags: map[string]interface{}{}}
	}

	lc.tags[k] = v

	return lc
}

// WithValue is a hack to support adding WithValue to contexts without losing
// logging information.
func WithValue(parent context.Context, k string, v interface{}) context.Context {
	switch parent.(type) {
	case loggingContext:
		lc := parent.(loggingContext)
		lc.Context = context.WithValue(lc.Context, k, v)
		return lc
	default:
		ctx := context.WithValue(parent, k, v)
		return loggingContext{Context: ctx, tags: map[string]interface{}{}}
	}
}

// logf prints a log to the console with colorized tags.
func logf(ctx context.Context, c *color.Color, levelname string, msg string, args ...interface{}) {
	// TODO(silversupreme): Implement some logging to like JSON here when not attached to a TTY.
	msg = fmt.Sprintf(msg, args...)
	s := fmt.Sprintf("[%s] %-40s %s=%s", c.Sprintf("%-6s", levelname), msg, c.Sprint("uuid"), thisUUID)

	switch ctx.(type) {
	case loggingContext:
		lc := ctx.(loggingContext)
		for k, v := range lc.tags {
			s = fmt.Sprintf("%s %s=%v", s, c.Sprint(k), v)
		}
	default:
	}

	fmt.Println(s)
}

// EnableDebug will print debug output.
func EnableDebug() {
	debug = true
}

// DisableDebug will turn off debug output.
func DisableDebug() {
	debug = false
}

// Infof prints an informational string to the console.
func Infof(ctx context.Context, msg string, args ...interface{}) {
	logf(ctx, infoC, "INFO", msg, args...)
}

// Debugf prints debug info if that has been enabled in the program.
func Debugf(ctx context.Context, msg string, args ...interface{}) {
	if !debug {
		return
	}

	logf(ctx, debugC, "DEBUG", msg, args...)
}

// Errorf prints an error log to the console.
func Errorf(ctx context.Context, msg string, args ...interface{}) {
	logf(ctx, errC, "ERROR", msg, args...)
}
