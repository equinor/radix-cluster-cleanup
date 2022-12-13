package main

import (
	"github.com/equinor/radix-cluster-cleanup/cmd"
	// Force loading of needed authentication library
	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
)

func init() {
}

func main() {
	cmd.Execute()
}
