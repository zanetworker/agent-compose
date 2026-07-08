package compose

import (
	"context"
	"testing"
)

func TestCLIExecutor_BinaryNotFound(t *testing.T) {
	ex := NewCLIExecutor("/nonexistent/openshell")
	spec := &ResolvedSpec{Image: "test:latest"}

	err := ex.CreateSandbox(context.Background(), "test", spec)
	if err == nil {
		t.Fatal("expected error for missing binary")
	}
}
