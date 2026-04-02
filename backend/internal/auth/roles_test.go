package auth

import "testing"

func TestIsAdminRole(t *testing.T) {
	if !IsAdminRole("admin") {
		t.Fatal("expected admin to be recognized as admin role")
	}
	if IsAdminRole("trustee") {
		t.Fatal("did not expect trustee to be recognized as admin role")
	}
}

func TestIsInvitableRole(t *testing.T) {
	for _, role := range []string{"trustee", "resident"} {
		if !IsInvitableRole(role) {
			t.Fatalf("expected %q to be invitable", role)
		}
	}
	for _, role := range []string{"admin", "agent", "owner", ""} {
		if IsInvitableRole(role) {
			t.Fatalf("did not expect %q to be invitable", role)
		}
	}
}
