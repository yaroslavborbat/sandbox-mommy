package main

import (
	"log/slog"
	"os"

	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/yaroslavborbat/sandbox-mommy/cmd/controller/app"
	"github.com/yaroslavborbat/sandbox-mommy/pkg/logging"
)

func main() {
	ctx := signals.SetupSignalHandler()
	if err := app.NewSandboxControllerCommand().ExecuteContext(ctx); err != nil {
		slog.Error("Failed to run sandbox-controller", logging.SlogErr(err))
		os.Exit(1)
	}
}
