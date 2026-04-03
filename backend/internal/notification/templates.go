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
