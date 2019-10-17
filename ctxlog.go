package ctxlog

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

var (
	debug   = flag.Bool("debug", false, "Enable debug logging.")
	noColor = flag.Bool("nocolor", false, "Disable colored output.")

	infoC  *color.Color = color.New(color.FgCyan, color.Bold)
	debugC *color.Color = color.New(color.FgMagenta, color.Bold)
	errC   *color.Color = color.New(color.FgRed, color.Bold)
	fatalC *color.Color = color.New(color.FgBlack, color.BgRed, color.Bold)
)

// Context returns a root context that is suitable for logging.
func Context() (context.Context, error) {
	// Disable colorized log output if we've been requested to do that.
	if *noColor {
		infoC.DisableColor()
		debugC.DisableColor()
		errC.DisableColor()
		fatalC.DisableColor()
	}

	// The logging context will always include a random UUID which is tagged
	// to uniquely identify this particular version/invocation of this program.
	// Allows us to see when restarts happen/induce changes in behaviour.
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, errors.Wrapf(err, "Could not generate a new context")
	}

	return With(context.Background(), "uuid", id.String()), nil
}

// loggingContext allows structured logging information (in the form of "tags")
// to be carried across API boundaries in an application.
type loggingContext struct {
	context.Context

	tags  map[string]interface{}
	order []string
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
	// TODO(silversupreme): Make this work better with calls that override
	// existing tags.
	lc.order = append(lc.order, k)

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
	s := fmt.Sprintf("[%s] %-40s", c.Sprintf("%-6s", levelname), msg)

	switch ctx.(type) {
	case loggingContext:
		lc := ctx.(loggingContext)
		// Ensure that tags are printed in the order that they were added,
		// which creates a nice nesting effect for logs.
		for _, k := range lc.order {
			s = fmt.Sprintf("%s %s=%v", s, c.Sprint(k), lc.tags[k])
		}
	default:
	}

	fmt.Println(s)
}

// Infof prints an informational string to the console.
func Infof(ctx context.Context, msg string, args ...interface{}) {
	logf(ctx, infoC, "INFO", msg, args...)
}

// Debugf prints debug info if that has been enabled in the program.
func Debugf(ctx context.Context, msg string, args ...interface{}) {
	if !*debug {
		return
	}

	logf(ctx, debugC, "DEBUG", msg, args...)
}

// Errorf prints an error log to the console.
func Errorf(ctx context.Context, msg string, args ...interface{}) {
	logf(ctx, errC, "ERROR", msg, args...)
}

// Fatalf prints an error and immediately stops execution.
func Fatalf(ctx context.Context, msg string, args ...interface{}) {
	logf(ctx, fatalC, "FATAL", msg, args...)
	os.Exit(1)
}
