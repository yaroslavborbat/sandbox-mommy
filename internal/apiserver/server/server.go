package server

import (
	"context"

	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/tools/cache"
)

func NewServer(
	sandboxes cache.Controller,
	apiserver *genericapiserver.GenericAPIServer,
) *Server {
	return &Server{
		sandboxes:        sandboxes,
		GenericAPIServer: apiserver,
	}
}

type Server struct {
	*genericapiserver.GenericAPIServer
	sandboxes cache.Controller
}

func (s *Server) Run(ctx context.Context) error {
	go s.sandboxes.Run(ctx.Done())

	// Ensure cache is up-to-date
	ok := cache.WaitForCacheSync(ctx.Done(), s.sandboxes.HasSynced)
	if !ok {
		return nil
	}
	return s.GenericAPIServer.PrepareRun().RunWithContext(ctx)
}
