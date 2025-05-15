package auth

import (
	"context"
	"errors"
	"github.com/golang-jwt/jwt/v4"
	"github.com/renatus-cartesius/metricserv/pkg/logger"
	"github.com/renatus-cartesius/nedovault/api"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"reflect"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("client passed invalid credentials")
	ErrMetadataGet        = errors.New("something went wrong when getting auth metadata")
	ErrInvalidToken       = errors.New("client passed invalid token")
)

type Username string

type Storage interface {
	GetAuthMeta(ctx context.Context, username []byte) (*Meta, error)
	AddAuthMeta(ctx context.Context, username []byte, meta *Meta) error
	AppendTokens(ctx context.Context, username, token []byte) error
}

type Meta struct {
	Hash   []byte
	Tokens []jwt.Token
}

type Claims struct {
	jwt.RegisteredClaims
	Username string `json:"username"`
}

type LocalAuth struct {
	key      []byte
	tokenTTL time.Duration
	storage  Storage

	method jwt.SigningMethod
}

func (a *LocalAuth) ParseToken(ctx context.Context, tok []byte) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		string(tok),
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {

			if reflect.TypeOf(token.Method) != reflect.TypeOf(a.method) {
				logger.Log.Error(
					"passed token has invalid signing type",
				)
				return nil, ErrInvalidToken
			}

			return a.key, nil
		},
	)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)

	if !ok || !token.Valid {

		return nil, ErrInvalidToken
	}

	return claims, nil
}

func NewLocalAuth(key []byte, tokenTTL time.Duration, storage Storage, method jwt.SigningMethod) *LocalAuth {
	return &LocalAuth{
		key:      key,
		tokenTTL: tokenTTL,
		storage:  storage,
		method:   method,
	}
}

func (a *LocalAuth) Authorize(ctx context.Context, in *api.AuthRequest) (string, error) {

	// check if user exists
	meta, err := a.storage.GetAuthMeta(ctx, in.Username)
	if err != nil {

		logger.Log.Error(
			"error on getting auth meta",
			zap.Error(err),
		)

		return "", ErrMetadataGet
	}

	if meta != nil {

		logger.Log.Debug(
			"comparing password with hash",
			zap.String("username", string(in.Username)),
		)

		// comparing hash and pass with bcrypt
		if err = bcrypt.CompareHashAndPassword(meta.Hash, in.Password); err != nil {
			return "", err
		}

		// getting token and return it

		return a.IssueToken(ctx, in.Username)
	} else {
		// adding auth metadata for user
		logger.Log.Debug(
			"adding auth metadata for user",
			zap.String("username", string(in.Username)),
		)

		hash, err := bcrypt.GenerateFromPassword(in.Password, 1)
		if err != nil {
			logger.Log.Error(
				"error generating password hash",
				zap.String("username", string(in.Username)),
			)
		}

		meta = &Meta{
			Hash: hash,
		}

		if err = a.storage.AddAuthMeta(ctx, in.Username, meta); err != nil {
			return "", err
		}

	}

	return a.IssueToken(ctx, in.Username)
}

func (a *LocalAuth) IssueToken(ctx context.Context, username []byte) (string, error) {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:  "",
			Subject: "",
			ExpiresAt: &jwt.NumericDate{
				Time: time.Now().Add(a.tokenTTL),
			},
			NotBefore: nil,
			IssuedAt: &jwt.NumericDate{
				Time: time.Now(),
			},
			ID: "",
		},
		Username: string(username),
	}

	token, err := jwt.NewWithClaims(a.method, claims).SignedString([]byte(a.key))
	if err != nil {

		logger.Log.Error(
			"error creating new jwt token",
			zap.Error(err),
		)

		return "", err
	}

	// we need to append a new token to user existing tokens
	err = a.storage.AppendTokens(ctx, username, []byte(token))

	return token, err
}
