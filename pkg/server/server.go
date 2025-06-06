package server

import (
	"bytes"
	"context"
	"errors"
	"github.com/dgraph-io/badger/v4"
	"github.com/renatus-cartesius/metricserv/pkg/logger"
	"github.com/renatus-cartesius/nedovault/api"
	"github.com/renatus-cartesius/nedovault/internal/auth"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"sync"
	"time"
)

var (
	ErrMetadataParseFail = status.Errorf(codes.Internal, "failed to parse request metadata")
)

type Storage interface {
	AddSecret(ctx context.Context, username []byte, in *api.AddSecretRequest) error
	DeleteSecret(ctx context.Context, username []byte, in *api.DeleteSecretRequest) error
	GetSecret(ctx context.Context, username, key []byte) (*api.Secret, *api.SecretMeta, error)
	ListSecretsMeta(ctx context.Context, username []byte) ([]*api.SecretMeta, error)
}

type Auth interface {
	Authorize(ctx context.Context, in *api.AuthRequest) (string, error)
	ParseToken(ctx context.Context, token []byte) (*auth.Claims, error)
}

type session struct {
	username []byte
}

type Server struct {
	api.UnimplementedNedoVaultServer

	smap    *sync.Map
	storage Storage
	auth    Auth
}

func (s *Server) DeleteSecret(ctx context.Context, in *api.DeleteSecretRequest) (*emptypb.Empty, error) {
	username := ctx.Value(auth.Username("username")).([]byte)

	logger.Log.Info(
		"deleting secret",
		zap.String("username", string(username)),
	)

	if err := s.storage.DeleteSecret(ctx, username, in); err != nil {
		logger.Log.Error(
			"error when deleting secret",
			zap.Error(err),
		)
		return &emptypb.Empty{}, status.Errorf(codes.Internal, "pair %s already exists", in.Key)
	}

	s.notifyMetadataStreams(username)

	return &emptypb.Empty{}, nil
}

func NewServer(storage Storage, auth Auth) *Server {
	return &Server{
		storage: storage,
		smap:    &sync.Map{},
		auth:    auth,
	}
}

func (s *Server) Authorize(ctx context.Context, in *api.AuthRequest) (*api.AuthResponse, error) {
	token, err := s.auth.Authorize(ctx, in)
	if err != nil {

		if errors.Is(err, auth.ErrInvalidCredentials) {
			return nil, status.Errorf(codes.Unauthenticated, "error authorizing")
		}

		logger.Log.Error(
			"error when autorizing",
			zap.Error(err),
		)

		return nil, status.Errorf(codes.Internal, "something went wrong when authorizing")
	}

	response := &api.AuthResponse{
		Token: token,
	}

	logger.Log.Info(
		"user authorized",
		zap.String("username", string(in.Username)),
	)

	return response, nil
}

func (s *Server) ListSecretsMetaStream(e *emptypb.Empty, g grpc.ServerStreamingServer[api.ListSecretsMetaResponse]) error {
	username := g.Context().Value(auth.Username("username")).([]byte)

	sess := &session{
		username: username,
	}

	ch := make(chan any)
	defer func() {
		close(ch)
		s.smap.Delete(sess)
	}()
	s.smap.Store(sess, ch)

	ctx := g.Context()

	t := time.NewTicker(time.Second * 4)

	for {

		select {
		case <-ctx.Done():
			logger.Log.Info(
				"closing metadata stream",
			)
			return ctx.Err()
		case <-ch:
			logger.Log.Info("sending metadata to client by event")
			meta, err := s.storage.ListSecretsMeta(context.Background(), username)
			if err != nil {
				return status.Errorf(codes.Internal, "error listing secrets metadata")
			}

			response := &api.ListSecretsMetaResponse{
				SecretsMeta: meta,
			}

			g.Send(response)

		case <-t.C:
			logger.Log.Info("sending metadata to client by timer")
			meta, err := s.storage.ListSecretsMeta(context.Background(), username)
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
	username := ctx.Value(auth.Username("username")).([]byte)

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
	username := ctx.Value(auth.Username("username")).([]byte)

	logger.Log.Info(
		"adding secret",
		zap.String("username", string(username)),
	)

	if err := s.storage.AddSecret(ctx, username, in); err != nil {
		logger.Log.Error(
			"error when adding secret",
			zap.String("username", string(username)),
			zap.Error(err),
		)
		return &emptypb.Empty{}, status.Errorf(codes.Internal, "pair %s already exists", in.Key)
	}

	s.notifyMetadataStreams(username)

	return &emptypb.Empty{}, nil
}

func (s *Server) ListSecretsMeta(ctx context.Context, e *emptypb.Empty) (*api.ListSecretsMetaResponse, error) {
	username := ctx.Value(auth.Username("username")).([]byte)

	logger.Log.Debug(
		"listing secrets metadata",
		zap.String("username", string(username)),
	)

	meta, err := s.storage.ListSecretsMeta(ctx, username)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error listing secrets metadata")
	}

	response := &api.ListSecretsMetaResponse{
		SecretsMeta: meta,
	}

	return response, nil
}

func (s *Server) notifyMetadataStreams(username []byte) {
	s.smap.Range(func(key, value any) bool {

		if bytes.Equal(key.(*session).username, username) {
			value.(chan any) <- struct{}{}
		}

		return true
	})
}
