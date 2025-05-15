package server

import (
	"context"
	"errors"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
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

		token, ok := md["token"]
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "request without token")
		}

		claims, err := a.ParseToken(ctx, []byte(token[0]))
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
func NewAuthStreamInterceptor(a Auth) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {

		ctx := ss.Context()

		if info.FullMethod == api.NedoVault_Authorize_FullMethodName {
			return handler(srv, ss)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return status.Errorf(codes.Unauthenticated, "metadata is not provided")
		}

		token, ok := md["token"]
		if !ok {
			return status.Errorf(codes.Unauthenticated, "request without token")
		}

		claims, err := a.ParseToken(ctx, []byte(token[0]))
		if err != nil {
			if errors.Is(err, auth.ErrInvalidToken) {

				return status.Errorf(codes.Unauthenticated, "request with invalid token")
			}

			logger.Log.Error(
				"error parsing client token",
				zap.Error(err),
				zap.String("token", token[0]),
			)

			return err
		}

		logger.Log.Info(
			"accepted stream request",
			zap.String("method", info.FullMethod),
			zap.String("user", claims.Username),
		)

		type wrappedStream struct {
			grpc.ServerStream
			ctx context.Context
		}

		return handler(srv, &grpc_middleware.WrappedServerStream{
			ServerStream:   ss,
			WrappedContext: context.WithValue(ctx, auth.Username("username"), []byte(claims.Username)),
		})
	}
}
