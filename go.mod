module github.com/bakito/kubexporter

go 1.16

require (
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/coreos/go-semver v0.3.0
	github.com/ghodss/yaml v1.0.0
	github.com/golang/mock v1.5.0
	github.com/google/uuid v1.2.0
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/imdario/mergo v0.3.9 // indirect
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/olekukonko/tablewriter v0.0.0-20170122224234-a0225b3f23b5
	github.com/onsi/ginkgo v1.15.1
	github.com/onsi/gomega v1.11.0
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/vardius/worker-pool/v2 v2.1.0
	github.com/vbauerster/mpb/v5 v5.4.0
	k8s.io/api v0.20.4
	k8s.io/apimachinery v0.20.4
	k8s.io/cli-runtime v0.20.4
	k8s.io/client-go v0.20.4
	k8s.io/klog/v2 v2.8.0
	k8s.io/kubectl v0.20.4
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920
)

// fix for darwin
replace golang.org/x/sys => golang.org/x/sys v0.0.0-20200826173525-f9321e4c35a6
