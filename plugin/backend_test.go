package jenkinssecretsengine

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"
)

const (
	envVarJenkinsUsername = "TEST_JENKINS_USERNAME"
	envVarJenkinsPassword = "TEST_JENKINS_PASSWORD"
	envVarJenkinsURL      = "TEST_JENKINS_URL"
	testTokenName         = "test-user-token"
)

var testUsername, testPassword, testURL string

func setTestVars() {
	if val, ok := os.LookupEnv(envVarJenkinsUsername); ok {
		testUsername = val
	} else {
		testUsername = "admin"
	}
	if val, ok := os.LookupEnv(envVarJenkinsPassword); ok {
		testPassword = val
	} else {
		testPassword = "admin"
	}
	if val, ok := os.LookupEnv(envVarJenkinsURL); ok {
		testURL = val
	} else {
		testURL = "http://localhost:8080"
	}
}

// getTestBackend will help you construct a test backend object.
func getTestBackend(tb testing.TB) (*jenkinsBackend, logical.Storage) {
	// Have ability to override test vars while also having defaults
	setTestVars()
	tb.Helper()

	config := logical.TestBackendConfig()
	config.StorageView = new(logical.InmemStorage)
	config.Logger = hclog.NewNullLogger()
	config.System = logical.TestSystemView()

	b, err := Factory(context.Background(), config)
	if err != nil {
		tb.Fatal(err)
	}

	return b.(*jenkinsBackend), config.StorageView
}

// AddConfig adds the configuration to the test backend.
// Make sure data includes all of the configuration
// attributes you need and the `config` path!
func AddTestConfig(t *testing.T, b logical.Backend, s logical.Storage) {
	req := &logical.Request{
		Operation: logical.CreateOperation,
		Path:      configPrefix,
		Storage:   s,
		Data: map[string]interface{}{
			"username": testUsername,
			"password": testPassword,
			"url":      testURL,
		},
	}
	resp, err := b.HandleRequest(context.Background(), req)
	require.Nil(t, resp)
	require.Nil(t, err)
}
