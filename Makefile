GOARCH = amd64

UNAME = $(shell uname -s)

ifndef OS
	ifeq ($(UNAME), Linux)
		OS = linux
	else ifeq ($(UNAME), Darwin)
		OS = darwin
	endif
endif

.DEFAULT_GOAL := all

all: fmt build start

build:
	GOOS="$(OS)" GOARCH="$(GOARCH)" go build -o vault/plugins/vault-plugin-secrets-jenkins
	chmod 755 vault/plugins/*

start:
	vault server -dev -dev-root-token-id=root -dev-plugin-dir=./vault/plugins

enable:
	vault secrets enable -path=jenkins vault-plugin-secrets-jenkins

clean:
	rm -f ./vault/plugins/vault-plugin-secrets-jenkins

fmt:
	go fmt $$(go list ./...)

lint:
	golangci-lint run -v

jenkins:
	docker rm -f vault-jenkins
	docker build -t vault-jenkins -f Dockerfile.jenkins .
	docker run --name vault-jenkins -d --rm -p 8080:8080 vault-jenkins

test: jenkins
	sleep 15
	go test -v ./...

set-vault-var:
	export VAULT_ADDR="http://localhost:8200"

enable-plugin: build
	vault secrets enable vault-plugin-secrets-jenkins || exit 0
	vault write sys/plugins/catalog/jenkins \
      sha_256="$$(shasum -a 256 ./vault/plugins/vault-plugin-secrets-jenkins | cut -d " " -f1)" \
      command="vault-plugin-secrets-jenkins"
	vault write vault-plugin-secrets-jenkins/config url=http://localhost:8080 username=admin password=admin

token: set-vault-var enable-plugin
	vault read vault-plugin-secrets-jenkins/tokens/mytoken ttl=30

user: set-vault-var enable-plugin
	vault write vault-plugin-secrets-jenkins/users/myuser ttl=45 password=testpass fullname=fullname email=email@email.com

.PHONY: build clean fmt start enable