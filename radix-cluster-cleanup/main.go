package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/equinor/radix-cluster-cleanup/cmd"
)

func init() {
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGTERM)
	defer cancel()
	cmd.Execute(ctx)
}
