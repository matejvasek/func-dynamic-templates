package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/matejvasek/func-dynamic-tempates/lib"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
		<-sigs
		os.Exit(1)
	}()

	rootCmd, err := lib.NewCommandFromRepository(quarkusRepository{})

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	err = rootCmd.ExecuteContext(ctx)
	if err != nil {
		os.Exit(1)
	}
}
