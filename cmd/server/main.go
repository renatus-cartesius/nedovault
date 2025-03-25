package main

import (
	"github.com/renatus-cartesius/metricserv/pkg/logger"
	"github.com/renatus-cartesius/nedovault/api"
	"github.com/renatus-cartesius/nedovault/pkg/server"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"log"
	"net"
)

func main() {

	address := ":1337"

	if err := logger.Initialize("INFO"); err != nil {
		log.Fatalln(err)
	}

	lis, err := net.Listen("tcp", address)
	if err != nil {
		logger.Log.Fatal(
			"error creating listen struct for grpc server",
			zap.Error(err),
		)
	}
	var opts []grpc.ServerOption

	logger.Log.Info(
		"starting grpc server",
		zap.String("address", address),
	)
	grpcServer := grpc.NewServer(opts...)
	api.RegisterNedoVaultServer(grpcServer, server.NewServer())
	grpcServer.Serve(lis)
}
