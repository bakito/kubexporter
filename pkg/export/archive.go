package export

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (e *exporter) tarGz() error {
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}

	dir := e.config.Target
	if e.config.ArchiveTarget != "" {
		dir = e.config.ArchiveTarget
	}

	var name string
	if e.config.Namespace != "" {
		name = fmt.Sprintf("%s-%s-%s.tar.gz", filepath.Base(e.config.Target), e.config.Namespace, time.Now().Format("2006-01-02-150405"))
	} else {
		name = fmt.Sprintf("%s-%s.tar.gz", filepath.Base(e.config.Target), time.Now().Format("2006-01-02-150405"))
	}
	name = filepath.Join(dir, name)
	e.l.Printf("\n    Creating archive ...\n")
	// set up the output file
	file, err := os.Create(name)
	if err != nil {
		return err
	}
	defer closeIgnoreError(file)()
	// set up the gzip writer
	gw := gzip.NewWriter(file)
	defer closeIgnoreError(gw)()
	tw := tar.NewWriter(gw)
	defer closeIgnoreError(tw)()

	walker := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(info.Name()) != fmt.Sprintf(".%s", e.config.OutputFormat()) {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer closeIgnoreError(file)()

		if err := addFile(tw, workDir, path); err != nil {
			return err
		}
		return err
	}
	e.archive = name
	return filepath.Walk(e.config.Target, walker)
}

func addFile(tw *tar.Writer, workDir, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	fPath := filepath.ToSlash(strings.Replace(path, workDir, "", 1))

	defer closeIgnoreError(file)()
	if stat, err := file.Stat(); err == nil {
		// now let's create the header as needed for this file within the tarball
		header := new(tar.Header)
		header.Name = fPath
		header.Size = stat.Size()
		header.Mode = int64(stat.Mode())
		header.ModTime = stat.ModTime()
		// write the header to the tarball archive
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		// copy the file data to the tarball
		if _, err := io.Copy(tw, file); err != nil {
			return err
		}
	}
	return nil
}

func closeIgnoreError(f io.Closer) func() {
	return func() {
		_ = f.Close()
	}
}
