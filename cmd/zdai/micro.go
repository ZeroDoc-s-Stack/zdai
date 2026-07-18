package main

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/hashicorp/consul/api"
	micrologrus "github.com/micro/plugins/v5/logger/logrus"
	"github.com/micro/plugins/v5/registry/consul"
	"github.com/zerodoc-s-stack/zdai/internal/controllers"
	pb "github.com/zerodoc-s-stack/zdai/package/grpc"
	"go-micro.dev/v5"
	"go-micro.dev/v5/logger"
	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/server"
	"go-micro.dev/v5/transport"
)

var (
	service = "zdai"
	version = "latest"
)

// StartMicro initialises the go-micro v5 gRPC service, registers it with
// Consul, and runs it in a background goroutine. Mirrors the zdauth pattern.
func StartMicro(ctx context.Context, h *controllers.Zdai) (registry.Registry, error) {
	reg := consul.NewRegistry(consul.Config(
		&api.Config{
			Address: os.Getenv("CONSUL_ADDRESS"),
		},
	))

	listener, err := setupListener()
	if err != nil {
		return nil, err
	}

	srv := micro.NewService(
		micro.Name(service),
		micro.Version(version),
		micro.Context(ctx),
		micro.Registry(reg),
		micro.Logger(logger.NewLogger(micrologrus.WithLogger(log))),
		listener,
	)
	srv.Init(
		micro.AddListenOption(
			server.Advertise(os.Getenv("NOMAD_HOST_ADDR_micro")),
		),
	)

	if err := pb.RegisterZdaiHandler(srv.Server(), h); err != nil {
		return nil, err
	}

	go func() {
		log.Infof("starting go-micro on [address=%s]...", os.Getenv("NOMAD_HOST_ADDR_micro"))
		if err := srv.Run(); err != nil {
			log.Fatal(err)
		}
		log.Info("shutting down go-micro...")
	}()

	return reg, nil
}

func setupListener() (micro.Option, error) {
	listener, err := createListener()
	if err != nil {
		return nil, err
	}
	return micro.AddListenOption(
		server.ListenOption(
			transport.NetListener(listener),
		),
	), nil
}

func createListener() (net.Listener, error) {
	address := ":" + os.Getenv("MICRO_PORT")
	listen, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to create listener on [address=%s]: %w", address, err)
	}
	log.Infof("listening on [address=%s]", address)
	return listen, nil
}
