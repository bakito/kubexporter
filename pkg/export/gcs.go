package export

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	"github.com/bakito/kubexporter/pkg/types"
)

func (e *exporter) uploadGCS(ctx context.Context) error {
	cfg := e.config.GCSConfig

	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	// Open the file for reading
	f, err := os.Open(e.archive)
	if err != nil {
		return err
	}
	defer f.Close()

	// Create a writer for the GCS object
	obj := client.Bucket(cfg.Bucket).Object(filepath.Base(e.archive))
	wc := obj.NewWriter(ctx)
	if _, err = io.Copy(wc, f); err != nil {
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}

	if e.config.ArchiveRetentionDays > 0 {
		err := e.pruneGCS(ctx, client, cfg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *exporter) pruneGCS(ctx context.Context, client *storage.Client, cfg *types.GCSConfig) error {
	deleteOlderThan := e.config.MaxArchiveAge()

	it := client.Bucket(cfg.Bucket).Objects(ctx, &storage.Query{Prefix: e.config.Target})
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}

		if attrs.Created.Before(deleteOlderThan) {
			obj := client.Bucket(cfg.Bucket).Object(attrs.Name)
			err := obj.Delete(ctx)
			if err != nil {
				return err
			}
			e.deletedArchives = append(e.deletedArchives, "gcs:"+attrs.Name)
		}
	}
	return nil
}
