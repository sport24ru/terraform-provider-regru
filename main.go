package main

import (
	"log"
	"terraform-provider-regru/provider"
	"terraform-provider-regru/version"

	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	log.Printf("[INFO] Starting Reg.ru DNS Provider %s", version.Full())

	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: provider.Provider,
	})
}
