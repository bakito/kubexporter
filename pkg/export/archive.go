package export

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

func (e *exporter) tarGz() error {
	var name string
	if e.config.Namespace != "" {
		name = fmt.Sprintf("%s-%s-%s.tar.gz", e.config.Target, e.config.Namespace, time.Now().Format("2006-01-02"))
	} else {
		name = fmt.Sprintf("%s-%s.tar.gz", e.config.Target, time.Now().Format("2006-01-02"))
	}
	name = filepath.Join(e.config.Target, name)
	fmt.Printf("\ncreating archive\n")
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
		if info.IsDir() || filepath.Ext(info.Name()) != fmt.Sprintf(".%s", e.config.OutputFormat) {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer closeIgnoreError(file)()

		if err := addFile(tw, path); err != nil {
			return err
		}
		return err
	}
	err = filepath.Walk(e.config.Target, walker)
	if err != nil {
		return err
	}
	fmt.Printf("created archive %s\n", name)
	return nil
}

func addFile(tw *tar.Writer, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer closeIgnoreError(file)()
	if stat, err := file.Stat(); err == nil {
		// now lets create the header as needed for this file within the tarball
		header := new(tar.Header)
		header.Name = path
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
