package compose

import (
	"context"
	"io"
)

type Executor interface {
	CreateSandbox(ctx context.Context, name string, spec *ResolvedSpec) error
	UpdatePolicy(ctx context.Context, name string, spec *ResolvedSpec) error
	ExecInSandbox(ctx context.Context, name string, cmd []string) error
	ConnectSandbox(ctx context.Context, name string) error
	DeleteSandbox(ctx context.Context, name string) error
	SandboxLogs(ctx context.Context, name string) (io.ReadCloser, error)
	SandboxStatus(ctx context.Context, name string) (SandboxState, error)
	ListSandboxes(ctx context.Context, labelSelector string) ([]string, error)
}
