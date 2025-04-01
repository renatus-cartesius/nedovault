package main

import (
	"context"
	"fmt"
	"github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"
	"github.com/renatus-cartesius/metricserv/pkg/logger"
	"github.com/renatus-cartesius/nedovault/api"
	"github.com/renatus-cartesius/nedovault/pkg/server"
	"github.com/renatus-cartesius/nedovault/pkg/storage"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"log"
	"net"
	"sync"
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

	badgerStorage := storage.NewBadgerStorage(db)

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		t := time.NewTicker(5 * time.Second)

		for {
			select {
			case <-t.C:
				logger.Log.Info("adding random secret")
				badgerStorage.AddSecret(context.Background(), []byte("admin"), &api.AddSecretRequest{
					Key:        []byte(fmt.Sprintf("random-%s", uuid.NewString())),
					Name:       []byte("name"),
					SecretType: api.SecretType_TYPE_TEXT,
					Secret: &api.Secret{
						Secret: &api.Secret_Text{
							Text: &api.Text{
								Data: "Some random text!",
							},
						},
					},
				})
			}
		}

	}()

	api.RegisterNedoVaultServer(grpcServer, server.NewServer(badgerStorage))
	grpcServer.Serve(lis)
	wg.Wait()
}
