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
		Key: []byte(fmt.Sprintf("%s-%s", "logpass", uuid.NewString())),
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
		Key: []byte(fmt.Sprintf("%s-%s", "text", uuid.NewString())),
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

	secretsMeta, err := client.ListSecretsMeta(ctx, &api.ListSecretsMetaRequest{
		Username: []byte("admin"),
	})
	if err != nil {
		logger.Log.Error(
			"error on listing secrets",
			zap.Error(err),
		)
	}

	for _, s := range secretsMeta.SecretsMeta {
		fmt.Println(s.Timestamp, s.Type, string(s.Key))
	}

	logger.Log.Info(
		"getting specific secret",
	)

	key := []byte("logpass-af3b1dcf-53c4-405f-9a5b-50953daf036d")

	getSecretResponse, err := client.GetSecret(ctx, &api.GetSecretRequest{
		Key: key,
	})

	fmt.Println("Secret meta:", getSecretResponse.SecretMeta)
	fmt.Println("Secret data:", getSecretResponse.Secret)
}
