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

const usersPrefix = "users"

// pathUsers extends the Vault API with a `/user`
// endpoint for a user.
func pathUsers(b *jenkinsBackend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: fmt.Sprintf("%s/%s", usersPrefix, framework.GenericNameRegex("name")),
			Fields: map[string]*framework.FieldSchema{
				"password": {
					Type:        framework.TypeString,
					Description: "Password for the Jenkins user",
					Required:    true,
					DisplayAttrs: &framework.DisplayAttributes{
						Sensitive: true,
					},
				},
				"fullname": {
					Type:        framework.TypeString,
					Description: "Fullname for the Jenkins user",
					Required:    true,
				},
				"email": {
					Type:        framework.TypeString,
					Description: "Email for the Jenkins user",
					Required:    true,
				},
				"ttl": {
					Type:        framework.TypeDurationSecond,
					Description: "Default lease for a user. If not set or set to 0, will use system default.",
					Required:    false,
				},
				"max_ttl": {
					Type:        framework.TypeDurationSecond,
					Description: "Maximum time for a user. If not set or set to 0, will use system default.",
					Required:    false,
				},
			},
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.ReadOperation:   b.pathUsersRead,
				logical.CreateOperation: b.pathUsersWrite,
				logical.UpdateOperation: b.pathUsersWrite,
				logical.DeleteOperation: b.pathUsersDelete,
			},
			ExistenceCheck:  b.pathUsersExistenceCheck,
			HelpSynopsis:    pathUsersHelpSyn,
			HelpDescription: pathUsersHelpDesc,
		},
		{
			Pattern: fmt.Sprintf("%s/?$", usersPrefix),
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ListOperation: &framework.PathOperation{
					Callback: b.pathUsersList,
				},
			},
			HelpSynopsis:    pathUsersListHelpSyn,
			HelpDescription: pathUsersListHelpDescription,
		},
	}
}

// pathUsersExistenceCheck verifies if a user exists.
func (b *jenkinsBackend) pathUsersExistenceCheck(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
	out, err := req.Storage.Get(ctx, req.Path)
	if err != nil {
		return false, fmt.Errorf("existence check failed: %w", err)
	}

	return out != nil, nil
}

// pathUserList makes a request to Vault storage to retrieve a list of roles for the backend
func (b *jenkinsBackend) pathUsersList(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	entries, err := req.Storage.List(ctx, fmt.Sprintf("%s/", usersPrefix))
	if err != nil {
		return nil, err
	}

	return logical.ListResponse(entries), nil
}

// pathUsersRead returns a Jenkins user object in storage
func (b *jenkinsBackend) pathUsersRead(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	username := b.parseUsernameFromPath(req.Path)
	entry, err := b.getUserFromStorage(ctx, req.Storage, username)
	if err != nil {
		return nil, err
	}

	if entry == nil {
		return nil, nil
	}

	return &logical.Response{
		Data: entry.toResponseData(),
	}, nil
}

// pathUsersDelete deletes a Jenkins user
func (b *jenkinsBackend) pathUsersDelete(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	username := b.parseUsernameFromPath(req.Path)

	client, err := b.getClient(ctx, req.Storage)
	if err != nil {
		return nil, err
	}

	err = deleteUser(ctx, client, username)
	if err != nil {
		return logical.ErrorResponse(err.Error()), err
	}

	err = req.Storage.Delete(ctx, b.getUserPath(username))
	if err != nil {
		return logical.ErrorResponse(err.Error()), err
	}

	return nil, nil
}

// pathUsersWrite creates a new Jenkins user each time it is called if a user doesn't exist.
func (b *jenkinsBackend) pathUsersWrite(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	exists, err := b.pathUsersExistenceCheck(ctx, req, d)
	if err != nil {
		return logical.ErrorResponse(err.Error()), err
	}

	if exists {
		return logical.ErrorResponse("user already exists"), nil
	}

	username := b.parseUsernameFromPath(req.Path)
	password := d.Get("password").(string)
	fullname := d.Get("fullname").(string)
	email := d.Get("email").(string)
	ttl := time.Duration(d.Get("ttl").(int)) * time.Second
	maxTtl := time.Duration(d.Get("max_ttl").(int)) * time.Second
	jenkinsUserConfig := &jenkinsUser{
		Username: username,
		Password: password,
		Fullname: fullname,
		Email:    email,
		TTL:      ttl,
		MaxTTL:   maxTtl,
	}

	return b.createJenkinsUser(ctx, req, *jenkinsUserConfig)
}

// createJenkinsUser creates a new Jenkins user to store into the Vault backend, generates
// a response with the user information, and checks the TTL and MaxTTL attributes.
func (b *jenkinsBackend) createJenkinsUser(ctx context.Context, req *logical.Request, jenkinsUser jenkinsUser) (*logical.Response, error) {
	user, err := b.createUser(ctx, req.Storage, jenkinsUser)
	if err != nil {
		return nil, err
	}

	// We won't store the password
	// Need to store username to revoke later, ttl to renew later
	internalData := map[string]interface{}{
		"username": user.Username,
		"fullname": user.Fullname,
		"email":    user.Email,
		"ttl":      user.TTL,
		"max_ttl":  user.MaxTTL,
	}

	// Create secret with lease
	resp := b.Secret(jenkinsUserType).Response(user.toResponseData(), internalData)

	// Create thing to store
	entry, err := logical.StorageEntryJSON(b.getUserPath(jenkinsUser.Username), internalData)
	if err != nil {
		return logical.ErrorResponse("error creating user storage entry"), err
	}

	// Write to storage to view user inventory
	err = req.Storage.Put(ctx, entry)
	if err != nil {
		return logical.ErrorResponse("error writing user to internal storage"), err
	}

	// Set TTL
	if jenkinsUser.TTL > 0 {
		resp.Secret.TTL = jenkinsUser.TTL
	}
	if jenkinsUser.MaxTTL > 0 {
		resp.Secret.MaxTTL = jenkinsUser.MaxTTL
	}

	return resp, nil
}

// createUser uses the Jenkins client create a new user
func (b *jenkinsBackend) createUser(ctx context.Context, s logical.Storage, userConfig jenkinsUser) (*jenkinsUser, error) {
	client, err := b.getClient(ctx, s)
	if err != nil {
		return nil, err
	}

	var user *jenkinsUser

	user, err = createUser(ctx, client, userConfig.Username, userConfig.Password, userConfig.Fullname, userConfig.Email)
	if err != nil {
		return nil, fmt.Errorf("error creating Jenkins user: %w", err)
	}

	if user == nil {
		return nil, errors.New("error creating Jenkins user")
	}

	return user, nil
}

// parseUsername gets Jenkins username from /users request path
func (b *jenkinsBackend) parseUsernameFromPath(path string) string {
	return strings.TrimPrefix(path, fmt.Sprintf("%s/", usersPrefix))
}

const (
	pathUsersHelpSyn = `
Create a Jenkins User.
`

	pathUsersHelpDesc = `
This path generates a Jenkins user
using the root user configured under the /config mount.
`

	pathUsersListHelpSyn = `
List Jenkins users.

`
	pathUsersListHelpDescription = `
List all Jenkins users created under /users mount.
`
)
