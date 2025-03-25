package main

import (
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/bakito/kubexporter/cmd"
)

func main() {
	cmd.Execute()
}
