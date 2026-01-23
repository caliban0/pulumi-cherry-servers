package provider

import (
	"context"
	p "github.com/pulumi/pulumi-go-provider"
)

type Logger interface {
	Warningf(msg string, a ...any)
}

type PulumiLogger struct {
	p.Logger
}

var _ Logger = (*PulumiLogger)(nil)

type GetLoggerFunc func(context.Context) Logger

func GetLogger(ctx context.Context) Logger {
	return PulumiLogger{Logger: p.GetLogger(ctx)}
}
