package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"

	environ "github.com/ydb-platform/ydb-go-sdk-auth-environ"
)

var (
	dsn  string
	port int
)

func init() {
	required := []string{"ydb"}
	flagSet := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flagSet.Usage = func() {
		out := flagSet.Output()
		_, _ = fmt.Fprintf(out, "Usage:\n%s [options]\n", os.Args[0])
		_, _ = fmt.Fprintf(out, "\nOptions:\n")
		flagSet.PrintDefaults()
	}
	flagSet.StringVar(&dsn,
		"ydb", "",
		"YDB connection string",
	)
	flagSet.IntVar(&port,
		"port", 8080,
		"http port for web-server",
	)

	if err := flagSet.Parse(os.Args[1:]); err != nil {
		flagSet.Usage()
		os.Exit(1)
	}
	flagSet.Visit(func(f *flag.Flag) {
		for i, arg := range required {
			if arg == f.Name {
				required = append(required[:i], required[i+1:]...)
			}
		}
	})
	if len(required) > 0 {
		fmt.Printf("\nSome required options not defined: %v\n\n", required)
		flagSet.Usage()
		os.Exit(1)
	}
}

func main() {
	var (
		ctx    context.Context
		cancel context.CancelFunc
		done   = make(chan struct{})
	)

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	s, err := getService(ctx, dsn, environ.WithEnvironCredentials(ctx))
	if err != nil {
		fmt.Printf("Create service failed: %v\n", err)
		return
	}
	defer s.Close(ctx)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: s.router,
	}

	defer func() {
		_ = server.Shutdown(ctx)
	}()

	go func() {
		_ = server.ListenAndServe()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return
	case <-done:
		return
	}
}
