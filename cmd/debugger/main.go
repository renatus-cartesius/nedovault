package main

import (
	"context"
	"fmt"
	"github.com/renatus-cartesius/metricserv/pkg/logger"
	"github.com/renatus-cartesius/nedovault/api"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
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

	res, err := client.Authorize(ctx, &api.AuthRequest{
		Username: []byte("admin"),
		Password: []byte("passs"),
	})

	if err != nil {
		log.Fatalln(err)
	}

	//token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NDcwNzM2NTcsImlhdCI6MTc0NDQ4MTY1NywidXNlcm5hbWUiOiJkdW1teSJ9.FUOu-DtS1_azr8YJgpEdygKuSpOQKiPbZAilra3p8xI"

	ctx = metadata.AppendToOutgoingContext(ctx, "token", res.Token)

	listSecretsMetaResponse, err := client.ListSecretsMeta(ctx, &emptypb.Empty{})

	fmt.Println(listSecretsMetaResponse)

}
