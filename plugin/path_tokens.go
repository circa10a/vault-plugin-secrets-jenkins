package jenkinssecretsengine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const tokensPrefix = "tokens"

// pathTokens extends the Vault API with a `/tokens`
// endpoint for an api token.
func pathTokens(b *jenkinsBackend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: fmt.Sprintf("%s/%s", tokensPrefix, framework.GenericNameRegex("name")),
			Fields: map[string]*framework.FieldSchema{
				"ttl": {
					Type:        framework.TypeDurationSecond,
					Description: "Default lease for generated token. If not set or set to 0, will use system default.",
					Required:    false,
				},
				"max_ttl": {
					Type:        framework.TypeDurationSecond,
					Description: "Maximum time for token. If not set or set to 0, will use system default.",
					Required:    false,
				},
			},
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.ReadOperation:   b.pathTokensRead,
				logical.UpdateOperation: b.pathTokensRead,
			},
			HelpSynopsis:    pathTokensHelpSyn,
			HelpDescription: pathTokensHelpDesc,
		},
	}
}

// pathTokensRead creates a new Jenkins token each time it is called if a user exists.
func (b *jenkinsBackend) pathTokensRead(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	ttl := time.Duration(d.Get("ttl").(int)) * time.Second
	maxTtl := time.Duration(d.Get("max_ttl").(int)) * time.Second
	jenkinsTokenConfig := &jenkinsToken{
		TTL:    ttl,
		MaxTTL: maxTtl,
	}

	return b.createUserToken(ctx, req, *jenkinsTokenConfig)
}

// createUserToken creates a new Jenkins token to store into the Vault backend, generates
// a response with the secrets information, and checks the TTL and MaxTTL attributes.
func (b *jenkinsBackend) createUserToken(ctx context.Context, req *logical.Request, jenkinsToken jenkinsToken) (*logical.Response, error) {
	tokenName := strings.TrimPrefix(req.Path, fmt.Sprintf("%s/", tokensPrefix))
	token, err := b.createToken(ctx, req.Storage, tokenName)
	if err != nil {
		return nil, err
	}

	// We won't store the token
	// It's only available in the initial read response
	token.Name = tokenName

	// Need to store token ID to revoke later
	internalData := map[string]interface{}{
		"token_id":   token.TokenID,
		"token_name": tokenName,
		"ttl":        token.TTL,
		"max_ttl":    token.MaxTTL,
	}

	// Create secret with lease
	resp := b.Secret(jenkinsTokenType).Response(token.toResponseData(), internalData)

	if jenkinsToken.TTL > 0 {
		resp.Secret.TTL = jenkinsToken.TTL
	}
	if jenkinsToken.MaxTTL > 0 {
		resp.Secret.MaxTTL = jenkinsToken.MaxTTL
	}

	return resp, nil
}

// createToken uses the Jenkins client create a new token
func (b *jenkinsBackend) createToken(ctx context.Context, s logical.Storage, tokenName string) (*jenkinsToken, error) {
	client, err := b.getClient(ctx, s)
	if err != nil {
		return nil, err
	}

	var token *jenkinsToken

	token, err = createToken(ctx, client, tokenName)
	if err != nil {
		return nil, fmt.Errorf("error creating Jenkins token: %w", err)
	}

	if token == nil {
		return nil, errors.New("error creating Jenkins token")
	}

	return token, nil
}

const (
	pathTokensHelpSyn = `
Generate a Jenkins API token for the configured user.
`

	pathTokensHelpDesc = `
This path generates a Jenkins API tokens
for the user configured under the /config mount.
`
)
