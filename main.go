package main

import (
	"context"
	"flag"
	"log"

	"github.com/dkurasov/incidentgarden-terraform-provider/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run the provider with debugger support")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/incidentgarden/incidentgarden",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err)
	}
}
