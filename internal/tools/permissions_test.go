package tools

import (
	"context"
	"testing"
)

type permissionTestAsker struct{ answer string }

func (a permissionTestAsker) Ask(context.Context, string, []string) (string, error) {
	return a.answer, nil
}

func TestPolicyGateReadOnlyToolAllowed(t *testing.T) {
	gate := NewPolicyGate(PermissionSafe)
	if err := gate.Check(context.Background(), PermissionRequest{Tool: "read_file"}); err != nil {
		t.Fatalf("read-only check error = %v", err)
	}
}

func TestPolicyGateDeniesWithoutInteractiveApproval(t *testing.T) {
	gate := NewPolicyGate(PermissionSafe)
	if err := gate.Check(context.Background(), PermissionRequest{Tool: "bash", Reason: "runs commands"}); err == nil {
		t.Fatal("bash without asker should be denied")
	}
}

func TestPolicyGateRequiresAndAcceptsApproval(t *testing.T) {
	gate := NewPolicyGate(PermissionSafe)
	ctx := WithAsker(context.Background(), permissionTestAsker{answer: "Allow once"})
	if err := gate.Check(ctx, PermissionRequest{Tool: "write_file", Reason: "modifies files"}); err != nil {
		t.Fatalf("approved write check error = %v", err)
	}
}
