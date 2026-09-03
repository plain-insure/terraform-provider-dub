// Copyright (c) HashiCorp, Inc.

// Package main provides the entrypoint for the terraform-provider-dub
// Terraform provider binary.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/plain-insure/terraform-provider-dub/internal/provider"
)

// version is set at build time via -ldflags "-X main.version=..."
var version = "dev"

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/plain-insure/dub",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
