package provider_test

import (
	"context"

	"github.com/caliban0/pulumi-cherry-servers/provider"
)

type FakeLogger struct {
}

func (l FakeLogger) Warningf(msg string, a ...any) {
	// do nothing
}

var _ provider.Logger = (*FakeLogger)(nil)

func GetFakeLogger(_ context.Context) provider.Logger {
	return FakeLogger{}
}