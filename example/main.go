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

	fmt.Printf("base: %#v\n\n", ctx)

	ctx = ctxlog.With(ctx, "test", "value")
	fmt.Printf("ctxlog 1: %#v\n\n", ctx)

	ctx = ctxlog.With(ctx, "test", "list")
	fmt.Printf("ctxlog 2: %#v\n\n", ctx)

	clone := ctxlog.Clone(ctx)
	fmt.Printf("clone: %#v\n\n", clone)
	cancel()

	{
		clone := ctxlog.With(clone, "test", "three")
		fmt.Printf("nested: %#v\n\n", clone)
	}

	fmt.Printf("outside: %#v\n\n", clone)
}
