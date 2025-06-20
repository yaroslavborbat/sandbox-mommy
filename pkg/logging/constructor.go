package logging

import (
	"log/slog"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func NewConstructor(log *slog.Logger) func(req *reconcile.Request) logr.Logger {
	return func(req *reconcile.Request) logr.Logger {
		log := log
		if req != nil {
			log = log.With(SlogNamespace(req.Namespace), SlogName(req.Name))
		}

		return logr.FromSlogHandler(log.Handler())
	}
}
