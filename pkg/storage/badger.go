package storage

import (
	"context"
	"fmt"
	"github.com/dgraph-io/badger/v4"
	"github.com/renatus-cartesius/metricserv/pkg/logger"
	"github.com/renatus-cartesius/nedovault/api"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	secretsData     = "secrets_data"
	secretsMetadata = "secrets_metadata"
)

func NewBadgerStorage(db *badger.DB) *BadgerStorage {
	return &BadgerStorage{
		db: db,
	}
}

type BadgerStorage struct {
	db *badger.DB
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
		if err := txn.Set(dataPath, sDataRaw); err != nil {
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
