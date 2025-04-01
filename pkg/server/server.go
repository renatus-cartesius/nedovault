package server

import (
	"context"
	"errors"
	"github.com/dgraph-io/badger/v4"
	"github.com/renatus-cartesius/metricserv/pkg/logger"
	"github.com/renatus-cartesius/nedovault/api"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"time"
)

type Storage interface {
	AddSecret(ctx context.Context, username []byte, in *api.AddSecretRequest) error
	GetSecret(ctx context.Context, username, key []byte) (*api.Secret, *api.SecretMeta, error)
	ListSecretsMeta(ctx context.Context, username []byte) ([]*api.SecretMeta, error)
}

type Server struct {
	api.UnimplementedNedoVaultServer

	storage Storage
}

func NewServer(storage Storage) *Server {
	return &Server{
		storage: storage,
	}
}

func (s *Server) ListSecretsMetaStream(request *api.ListSecretsMetaRequest, g grpc.ServerStreamingServer[api.ListSecretsMetaResponse]) error {
	t := time.NewTicker(time.Second)

	for {

		select {
		case <-t.C:
			logger.Log.Info("sending metadata to client")
			meta, err := s.storage.ListSecretsMeta(context.Background(), request.Username)
			if err != nil {
				return status.Errorf(codes.Internal, "error listing secrets metadata")
			}

			response := &api.ListSecretsMetaResponse{
				SecretsMeta: meta,
			}

			g.Send(response)
		}

	}
}

func (s *Server) GetSecret(ctx context.Context, request *api.GetSecretRequest) (*api.GetSecretResponse, error) {
	username := []byte("admin")

	secret, secretMeta, err := s.storage.GetSecret(ctx, username, request.GetKey())
	if err != nil {

		if errors.Is(err, badger.ErrKeyNotFound) {
			logger.Log.Debug(
				"requested unknown key",
				zap.String("key", string(request.Key)),
			)

			return nil, status.Errorf(codes.Internal, "no such secret")
		}

		logger.Log.Error(
			"error getting secret",
			zap.Error(err),
		)
		return nil, status.Errorf(codes.Internal, "error getting secret data")
	}

	response := &api.GetSecretResponse{
		Secret:     secret,
		SecretMeta: secretMeta,
	}

	return response, nil
}

func (s *Server) AddSecret(ctx context.Context, in *api.AddSecretRequest) (*emptypb.Empty, error) {

	logger.Log.Info(
		"adding secret",
	)

	if err := s.storage.AddSecret(ctx, []byte("admin"), in); err != nil {
		logger.Log.Error(
			"error when adding secret",
			zap.Error(err),
		)
		return &emptypb.Empty{}, status.Errorf(codes.Internal, "pair %s already exists", in.Key)
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) ListSecretsMeta(ctx context.Context, request *api.ListSecretsMetaRequest) (*api.ListSecretsMetaResponse, error) {

	logger.Log.Debug(
		"listing secrets metadata",
		zap.String("username", string(request.Username)),
	)

	meta, err := s.storage.ListSecretsMeta(ctx, request.Username)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error listing secrets metadata")
	}

	response := &api.ListSecretsMetaResponse{
		SecretsMeta: meta,
	}

	return response, nil
}
