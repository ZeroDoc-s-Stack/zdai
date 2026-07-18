package main

import (
	"context"
	"os"

	"github.com/zerodoc-s-stack/zdai/internal/controllers"
	pb "github.com/zerodoc-s-stack/zdai/package/grpc"
	"github.com/zerodoc-s-stack/zdlib/base/grpc"
	"go.opentelemetry.io/otel"
)

var (
	service = "zdai"
	version = "latest"
)

// StartMicro initialises the go-micro v5 gRPC service via zdlib, registers it
// with Consul, and runs it in a background goroutine.
func StartMicro(ctx context.Context, h *controllers.Zdai) error {
	srv, err := grpc.LoadServer(ctx, service, version)
	if err != nil {
		return err
	}

	if err := pb.RegisterZdaiHandler(srv.Server(), h); err != nil {
		return err
	}

	go func() {
		log.Infof("starting go-micro on [address=%s]...", os.Getenv("NOMAD_HOST_ADDR_micro"))
		// ponytail: no OTLP_ENDPOINT configured for zdai yet, so the global
		// provider is a no-op; swap in metric.LoadTraces when tracing lands
		if err := grpc.RunServer(srv, log, otel.GetTracerProvider()); err != nil {
			log.Fatal(err)
		}
		log.Info("shutting down go-micro...")
	}()

	return nil
}
