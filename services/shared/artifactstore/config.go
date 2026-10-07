package artifactstore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// FromEnv builds the configured artifact store.
//
// ARTIFACT_BACKEND selects "filesystem" (default; ARTIFACT_LOCAL_DIR) or
// "s3" (ARTIFACT_S3_ENDPOINT, ARTIFACT_S3_REGION, ARTIFACT_S3_BUCKET,
// ARTIFACT_S3_ACCESS_KEY_ID, ARTIFACT_S3_SECRET_ACCESS_KEY). Error messages
// name variables, never their values.
func FromEnv(maxBytes int64) (Store, error) {
	switch backend := os.Getenv("ARTIFACT_BACKEND"); backend {
	case "", "filesystem":
		dir := os.Getenv("ARTIFACT_LOCAL_DIR")
		if dir == "" {
			dir = "./artifacts"
		}
		absolute, err := filepath.Abs(dir)
		if err != nil {
			return nil, fmt.Errorf("resolve ARTIFACT_LOCAL_DIR: %w", err)
		}
		return Filesystem{Directory: absolute, MaxBytes: maxBytes}, nil
	case "s3":
		store := S3{
			Endpoint:        os.Getenv("ARTIFACT_S3_ENDPOINT"),
			Region:          os.Getenv("ARTIFACT_S3_REGION"),
			Bucket:          os.Getenv("ARTIFACT_S3_BUCKET"),
			AccessKeyID:     os.Getenv("ARTIFACT_S3_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("ARTIFACT_S3_SECRET_ACCESS_KEY"),
			MaxBytes:        maxBytes,
		}
		for name, value := range map[string]string{
			"ARTIFACT_S3_ENDPOINT":          store.Endpoint,
			"ARTIFACT_S3_REGION":            store.Region,
			"ARTIFACT_S3_BUCKET":            store.Bucket,
			"ARTIFACT_S3_ACCESS_KEY_ID":     store.AccessKeyID,
			"ARTIFACT_S3_SECRET_ACCESS_KEY": store.SecretAccessKey,
		} {
			if value == "" {
				return nil, fmt.Errorf("%s is required when ARTIFACT_BACKEND=s3", name)
			}
		}
		if err := store.validate(); err != nil {
			return nil, err
		}
		return store, nil
	default:
		return nil, errors.New("ARTIFACT_BACKEND must be filesystem or s3")
	}
}
