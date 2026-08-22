package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type PermissionMode string

const (
	PermissionSafe           PermissionMode = "safe"
	PermissionWorkspaceWrite PermissionMode = "workspace-write"
	PermissionFull           PermissionMode = "full"
)

type PermissionRequest struct {
	Tool      string
	Arguments json.RawMessage
	Reason    string
}

type PermissionGate interface {
	Check(context.Context, PermissionRequest) error
}

// PolicyGate is a host-side approval gate. It never trusts model output as an
// approval; approval must arrive through the injected Asker channel.
type PolicyGate struct{ Mode PermissionMode }

func NewPolicyGate(mode PermissionMode) *PolicyGate { return &PolicyGate{Mode: mode} }

func (g *PolicyGate) Check(ctx context.Context, req PermissionRequest) error {
	if g == nil {
		return nil
	}
	if req.Tool == "ask_user_question" || isReadOnlyTool(req.Tool) {
		return nil
	}
	if g.Mode == PermissionFull {
		return nil
	}
	if g.Mode == PermissionWorkspaceWrite {
		switch req.Tool {
		case "write_file", "edit_file", "todo_write":
			return nil
		}
	}
	asker, ok := AskerFromContext(ctx)
	if !ok {
		return fmt.Errorf("permission denied: %s requires user approval but no interactive channel is available", req.Tool)
	}
	question := fmt.Sprintf("Allow tool %q? %s", req.Tool, strings.TrimSpace(req.Reason))
	answer, err := asker.Ask(ctx, question, []string{"Allow once", "Deny"})
	if err != nil {
		return fmt.Errorf("permission approval: %w", err)
	}
	if strings.EqualFold(strings.TrimSpace(answer), "allow once") || strings.EqualFold(strings.TrimSpace(answer), "allow") {
		return nil
	}
	return fmt.Errorf("permission denied by user: %s", req.Tool)
}

func isReadOnlyTool(name string) bool {
	switch name {
	case "read_file", "list_dir", "glob", "grep":
		return true
	default:
		return false
	}
}
