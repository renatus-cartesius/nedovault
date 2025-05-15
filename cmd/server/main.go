package main

import (
	"github.com/dgraph-io/badger/v4"
	"github.com/golang-jwt/jwt/v4"
	"github.com/renatus-cartesius/metricserv/pkg/logger"
	"github.com/renatus-cartesius/nedovault/api"
	"github.com/renatus-cartesius/nedovault/internal/auth"
	"github.com/renatus-cartesius/nedovault/pkg/server"
	"github.com/renatus-cartesius/nedovault/pkg/storage"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"log"
	"net"
	"time"
)

func main() {

	address := ":1337"

	badgerOpts := badger.DefaultOptions("./.nedovault")

	badgerOpts.EncryptionKey = []byte("verysstrongkeeeeyfromsomeconfigg")
	badgerOpts.IndexCacheSize = 100 << 20

	db, err := badger.Open(badgerOpts)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err = logger.Initialize("INFO"); err != nil {
		log.Fatalln(err)
	}

	badgerStorage := storage.NewBadgerStorage(db)
	localAuth := auth.NewLocalAuth(
		[]byte("d6b32087c4b1f7c8b88c945234d54cfa5aa73d4b14e5e7a778448d515db00028b20db"), // TODO: store key in the storage
		time.Hour*24*30,
		badgerStorage,
		jwt.SigningMethodHS256,
	)

	lis, err := net.Listen("tcp", address)
	if err != nil {
		logger.Log.Fatal(
			"error creating listen struct for grpc server",
			zap.Error(err),
		)
	}
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(server.NewAuthUnaryInterceptor(localAuth)),
		grpc.StreamInterceptor(server.NewAuthStreamInterceptor(localAuth)),
	}

	logger.Log.Info(
		"starting grpc server",
		zap.String("address", address),
	)
	grpcServer := grpc.NewServer(opts...)

	api.RegisterNedoVaultServer(
		grpcServer,
		server.NewServer(
			badgerStorage,
			localAuth,
		),
	)
	grpcServer.Serve(lis)
}
