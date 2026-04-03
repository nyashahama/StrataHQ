// backend/internal/notification/templates.go
package notification

import "fmt"

func InvitationEmail(name, inviteURL string) (subject, htmlBody string) {
	subject = "You've been invited to StrataHQ"
	htmlBody = fmt.Sprintf(`<!DOCTYPE html>
<html><body style="font-family:sans-serif;max-width:600px;margin:0 auto;padding:24px">
<h2>Hi %s,</h2>
<p>You've been invited to manage a scheme on StrataHQ.</p>
<p style="margin:32px 0">
  <a href="%s" style="background:#18181b;color:#fff;padding:12px 24px;border-radius:8px;text-decoration:none;font-weight:600">
    Accept invitation
  </a>
</p>
<p style="color:#71717a;font-size:13px">This link expires in 7 days. If you didn't expect this email, you can safely ignore it.</p>
</body></html>`, name, inviteURL)
	return
}

func PasswordResetEmail(resetURL string) (subject, htmlBody string) {
	subject = "Reset your StrataHQ password"
	htmlBody = fmt.Sprintf(`<!DOCTYPE html>
<html><body style="font-family:sans-serif;max-width:600px;margin:0 auto;padding:24px">
<h2>Password reset</h2>
<p>Click the button below to reset your StrataHQ password. This link expires in 1 hour.</p>
<p style="margin:32px 0">
  <a href="%s" style="background:#18181b;color:#fff;padding:12px 24px;border-radius:8px;text-decoration:none;font-weight:600">
    Reset password
  </a>
</p>
<p style="color:#71717a;font-size:13px">If you didn't request a password reset, you can safely ignore this email.</p>
</body></html>`, resetURL)
	return
}

func EarlyAccessApprovalEmail(name, setPasswordURL string) (subject, htmlBody string) {
	subject = "Your StrataHQ access has been approved"
	htmlBody = fmt.Sprintf(`<!DOCTYPE html>
<html><body style="font-family:sans-serif;max-width:600px;margin:0 auto;padding:24px">
<h2>Hi %s,</h2>
<p>Great news — your early access request for StrataHQ has been approved.</p>
<p>Click the button below to set your password and get started:</p>
<p style="margin:32px 0">
  <a href="%s" style="background:#18181b;color:#fff;padding:12px 24px;border-radius:8px;text-decoration:none;font-weight:600">
    Set your password
  </a>
</p>
<p style="color:#71717a;font-size:13px">This link expires in 24 hours. If you didn't request access, you can safely ignore this email.</p>
</body></html>`, name, setPasswordURL)
	return
}

func NewEarlyAccessRequestEmail(requesterName, requesterEmail, schemeName string, unitCount int32, approveURL, rejectURL string) (subject, htmlBody string) {
	subject = fmt.Sprintf("New early access request — %s (%s)", requesterName, schemeName)
	htmlBody = fmt.Sprintf(`<!DOCTYPE html>
<html><body style="font-family:sans-serif;max-width:600px;margin:0 auto;padding:24px">
<h2>New early access request</h2>
<table style="border-collapse:collapse;width:100%%;margin-bottom:24px">
  <tr><td style="padding:8px 0;color:#71717a;width:120px">Name</td><td style="padding:8px 0;font-weight:600">%s</td></tr>
  <tr><td style="padding:8px 0;color:#71717a">Email</td><td style="padding:8px 0">%s</td></tr>
  <tr><td style="padding:8px 0;color:#71717a">Scheme</td><td style="padding:8px 0">%s</td></tr>
  <tr><td style="padding:8px 0;color:#71717a">Units</td><td style="padding:8px 0">%d</td></tr>
</table>
<div style="display:flex;gap:12px;margin-bottom:24px">
  <a href="%s" style="background:#18181b;color:#fff;padding:12px 24px;border-radius:8px;text-decoration:none;font-weight:600;margin-right:12px">
    Approve
  </a>
  <a href="%s" style="background:#f4f4f5;color:#18181b;padding:12px 24px;border-radius:8px;text-decoration:none;font-weight:600">
    Reject
  </a>
</div>
<p style="color:#71717a;font-size:13px">These links expire in 7 days.</p>
</body></html>`, requesterName, requesterEmail, schemeName, unitCount, approveURL, rejectURL)
	return
}
