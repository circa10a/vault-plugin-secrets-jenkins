package jenkinssecretsengine

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	jenkinsTokenType = "jenkins_token"
)

// jenkinsToken defines a secret for the Jenkins token
type jenkinsToken struct {
	Token   string        `json:"token"`
	TokenID string        `json:"token_id"`
	Name    string        `json:"token_name"`
	TTL     time.Duration `json:"ttl"`
	MaxTTL  time.Duration `json:"max_ttl"`
}

// toResponseData returns response data for a token
func (token *jenkinsToken) toResponseData() map[string]interface{} {
	respData := map[string]interface{}{
		"token":      token.Token,
		"token_name": token.Name,
		"token_id":   token.TokenID,
	}
	return respData
}

// jenkinsToken defines an api token to store for a given user
// and how it should be revoked or renewed.
func (b *jenkinsBackend) jenkinsToken() *framework.Secret {
	return &framework.Secret{
		Type: jenkinsTokenType,
		Fields: map[string]*framework.FieldSchema{
			"token": {
				Type:        framework.TypeString,
				Description: "Jenkins Token",
			},
		},
		Revoke: b.tokenRevoke,
		Renew:  b.tokenRenew,
	}
}

// tokenRevoke removes the token from the Vault storage API and calls the client to revoke the token
func (b *jenkinsBackend) tokenRevoke(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	client, err := b.getClient(ctx, req.Storage)
	if err != nil {
		return nil, fmt.Errorf("error getting client: %w", err)
	}

	tokenID := ""
	tokenIDRaw, ok := req.Secret.InternalData["token_id"]
	if ok {
		tokenID, ok = tokenIDRaw.(string)
		if !ok {
			return nil, fmt.Errorf("invalid value for token_id in secret internal data")
		}
	}

	// Delete from Jenkins
	if err := deleteToken(ctx, client, tokenID); err != nil {
		return nil, fmt.Errorf("error revoking user token: %w", err)
	}

	return nil, nil
}

// tokenRenew renews the ttl time in vault
func (b *jenkinsBackend) tokenRenew(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	ttlRaw, ok := req.Secret.InternalData["ttl"]
	if !ok {
		return nil, fmt.Errorf("secret is missing ttl internal data")
	}
	maxTtlRaw, ok := req.Secret.InternalData["max_ttl"]
	if !ok {
		return nil, fmt.Errorf("secret is missing max_ttl internal data")
	}

	resp := &logical.Response{Secret: req.Secret}
	ttl := time.Duration(ttlRaw.(float64)) * time.Second
	maxTtl := time.Duration(maxTtlRaw.(float64)) * time.Second

	if ttl > 0 {
		resp.Secret.TTL = ttl
	}
	if maxTtl > 0 {
		resp.Secret.MaxTTL = maxTtl
	}

	return resp, nil
}

// createToken calls the jenkins client to generate and return a new token
func createToken(ctx context.Context, j *jenkinsClient, tokenName string) (*jenkinsToken, error) {
	token, err := j.GenerateAPIToken(ctx, tokenName)
	if err != nil {
		return nil, fmt.Errorf("error creating jenkins token: %w", err)
	}

	return &jenkinsToken{
		Token:   token.Value,
		TokenID: token.UUID,
	}, nil
}

// deleteToken revokes the token
func deleteToken(ctx context.Context, j *jenkinsClient, tokenID string) error {
	err := j.RevokeAPIToken(ctx, tokenID)
	if err != nil {
		return err
	}

	return nil
}
