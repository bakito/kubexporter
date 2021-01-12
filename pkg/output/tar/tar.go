package tar

import (
	at "archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bakito/kubexporter/pkg/log"
	"github.com/bakito/kubexporter/pkg/output"
)

func New(target string, namespace string, outputFormat string) output.Output {
	return &tar{
		target:       target,
		namespace:    namespace,
		outputFormat: outputFormat,
	}
}

type tar struct {
	archive      string
	target       string
	namespace    string
	outputFormat string
}

func (t *tar) PrintStats(log log.YALI) {
	log.Checkf("Archive    🗜️  %s\n", t.archive)
}

func (t *tar) Do(log log.YALI) error {
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}

	var name string
	if t.namespace != "" {
		name = fmt.Sprintf("%s-%s-%s.tar.gz", filepath.Base(t.target), t.namespace, time.Now().Format("2006-01-02"))
	} else {
		name = fmt.Sprintf("%s-%s.tar.gz", filepath.Base(t.target), time.Now().Format("2006-01-02"))
	}
	name = filepath.Join(t.target, name)
	log.Printf("\n    Creating archive ...\n")
	// set up the output file
	file, err := os.Create(name)
	if err != nil {
		return err
	}
	defer closeIgnoreError(file)()
	// set up the gzip writer
	gw := gzip.NewWriter(file)
	defer closeIgnoreError(gw)()
	tw := at.NewWriter(gw)
	defer closeIgnoreError(tw)()

	walker := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(info.Name()) != fmt.Sprintf(".%s", t.outputFormat) {
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
	t.archive = name
	return filepath.Walk(t.target, walker)
}

func addFile(tw *at.Writer, workDir, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	fPath := strings.Replace(path, workDir, "", 1)

	defer closeIgnoreError(file)()
	if stat, err := file.Stat(); err == nil {
		// now lets create the header as needed for this file within the tarball
		header := new(at.Header)
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
