package jenkinssecretsengine

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	b := backend()
	if err := b.Setup(ctx, conf); err != nil {
		return nil, err
	}
	return b, nil
}

// jenkinsBackend defines an object that
// extends the Vault backend and stores the
// target API's client.
type jenkinsBackend struct {
	*framework.Backend
	client *jenkinsClient
	lock   sync.RWMutex
}

// backend defines the target API backend
// for Vault. It must include each path
// and the secrets it will store.
func backend() *jenkinsBackend {
	var b = jenkinsBackend{}

	b.Backend = &framework.Backend{
		Help: strings.TrimSpace(backendHelp),
		PathsSpecial: &logical.Paths{
			LocalStorage: []string{},
			SealWrapStorage: []string{
				configPrefix,
				fmt.Sprintf("%s/*", usersPrefix),
				fmt.Sprintf("%s/*", tokensPrefix),
			},
		},
		Paths: framework.PathAppend(
			[]*framework.Path{
				pathConfig(&b),
			},
			pathTokens(&b),
			pathUsers(&b),
		),
		Secrets: []*framework.Secret{
			b.jenkinsUser(),
			b.jenkinsToken(),
		},
		BackendType: logical.TypeLogical,
		Invalidate:  b.invalidate,
	}
	return &b
}

// reset clears any client configuration for a new
// backend to be configured
func (b *jenkinsBackend) reset() {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.client = nil
}

// invalidate clears an existing client configuration in
// the backend
func (b *jenkinsBackend) invalidate(ctx context.Context, key string) {
	if key == configPrefix {
		b.reset()
	}
}

// getClient locks the backend as it configures and creates a
// a new client for the Jenkins API
func (b *jenkinsBackend) getClient(ctx context.Context, s logical.Storage) (*jenkinsClient, error) {
	b.lock.RLock()
	unlockFunc := b.lock.RUnlock
	defer func() { unlockFunc() }()

	if b.client != nil {
		return b.client, nil
	}

	b.lock.RUnlock()
	b.lock.Lock()
	unlockFunc = b.lock.Unlock

	config, err := getConfig(ctx, s)
	if err != nil {
		return nil, err
	}

	if config == nil {
		config = new(jenkinsConfig)
	}

	b.client, err = newClient(config)
	if err != nil {
		return nil, err
	}

	return b.client, nil
}

// backendHelp should contain help information for the backend
const backendHelp = `
The Jenkins secrets backend dynamically generates user tokens.
After mounting this backend, credentials to manage Jenkins user tokens
must be configured with the "config/" endpoints.
`
