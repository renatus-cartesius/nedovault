package main

import (
	"github.com/dgraph-io/badger/v4"
	"github.com/renatus-cartesius/metricserv/pkg/logger"
	"github.com/renatus-cartesius/nedovault/api"
	"github.com/renatus-cartesius/nedovault/pkg/server"
	"github.com/renatus-cartesius/nedovault/pkg/storage"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"log"
	"net"
)

func main() {

	address := ":1337"

	badgerOpts := badger.DefaultOptions("./.nedovault")

	badgerOpts.EncryptionKey = []byte("verysstrongkeeeeyfromsomeconfigg")

	db, err := badger.Open(badgerOpts)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

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
	api.RegisterNedoVaultServer(grpcServer, server.NewServer(storage.NewBadgerStorage(db)))
	grpcServer.Serve(lis)
}
