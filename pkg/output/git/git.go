package git

import (
	at "archive/tar"
	"github.com/bakito/kubexporter/pkg/log"
	"github.com/bakito/kubexporter/pkg/output"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"io"
	"os"
	"strings"
)

func New(cfg Git) output.Output {
	return &git{
		cfg: cfg,
	}
}

type git struct {
	cfg Git
}

func (g *git) PrintStats(log log.YALI) {
}

func (g *git) Do(log log.YALI) error {
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}

	co := &gogit.CloneOptions{
		URL:      g.cfg.Repository,
		Progress: os.Stdout,
	}

	if g.cfg.SSHKey != "" {
		publicKeys, err := ssh.NewPublicKeysFromFile("git", g.cfg.SSHKey, g.cfg.Password)
		if err != nil {
			return err
		}
		co.Auth = publicKeys
	} else {
		co.Auth = &http.BasicAuth{Username: g.cfg.Username, Password: g.cfg.Password}
	}

	repo, err := gogit.PlainClone(workDir+"/asd", false, co)
	if err != nil {
		return err
	}
	println(repo)
	return err
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
