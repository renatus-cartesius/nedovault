package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/renatus-cartesius/metricserv/pkg/logger"
	"github.com/renatus-cartesius/nedovault/api"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
)

func main() {

	serverAddress := "127.0.0.1:1337"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := logger.Initialize("INFO"); err != nil {
		log.Fatalln(err)
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.NewClient(serverAddress, opts...)
	if err != nil {
		logger.Log.Fatal(
			"error creating grpc client",
			zap.String("server_address", serverAddress),
			zap.Error(err),
		)
	}
	defer conn.Close()

	client := api.NewNedoVaultClient(conn)

	logger.Log.Info(
		"adding log pass",
	)
	_, err = client.AddSecret(ctx, &api.AddSecretRequest{
		Name: fmt.Sprintf("%s-%s", "logpass", uuid.NewString()),
		Secret: &api.Secret{
			Secret: &api.Secret_LogPass{
				LogPass: &api.LogPass{
					Login:    "admin",
					Password: "root",
				},
			},
		},
	},
	)

	logger.Log.Info(
		"adding simple text",
	)
	_, err = client.AddSecret(ctx, &api.AddSecretRequest{
		Name: fmt.Sprintf("%s-%s", "text", uuid.NewString()),
		Secret: &api.Secret{
			Secret: &api.Secret_Text{
				Text: &api.Text{
					Data: "Hello World!",
				},
			},
		},
	},
	)

	if err != nil {
		logger.Log.Error(
			"error on adding secret",
			zap.Error(err),
		)
	}

	res, err := client.ListSecrets(ctx, &emptypb.Empty{})
	if err != nil {
		logger.Log.Error(
			"error on listing secrets",
			zap.Error(err),
		)
	}

	fmt.Println(res)
}
