package main

import (
	"bufio"
	"fmt"
	"github.com/coreos/go-semver/semver"
	"os"
	"os/exec"
	"strings"
)

func main() {
	cmd := exec.Command("git", "describe")
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	version := strings.TrimSpace(string(out))
	if strings.HasPrefix(version, "v") {
		version = version[1:]
	}
	v := semver.New(version)
	v.BumpPatch()
	reader := bufio.NewReader(os.Stdin)
	if _, err = fmt.Fprintf(os.Stderr, "Enter Release Version: [v%v] ", v); err != nil {
		panic(err)
	}

	text, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	if strings.HasPrefix(text, "v") {
		v = semver.New(version)
	}
	if _, err = fmt.Fprintf(os.Stderr, "Using Version: v%v\n", v); err != nil {
		panic(err)
	}
	fmt.Printf("v%v", v)
}
