package jenkinssecretsengine

import (
	"errors"

	"github.com/bndr/gojenkins"
)

// jenkinsClient creates an object storing
// the client.
type jenkinsClient struct {
	*gojenkins.Jenkins
}

// newClient creates a new client to access Jenkins
func newClient(config *jenkinsConfig) (*jenkinsClient, error) {
	if config == nil {
		return nil, errors.New("jenkins configuration was nil in /config")
	}

	if config.Username == "" {
		return nil, errors.New("jenkins username was not defined in /config")
	}

	if config.Password == "" {
		return nil, errors.New("jenkins password was not defined in /config")
	}

	if config.URL == "" {
		return nil, errors.New("jenkins URL was not defined in /config")
	}

	jenkins := gojenkins.CreateJenkins(nil, config.URL, config.Username, config.Password)

	return &jenkinsClient{jenkins}, nil
}
