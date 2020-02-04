package ctxlog

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/google/uuid"
)

// Tag represents a piece of structured information that should be
// added to a log line.
type Tag struct {
	K string
	V interface{}

	// Should this tag override anything that is already there?
	Override bool
}

var (
	debug             = flag.Bool("debug", false, "Enable debug logging.")
	noColorDEPRECATED = flag.Bool("nocolor", false, "Disable colored output.")

	infoC  *color.Color = color.New(color.FgCyan, color.Bold)
	debugC *color.Color = color.New(color.FgMagenta, color.Bold)
	errC   *color.Color = color.New(color.FgRed, color.Bold)
	fatalC *color.Color = color.New(color.FgBlack, color.BgRed, color.Bold)

	// The logging context will always include a random UUID which is tagged
	// to uniquely identify this particular version/invocation of this program.
	// Allows us to see when restarts happen/induce changes in behaviour.
	globalUUID uuid.UUID
)

func init() {
	// Disable colorized log output if we've been requested to do that.
	if noColor := os.Getenv("DISABLE_COLOR_OUTPUT"); noColor == "1" {
		infoC.DisableColor()
		debugC.DisableColor()
		errC.DisableColor()
		fatalC.DisableColor()
	}

	id, err := uuid.NewRandom()
	if err != nil {
		globalUUID = uuid.Nil
		console.Log(context.Background(), errC, "ERROR",
			"Could not create a unique ID for this application: %v", err)
	} else {
		globalUUID = id
	}
}

// LoggingContext allows structured logging information (in the form of "tags")
// to be carried across API boundaries in an application.
type LoggingContext struct {
	context.Context

	tags  map[string][]interface{}
	order []string
}

// ToJSON returns a representation of the context's current data suitable for
// logging to an external database.
func (c LoggingContext) ToJSON() map[string]interface{} {
	ret := map[string]interface{}{
		"instance_id": globalUUID.String(),
	}

	for k, v := range c.tags {
		// Special-case single-item lists, to just use the value. Helps with
		// querying in the future.
		if len(v) == 1 {
			ret[k] = v[0]
		} else {
			ret[k] = v
		}
	}

	return ret
}

// With adds a tag to the context, which is carried into subsequent logging calls.
func With(ctx context.Context, k string, v interface{}) context.Context {
	return WithAll(ctx, Tag{K: k, V: v})
}

// WithAll adds multiple tags at once to a context, which avoids a ton of
// GC churn when you know you have multiple things to add to a logging
// statement.
func WithAll(ctx context.Context, tags ...Tag) context.Context {
	ret := LoggingContext{
		tags:  map[string][]interface{}{},
		order: []string{},
	}

	switch ctx.(type) {
	case LoggingContext:
		lc := ctx.(LoggingContext)
		ret.tags = make(map[string][]interface{}, (len(lc.tags) + 1))
		ret.order = make([]string, len(lc.order))
		ret.Context = lc.Context

		// This sucks, in a lot of ways, but it's necessary to allow us to properly
		// log with ctxlog without downstream functions overwriting or adding to
		// the tag set for a given context.
		for i, x := range lc.order {
			ret.order[i] = x
		}

		for i, x := range lc.tags {
			ret.tags[i] = x
		}
	default:
		ret.Context = ctx
		ret.tags = make(map[string][]interface{}, 1)
	}

	// Add all the tags.
	for _, x := range tags {
		// Don't print multiple times.
		if _, exists := ret.tags[x.K]; !exists {
			ret.order = append(ret.order, x.K)
		}

		if x.Override {
			ret.tags[x.K] = []interface{}{x.V}
		} else {
			ret.tags[x.K] = append(ret.tags[x.K], x.V)
		}
	}

	return ret
}

// WithValue is a hack to support adding WithValue to contexts without losing
// logging information.
func WithValue(parent context.Context, k string, v interface{}) context.Context {
	switch parent.(type) {
	case LoggingContext:
		lc := parent.(LoggingContext)
		lc.Context = context.WithValue(lc.Context, k, v)
		return lc
	default:
		ctx := context.WithValue(parent, k, v)
		return LoggingContext{Context: ctx, tags: map[string][]interface{}{}}
	}
}

// Clone creates a copy of `source` with all of the tags intact.
// TODO: Make a version of this that takes in a context and copies over.
func Clone(source context.Context) context.Context {
	switch source.(type) {
	case LoggingContext:
		lc := source.(LoggingContext)
		ret := LoggingContext{
			Context: context.Background(),
			tags:    make(map[string][]interface{}, len(lc.tags)),
			order:   make([]string, len(lc.order)),
		}

		// This sucks, in a lot of ways, but it's necessary to allow us to properly
		// log with ctxlog without downstream functions overwriting or adding to
		// the tag set for a given context.
		for i, x := range lc.order {
			ret.order[i] = x
		}

		for i, x := range lc.tags {
			ret.tags[i] = x
		}

		return ret
	default:
		return LoggingContext{
			Context: context.Background(),
			tags:    map[string][]interface{}{},
		}
	}
}

func logf(ctx context.Context, c *color.Color, levelname string, msg string, args ...interface{}) {
	for name, sink := range sinks {
		if err := sink.Log(ctx, c, levelname, msg, args...); err != nil {
			console.Log(ctx, errC, "ERROR", "Could not process log sink '%s': %v", name, err)
		}
	}
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

// Trace allows nested logging of operations.
// TODO: make a version of this that can log across multiple pageviews/RPCs.
func Trace(ctx context.Context, name string, fn func(ctx context.Context) error) error {
	switch ctx.(type) {
	case LoggingContext:
		c := ctx.(LoggingContext)

		if n, ok := c.tags["span_id"]; ok {
			ctx = WithAll(ctx, Tag{
				K:        "parent_id",
				V:        n[0],
				Override: true,
			})
		}
	default:
	}

	spanID, err := uuid.NewRandom()
	if err != nil {
		Errorf(ctx, "could not generate span ID: %v", err)
		return err
	}

	start := time.Now()
	ctx = WithAll(ctx,
		Tag{
			K:        "span_id",
			V:        spanID.String(),
			Override: true,
		},
		Tag{
			K:        "name",
			V:        name,
			Override: true,
		},
		Tag{
			K:        "start_time",
			V:        start.Unix(),
			Override: true,
		},
	)
	err = fn(ctx)

	end := time.Now()
	ctx = WithAll(ctx,
		Tag{
			K:        "dur_ms",
			V:        end.Sub(start).Milliseconds(),
			Override: true,
		},
		Tag{
			K:        "end_time",
			V:        end.Unix(),
			Override: true,
		},
	)

	Infof(ctx, "span")
	return err
}
