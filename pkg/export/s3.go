package export

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func (e *exporter) uploadS3() error {
	cfg := e.config.S3Config
	minioClient, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, cfg.Token),
		Secure: cfg.Secure,
	})
	if err != nil {
		return err
	}

	ctx := context.Background()

	_, err = minioClient.FPutObject(ctx, cfg.Bucket, filepath.Base(e.archive), e.archive, minio.PutObjectOptions{ContentType: "application/x-gtar"})
	if err != nil {
		return err
	}

	deleteOlderThan := time.Now().AddDate(0, 0, -e.config.ArchiveRetentionDays)

	objectCh := minioClient.ListObjects(ctx, cfg.Bucket, minio.ListObjectsOptions{Prefix: e.config.Target})
	for object := range objectCh {
		if object.Err == nil {
			if object.LastModified.Before(deleteOlderThan) {
				err = minioClient.RemoveObject(ctx, cfg.Bucket, object.Key, minio.RemoveObjectOptions{})
				if err != nil {
					return err
				}
				e.deletedArchives = append(e.deletedArchives, "s3:"+object.Key)
			}
		}
	}
	return nil
}

func (e *exporter) pruneS3Archives() error {
	_, dir, err := e.archiveDirs()
	if err != nil {
		return err
	}

	pattern := regexp.MustCompile(fmt.Sprintf(`^%s-?.*-\d{4}-\d{2}-\d{2}-\d{6}\.tar\.gz$`, filepath.Base(e.config.Target)))

	deleteOlderThan := time.Now().AddDate(0, 0, -e.config.ArchiveRetentionDays)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var matches []string
	for _, e := range entries {
		if !e.IsDir() && pattern.MatchString(e.Name()) {
			f, err := e.Info()
			if err != nil {
				return err
			}
			if f.ModTime().Before(deleteOlderThan) {
				name := filepath.Join(dir, f.Name())
				if err = os.Remove(name); err != nil {
					return err
				}
				matches = append(matches, name)
			}
		}
	}
	e.deletedArchives = matches
	return nil
}
