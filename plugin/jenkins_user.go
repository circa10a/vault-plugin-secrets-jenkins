package jenkinssecretsengine

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	jenkinsUserType = "jenkins_user"
)

// jenkinsUser defines a user as secret
type jenkinsUser struct {
	Username string        `json:"username"`
	Password string        `json:"password,omitempty"`
	Fullname string        `json:"fullname"`
	Email    string        `json:"email"`
	TTL      time.Duration `json:"ttl"`
	MaxTTL   time.Duration `json:"max_ttl"`
}

// toResponseData returns response data for a user
func (user *jenkinsUser) toResponseData() map[string]interface{} {
	respData := map[string]interface{}{
		"username": user.Username,
		"fullname": user.Fullname,
		"email":    user.Email,
	}
	return respData
}

// jenkinsUser defines an a user in jenkins
// and how it should be revoked or renewed.
func (b *jenkinsBackend) jenkinsUser() *framework.Secret {
	return &framework.Secret{
		Type: jenkinsUserType,
		Fields: map[string]*framework.FieldSchema{
			"username": {
				Type:        framework.TypeString,
				Description: "Jenkins User",
			},
		},
		Revoke: b.userRevoke,
		Renew:  b.userRenew,
	}
}

// userRevoke removes the user from the Vault storage API and calls the client to revoke the user
func (b *jenkinsBackend) userRevoke(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	client, err := b.getClient(ctx, req.Storage)
	if err != nil {
		return nil, fmt.Errorf("error getting client: %w", err)
	}

	username := ""
	usernameRaw, ok := req.Secret.InternalData["username"]
	if ok {
		username, ok = usernameRaw.(string)
		if !ok {
			return nil, fmt.Errorf("invalid value for username in secret internal data")
		}
	}

	// Delete from Jenkins
	if err := deleteUser(ctx, client, username); err != nil {
		return nil, fmt.Errorf("error revoking user: %w", err)
	}

	// Delete from store
	err = req.Storage.Delete(ctx, b.getUserPath(username))
	if err != nil {
		return nil, fmt.Errorf("error remove user from storage: %w", err)
	}

	return nil, nil
}

// userRenew renews the ttl time in vault
func (b *jenkinsBackend) userRenew(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
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

// createUser calls the jenkins client to create and return a new user
func createUser(ctx context.Context, j *jenkinsClient, username, password, fullname, email string) (*jenkinsUser, error) {
	user, err := j.CreateUser(ctx, username, password, fullname, email)
	if err != nil {
		return nil, fmt.Errorf("error creating jenkins user: %w", err)
	}

	return &jenkinsUser{
		Username: user.UserName,
		Password: password,
		Fullname: user.FullName,
		Email:    user.Email,
	}, nil
}

// deleteUser revokes the user
func deleteUser(ctx context.Context, j *jenkinsClient, username string) error {
	err := j.DeleteUser(ctx, username)
	if err != nil {
		return err
	}

	return nil
}

// getUser gets the user from the Vault storage API
func (b *jenkinsBackend) getUserFromStorage(ctx context.Context, s logical.Storage, username string) (*jenkinsUser, error) {
	if username == "" {
		return nil, fmt.Errorf("missing username")
	}

	entry, err := s.Get(ctx, b.getUserPath(username))
	if err != nil {
		return nil, err
	}

	if entry == nil {
		return nil, nil
	}

	var user jenkinsUser

	if err := entry.DecodeJSON(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

// getUserPath returns the user storage path such as /users/user
func (b *jenkinsBackend) getUserPath(username string) string {
	return fmt.Sprintf("%s/%s", usersPrefix, username)
}
