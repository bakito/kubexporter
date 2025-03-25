package export

import (
	"context"
	"path/filepath"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/bakito/kubexporter/pkg/types"
)

func (e *exporter) uploadS3(ctx context.Context) error {
	cfg := e.config.S3Config
	minioClient, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, cfg.Token),
		Secure: cfg.Secure,
	})
	if err != nil {
		return err
	}

	_, err = minioClient.FPutObject(
		ctx,
		cfg.Bucket,
		filepath.Base(e.archive),
		e.archive,
		minio.PutObjectOptions{ContentType: "application/x-gtar"},
	)
	if err != nil {
		return err
	}

	if e.config.ArchiveRetentionDays > 0 {
		err := e.pruneS3(ctx, minioClient, cfg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *exporter) pruneS3(ctx context.Context, minioClient *minio.Client, cfg *types.S3Config) error {
	deleteOlderThan := e.config.MaxArchiveAge()
	objectCh := minioClient.ListObjects(ctx, cfg.Bucket, minio.ListObjectsOptions{Prefix: e.config.Target})
	for object := range objectCh {
		if object.Err == nil {
			if object.LastModified.Before(deleteOlderThan) {
				err := minioClient.RemoveObject(ctx, cfg.Bucket, object.Key, minio.RemoveObjectOptions{})
				if err != nil {
					return err
				}
				e.deletedArchives = append(e.deletedArchives, "s3:"+object.Key)
			}
		}
	}
	return nil
}
