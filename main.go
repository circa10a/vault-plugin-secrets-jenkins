package main

import (
	"os"

	jenkinssecretsengine "github.com/circa10a/vault-plugin-secrets-jenkins/plugin"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/plugin"
)

func main() {
	apiClientMeta := &api.PluginAPIClientMeta{}
	flags := apiClientMeta.FlagSet()
	// nolint
	flags.Parse(os.Args[1:])

	tlsConfig := apiClientMeta.GetTLSConfig()
	tlsProviderFunc := api.VaultPluginTLSProvider(tlsConfig)
	logger := hclog.New(&hclog.LoggerOptions{})

	err := plugin.Serve(&plugin.ServeOpts{
		BackendFactoryFunc: jenkinssecretsengine.Factory,
		TLSProviderFunc:    tlsProviderFunc,
	})
	if err != nil {
		logger.Error("plugin shutting down", "error", err)
		os.Exit(1)
	}
}
