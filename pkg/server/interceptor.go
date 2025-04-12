package server

import (
	"context"
	"errors"
	"github.com/renatus-cartesius/metricserv/pkg/logger"
	"github.com/renatus-cartesius/nedovault/api"
	"github.com/renatus-cartesius/nedovault/internal/auth"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func NewAuthUnaryInterceptor(a Auth) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {

		if info.FullMethod == api.NedoVault_Authorize_FullMethodName {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "metadata is not provided")
		}

		token := md["token"][0]

		claims, err := a.ParseToken(ctx, []byte(token))
		if err != nil {
			if errors.Is(err, auth.ErrInvalidToken) {

				return nil, status.Errorf(codes.Unauthenticated, "request with invalid token")
			}

			logger.Log.Error(
				"error parsing client token",
				zap.Error(err),
			)

			return nil, err
		}

		logger.Log.Info(
			"accepted request",
			zap.String("method", info.FullMethod),
			zap.String("user", claims.Username),
		)

		ctx = context.WithValue(ctx, auth.Username("username"), []byte(claims.Username))

		return handler(ctx, req)
	}
}
