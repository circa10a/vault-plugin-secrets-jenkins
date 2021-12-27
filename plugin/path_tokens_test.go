package jenkinssecretsengine

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"
)

// Test uses a mock backend to test creation of tokens
func TestToken(t *testing.T) {
	b, s := getTestBackend(t)
	AddTestConfig(t, b, s)

	t.Run("Create Token", func(t *testing.T) {
		resp, err := testTokenRead(t, b, s)

		require.Nil(t, err)
		require.Nil(t, resp.Error())
		require.NotNil(t, resp)
		require.Equal(t, resp.Data["token_name"], testTokenName)
	})
}

// Utility function to create a token by reading and return any errors
func testTokenRead(t *testing.T, b *jenkinsBackend, s logical.Storage) (*logical.Response, error) {
	t.Helper()
	return b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ReadOperation,
		Path:      fmt.Sprintf("%s/%s", tokensPrefix, testTokenName),
		Storage:   s,
	})
}
