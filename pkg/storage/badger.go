package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dgraph-io/badger/v4"
	"github.com/renatus-cartesius/metricserv/pkg/logger"
	"github.com/renatus-cartesius/nedovault/api"
	"github.com/renatus-cartesius/nedovault/internal/auth"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	secretsData     = "secrets_data"
	secretsMetadata = "secrets_metadata"
	authMetadata    = "auth_metadata"
)

func NewBadgerStorage(db *badger.DB) *BadgerStorage {
	return &BadgerStorage{
		db: db,
	}
}

type BadgerStorage struct {
	db *badger.DB
}

// GetAuthMeta getting user`s auth metadata from underlying storage
func (b *BadgerStorage) GetAuthMeta(ctx context.Context, username []byte) (*auth.Meta, error) {
	authMetadataPath := authMetadataPrefix(username)

	authMeta := &auth.Meta{}

	err := b.db.View(func(txn *badger.Txn) error {
		valCopy := make([]byte, 0)

		authMetaItem, err := txn.Get(authMetadataPath)
		if err != nil {
			return err
		}

		valCopy, err = authMetaItem.ValueCopy(nil)
		if err != nil {
			return err
		}

		if err = json.Unmarshal(valCopy, authMeta); err != nil {
			return err
		}

		return nil
	})

	if err != nil {

		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil, nil
		}

		logger.Log.Error(
			"error getting auth meta from storage",
			zap.Error(err),
		)
		return nil, err
	}

	return authMeta, nil
}

// AddAuthMeta adding user`s auth metadata to underlying storage
func (b *BadgerStorage) AddAuthMeta(ctx context.Context, username []byte, meta *auth.Meta) error {
	authMetadataPath := authMetadataPrefix(username)

	err := b.db.Update(func(txn *badger.Txn) error {
		txn = b.db.NewTransaction(true)

		var aMetaRaw bytes.Buffer
		if err := json.NewEncoder(&aMetaRaw).Encode(meta); err != nil {

			logger.Log.Error(
				"error marshalling auth metadata",
				zap.Error(err),
			)

			return err
		}

		if err := txn.Set(authMetadataPath, aMetaRaw.Bytes()); err != nil {
			return err
		}

		if err := txn.Commit(); err != nil {
			return err
		}

		return nil
	})

	return err
}

func (b *BadgerStorage) AppendTokens(ctx context.Context, username, token []byte) error {
	return nil
}

func (b *BadgerStorage) ListSecretsMeta(ctx context.Context, username []byte) ([]*api.SecretMeta, error) {

	prefix := secretsMetadataPrefix(username)
	secretsKeys := make([]*api.SecretMeta, 0)

	err := b.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			err := item.Value(func(v []byte) error {
				metadata := &api.SecretMeta{}
				if err := proto.Unmarshal(v, metadata); err != nil {
					logger.Log.Error(
						"error unmarshalling metadata",
						zap.Error(err),
					)
					return err
				}

				secretsKeys = append(secretsKeys, metadata)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	return secretsKeys, err
}

func (b *BadgerStorage) AddSecret(ctx context.Context, username []byte, in *api.AddSecretRequest) error {
	dataPath := []byte(fmt.Sprintf("%s/%s", secretsDataPrefix(username), in.GetKey()))
	metadataPath := []byte(fmt.Sprintf("%s/%s", secretsMetadataPrefix(username), in.GetKey()))

	err := b.db.Update(func(txn *badger.Txn) error {
		txn = b.db.NewTransaction(true)

		sDataRaw, err := proto.Marshal(in.GetSecret())
		if err = txn.Set(dataPath, sDataRaw); err != nil {
			return err
		}

		sMetadata := &api.SecretMeta{
			Key:       in.GetKey(),
			Name:      in.Name,
			Timestamp: timestamppb.Now(),
			Type:      in.GetSecretType(),
		}

		sMetadataRaw, err := proto.Marshal(sMetadata)
		if err != nil {
			return err
		}

		if err = txn.Set(metadataPath, sMetadataRaw); err != nil {
			return err
		}

		if err = txn.Commit(); err != nil {
			return err
		}

		return nil
	})

	return err

}

func (b *BadgerStorage) GetSecret(ctx context.Context, username, key []byte) (*api.Secret, *api.SecretMeta, error) {
	dataPath := []byte(fmt.Sprintf("%s/%s", secretsDataPrefix(username), key))
	metadataPath := []byte(fmt.Sprintf("%s/%s", secretsMetadataPrefix(username), key))

	secret := &api.Secret{}
	secretMeta := &api.SecretMeta{}

	err := b.db.View(func(txn *badger.Txn) error {
		valCopy := make([]byte, 0)

		secretItem, err := txn.Get(dataPath)
		if err != nil {
			return err
		}

		secretMetaItem, err := txn.Get(metadataPath)
		if err != nil {
			return err
		}

		valCopy, err = secretItem.ValueCopy(nil)
		if err != nil {
			return err
		}
		if err = proto.Unmarshal(valCopy, secret); err != nil {
			return err
		}

		valCopy, err = secretMetaItem.ValueCopy(nil)
		if err != nil {
			return err
		}
		if err = proto.Unmarshal(valCopy, secretMeta); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return secret, secretMeta, nil
}
