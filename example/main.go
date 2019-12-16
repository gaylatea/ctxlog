package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/silversupreme/ctxlog"
)

func main() {
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	ctxlog.Infof(ctx, "hello world")

	ctx = ctxlog.With(ctx, "test", "value")
	ctxlog.Infof(ctx, "testing with single value")
	fmt.Printf("%v\n", ctx.(ctxlog.LoggingContext).ToJSON())

	ctx = ctxlog.With(ctx, "test", "list")
	ctxlog.Infof(ctx, "testing with multiple values")
	fmt.Printf("%v\n", ctx.(ctxlog.LoggingContext).ToJSON())

	clone := ctxlog.Clone(ctx)
	cancel()

	fmt.Printf("Original context: %v\nCloned context: %v\n", ctx.Err(), clone.Err())
}
