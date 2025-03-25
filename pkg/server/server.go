package server

import (
	"context"
	"github.com/renatus-cartesius/nedovault/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	api.UnimplementedNedoVaultServer

	secrets map[string]*api.Secret
}

func NewServer() *Server {
	return &Server{
		secrets: make(map[string]*api.Secret),
	}
}

func (s Server) AddSecret(ctx context.Context, in *api.AddSecretRequest) (*emptypb.Empty, error) {

	if _, ok := s.secrets[in.Name]; ok {
		return &emptypb.Empty{}, status.Errorf(codes.Internal, "pair %s already exists", in.Name)
	}

	s.secrets[in.Name] = in.Secret

	return &emptypb.Empty{}, nil
}

func (s Server) ListSecrets(ctx context.Context, in *emptypb.Empty) (*api.ListSecretsResponse, error) {

	return &api.ListSecretsResponse{
		Secrets: s.secrets,
	}, nil
}
