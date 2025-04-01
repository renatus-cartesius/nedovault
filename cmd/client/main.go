package main

import (
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	"github.com/renatus-cartesius/metricserv/pkg/logger"
	"github.com/renatus-cartesius/nedovault/api"
	"github.com/renatus-cartesius/nedovault/internal/tui"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log"
	"os"
	"sync"
	"time"
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

	//logger.Log.Info(
	//	"adding log pass",
	//)
	//_, err = client.AddSecret(ctx, &api.AddSecretRequest{
	//	Key: []byte(fmt.Sprintf("%s-%s", "logpass", uuid.NewString())),
	//	Secret: &api.Secret{
	//		Secret: &api.Secret_LogPass{
	//			LogPass: &api.LogPass{
	//				Login:    "admin",
	//				Password: "root",
	//			},
	//		},
	//	},
	//},
	//)
	//
	//logger.Log.Info(
	//	"adding simple text",
	//)
	//_, err = client.AddSecret(ctx, &api.AddSecretRequest{
	//	Key: []byte(fmt.Sprintf("%s-%s", "text", uuid.NewString())),
	//	Secret: &api.Secret{
	//		Secret: &api.Secret_Text{
	//			Text: &api.Text{
	//				Data: "Hello World!",
	//			},
	//		},
	//	},
	//},
	//)

	if err != nil {
		logger.Log.Error(
			"error on adding secret",
			zap.Error(err),
		)
	}

	listSecretsMetaResponse, err := client.ListSecretsMeta(ctx, &api.ListSecretsMetaRequest{
		Username: []byte("admin"),
	})
	if err != nil {
		logger.Log.Error(
			"error on listing secrets",
			zap.Error(err),
		)
	}

	choices := make([]string, 0)
	for _, s := range listSecretsMetaResponse.SecretsMeta {
		fmt.Println(s.Timestamp, s.Type, string(s.Key))
		choices = append(choices, string(s.Key))
	}

	logger.Log.Info(
		"getting specific secret",
	)

	key := []byte("text-769bfb1c-bd4d-4d24-ac2b-db7bd2f0f16c")

	getSecretResponse, err := client.GetSecret(ctx, &api.GetSecretRequest{
		Key: key,
	})

	if err != nil {
		logger.Log.Error(
			"error getting secret",
			zap.Error(err),
		)
		os.Exit(0)
	}

	fmt.Println("Secret meta:", getSecretResponse.SecretMeta)
	fmt.Println("Secret data:", getSecretResponse.Secret)

	var items []list.Item

	for _, sm := range listSecretsMetaResponse.SecretsMeta {
		items = append(items, &tui.SecretItem{
			SecretMeta: sm,
		})
	}

	wg := &sync.WaitGroup{}
	ui := tui.NewUI(items)

	wg.Add(1)
	go func() {
		ui.Run()
		wg.Done()
	}()

	metadataStream, err := client.ListSecretsMetaStream(ctx, &api.ListSecretsMetaRequest{
		Username: []byte("admin"),
	})
	wg.Add(1)
	go func() {
		for {
			resp, err := metadataStream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				log.Fatalln("error when waiting messages from server")
			}

			items = nil
			for _, sm := range resp.SecretsMeta {
				items = append(items, &tui.SecretItem{
					SecretMeta: sm,
				})
			}
		}
	}()

	time.Sleep(time.Second * 3)

	items[0].(*tui.SecretItem).SecretMeta = &api.SecretMeta{
		Key:  []byte("sdfsdf"),
		Name: []byte("sdfsdf"),
		Type: api.SecretType_TYPE_TEXT,
	}

	wg.Wait()

}
