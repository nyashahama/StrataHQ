// backend/internal/notification/noop.go
package notification

import "context"

// NoopSender captures calls without hitting the network. Use in tests.
type NoopSender struct {
	InvitationsSent []string
	PasswordResets  []string
}

func (n *NoopSender) SendInvitation(_ context.Context, to, _, _ string) error {
	n.InvitationsSent = append(n.InvitationsSent, to)
	return nil
}

func (n *NoopSender) SendPasswordReset(_ context.Context, to, _ string) error {
	n.PasswordResets = append(n.PasswordResets, to)
	return nil
}

func (n *NoopSender) SendEarlyAccessApproval(_ context.Context, to, _, _ string) error {
	n.InvitationsSent = append(n.InvitationsSent, to)
	return nil
}

func (n *NoopSender) SendNewEarlyAccessRequest(_ context.Context, _, _, _, _ string, _ int32, _, _ string) error {
	return nil
}
