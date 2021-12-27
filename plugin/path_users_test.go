package jenkinssecretsengine

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
)

const (
	testUserUsername = "testUsername"
	testUserPassword = "testPassword"
	testUserFullname = "testFullname"
	testUserEmail    = "testEmail@testemail.com"
)

// TestUser mocks the creation, read operations for a Jenkins user
func TestUser(t *testing.T) {
	b, s := getTestBackend(t)
	AddTestConfig(t, b, s)

	userPath := fmt.Sprintf("%s/%s", usersPrefix, testUserUsername)

	t.Run("Test User", func(t *testing.T) {
		err := testUserCreate(t, b, s, userPath, map[string]interface{}{
			"password": testUserPassword,
			"fullname": testUserFullname,
			"email":    testUserEmail,
		})
		assert.NoError(t, err)

		// Users are ephemeral, will error if exists
		err = testUserUpdate(t, b, s, userPath, map[string]interface{}{
			"password": testUserPassword,
			"fullname": testUserFullname,
			"email":    testUserEmail,
		})
		assert.Error(t, err)

		err = testUserRead(t, b, s, userPath, map[string]interface{}{
			"username": testUserUsername,
			"fullname": testUserFullname,
			"email":    testUserEmail,
		})
		assert.NoError(t, err)

		err = testUserDelete(t, b, s, userPath)
		assert.NoError(t, err)
	})
}

func testUserDelete(t *testing.T, b logical.Backend, s logical.Storage, path string) error {
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.DeleteOperation,
		Path:      path,
		Storage:   s,
	})

	if err != nil {
		return err
	}

	if resp != nil && resp.IsError() {
		return resp.Error()
	}
	return nil
}

func testUserCreate(t *testing.T, b logical.Backend, s logical.Storage, path string, d map[string]interface{}) error {
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.CreateOperation,
		Path:      path,
		Data:      d,
		Storage:   s,
	})

	if err != nil {
		return err
	}

	if resp != nil && resp.IsError() {
		return resp.Error()
	}
	return nil
}

func testUserUpdate(t *testing.T, b logical.Backend, s logical.Storage, path string, d map[string]interface{}) error {
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      path,
		Data:      d,
		Storage:   s,
	})

	if err != nil {
		return err
	}

	if resp != nil && resp.IsError() {
		return resp.Error()
	}
	return nil
}

func testUserRead(t *testing.T, b logical.Backend, s logical.Storage, path string, expected map[string]interface{}) error {
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ReadOperation,
		Path:      path,
		Storage:   s,
	})

	if err != nil {
		return err
	}

	if resp == nil && expected == nil {
		return nil
	}

	if resp.IsError() {
		return resp.Error()
	}

	if len(expected) != len(resp.Data) {
		return fmt.Errorf("read data mismatch (expected %d values, got %d)", len(expected), len(resp.Data))
	}

	for k, expectedV := range expected {
		actualV, ok := resp.Data[k]

		if !ok {
			return fmt.Errorf(`expected data["%s"] = %v but was not included in read output"`, k, expectedV)
		} else if expectedV != actualV {
			return fmt.Errorf(`expected data["%s"] = %v, instead got %v"`, k, expectedV, actualV)
		}
	}

	return nil
}
