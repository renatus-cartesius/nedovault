package storage

import "fmt"

func secretsDataPrefix(username []byte) []byte {
	return []byte(fmt.Sprintf("%s/%s", username, secretsData))
}

func secretsMetadataPrefix(username []byte) []byte {
	return []byte(fmt.Sprintf("%s/%s", username, secretsMetadata))
}

func authMetadataPrefix(username []byte) []byte {
	return []byte(fmt.Sprintf("%s/%s", username, authMetadata))
}
