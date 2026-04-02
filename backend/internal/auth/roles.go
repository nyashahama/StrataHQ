package auth

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleAgent    Role = "agent"
	RoleTrustee  Role = "trustee"
	RoleResident Role = "resident"
	RoleOwner    Role = "owner"
)

func IsAdminRole(role string) bool {
	return role == string(RoleAdmin)
}

func IsResidentRole(role string) bool {
	return role == string(RoleResident)
}

func IsInvitableRole(role string) bool {
	return role == string(RoleTrustee) || role == string(RoleResident)
}

func IsSchemeRole(role string) bool {
	switch role {
	case string(RoleOwner), string(RoleTrustee), string(RoleResident), string(RoleAdmin):
		return true
	default:
		return false
	}
}
