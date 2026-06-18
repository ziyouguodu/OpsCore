package auth

const (
	RoleSuperAdmin  = "super_admin"
	RoleOpsEngineer = "ops_engineer"
)

const (
	PermissionAssetRead            = "asset:read"
	PermissionAssetWrite           = "asset:write"
	PermissionAssetCredential      = "asset:credential:read"
	PermissionAssetCredentialWrite = "asset:credential:write"
	PermissionOnCallRead           = "oncall:read"
	PermissionOnCallWrite          = "oncall:write"
	PermissionTaskRead             = "task:read"
	PermissionTaskWrite            = "task:write"
	PermissionIncidentRead         = "incident:read"
	PermissionIncidentFollowup     = "incident:followup"
	PermissionUserManage           = "user:manage"
)

var rolePermissions = map[string]map[string]bool{
	RoleSuperAdmin: {
		"*": true,
	},
	RoleOpsEngineer: {
		PermissionAssetRead:        true,
		PermissionAssetWrite:       true,
		PermissionOnCallRead:       true,
		PermissionTaskRead:         true,
		PermissionTaskWrite:        true,
		PermissionIncidentRead:     true,
		PermissionIncidentFollowup: true,
	},
}

func HasPermission(roles []string, permission string) bool {
	for _, role := range roles {
		perms := rolePermissions[role]
		if perms["*"] || perms[permission] {
			return true
		}
	}
	return false
}

func HasRole(roles []string, expected string) bool {
	for _, role := range roles {
		if role == expected {
			return true
		}
	}
	return false
}
