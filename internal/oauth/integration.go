// Package oauth defines provider-neutral OAuth integration contracts.
package oauth

import "context"

type Authorization struct {
	URL          string
	State        string
	CallbackPath string
}

type Integration interface {
	ID() string
	Name() string
	Connected() bool
	BeginAuthorization() (Authorization, error)
	CompleteAuthorization(context.Context, string) error
	Disconnect() error
}
