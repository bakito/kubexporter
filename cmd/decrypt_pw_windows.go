//go:build windows
// +build windows

package cmd

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

func readKey() (string, error) {
	fmt.Println("Please the aes key: ")
	key, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}
	return string(key), nil
}
