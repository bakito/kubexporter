package export

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"
	"github.com/bakito/kubexporter/pkg/types"
	"google.golang.org/api/iterator"
)

// https://github.com/GoogleCloudPlatform/golang-samples/blob/main/storage/objects/upload_file.go
// https://github.com/GoogleCloudPlatform/golang-samples/blob/main/storage/objects/list_files_with_prefix.go
//nolint: unused
func (e *exporter) uploadGoogle(ctx context.Context) error {
	cfg := e.config.S3Config

	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %w", err)
	}

	defer client.Close()

	// Open local file.
	f, err := os.Open(e.archive)
	if err != nil {
		return fmt.Errorf("os.Open: %w", err)
	}
	defer f.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	o := client.Bucket(cfg.Bucket).Object(filepath.Base(e.archive))

	// Upload an object with storage.Writer.
	wc := o.NewWriter(ctx)
	if _, err = io.Copy(wc, f); err != nil {
		return fmt.Errorf("io.Copy: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("Writer.Close: %w", err)
	}

	if e.config.ArchiveRetentionDays > 0 {
		err := e.pruneGoogle(ctx, client, cfg)
		if err != nil {
			return err
		}
	}

	return nil
}

//nolint: unused
func (e *exporter) pruneGoogle(ctx context.Context, client *storage.Client, cfg *types.S3Config) error {
	deleteOlderThan := e.config.MaxArchiveAge()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	it := client.Bucket(cfg.Bucket).Objects(ctx, &storage.Query{
		Prefix: e.config.Target,
	})
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return fmt.Errorf("Bucket(%q).Objects(): %w", cfg.Bucket, err)
		}
		if attrs.Created.Before(deleteOlderThan) {

			o := client.Bucket(attrs.Bucket).Object(attrs.Name)

			o = o.If(storage.Conditions{GenerationMatch: attrs.Generation})

			if err := o.Delete(ctx); err != nil {
				return fmt.Errorf("Object(%q).Delete: %w", attrs.Name, err)
			}

			e.deletedArchives = append(e.deletedArchives, "s3:"+attrs.Name)
		}
	}
	return nil
}
