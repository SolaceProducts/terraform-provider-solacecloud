// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"flag"
	"log"

	"terraform-provider-solacecloud/internal/provider"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// Run "go generate" to format example terraform files and validate the docs for the registry/website

// If you do not have terraform installed, you can remove the formatting command, but its suggested to
// ensure the documentation is formatted properly.
//go:generate terraform fmt -recursive ./examples/

// The docs in docs/ are hand-written, so validate them instead of generating them: "tfplugindocs generate"
// would overwrite them. Validation builds the provider and needs a terraform binary on PATH (or downloads one).
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.25.0 validate -provider-name solacecloud

var (
	// these will be set by the goreleaser configuration
	// to appropriate values for the compiled binary.
	// updating this will trigger a build
	version string = "0.2.11"

	// goreleaser can pass other information to the main package, such as the specific commit
	// https://goreleaser.com/cookbooks/using-main.version/
)

const providerNamespace = "solaceproducts"
const providerSimpleType = "solacecloud"
const providerType = providerSimpleType

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io" + "/" + providerNamespace + "/" + providerType,
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
