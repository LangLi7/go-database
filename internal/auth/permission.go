package auth

// Permission constants matching PROJEKT.md
const (
	PermConnectionsList   = "connections:list"
	PermConnectionsCreate = "connections:create"
	PermConnectionsDelete = "connections:delete"
	PermConnectionsQuery  = "connections:query"
	PermConnectionsExec   = "connections:execute"
	PermUsersList         = "users:list"
	PermUsersCreate       = "users:create"
	PermUsersEdit         = "users:edit"
	PermUsersDelete       = "users:delete"
	PermSettingsRead      = "settings:read"
	PermSettingsWrite     = "settings:write"
	PermJobsList          = "jobs:list"
	PermJobsCreate        = "jobs:create"
	PermJobsCancel        = "jobs:cancel"
	PermBackupCreate      = "backup:create"
	PermBackupRestore     = "backup:restore"
	PermTrafficView       = "traffic:view"
	PermAPIKeysManage     = "apikeys:manage"
	PermRolesManage       = "roles:manage"
	PermAdmin             = "*"
)

// Role defines a named set of permissions
type Role struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
	DBAccess    []string `json:"db_access"` // connection IDs this role can access
}

// User holds authentication and authorization data
type User struct {
	ID            string   `json:"id"`
	Username      string   `json:"username"`
	PasswordHash  string   `json:"-"`
	Role          string   `json:"role"`            // role ID
	ExtraPerm     []string `json:"extra_perm,omitempty"`  // user-specific overrides
	ExtraDBAccess []string `json:"extra_db_access,omitempty"`
}

// DefaultRoles returns the three built-in roles
func DefaultRoles() []Role {
	return []Role{
		{
			ID:   "admin",
			Name: "Administrator",
			Permissions: []string{PermAdmin},
		},
		{
			ID:   "developer",
			Name: "Developer",
			Permissions: []string{
				PermConnectionsList, PermConnectionsQuery, PermConnectionsExec,
				PermUsersList, PermJobsList, PermJobsCreate,
				PermTrafficView,
			},
		},
		{
			ID:   "readonly",
			Name: "Read Only",
			Permissions: []string{
				PermConnectionsList, PermConnectionsQuery,
			},
		},
	}
}

// RolePermissions returns the effective permissions for a user
func GetEffectivePerms(rolePerms []string, extraPerms []string) []string {
	// If role has admin (*), user gets everything
	for _, p := range rolePerms {
		if p == PermAdmin {
			return []string{PermAdmin}
		}
	}

	// Combine role permissions + user-specific overrides
	seen := make(map[string]bool)
	result := make([]string, 0)
	for _, p := range append(rolePerms, extraPerms...) {
		if !seen[p] {
			seen[p] = true
			result = append(result, p)
		}
	}
	return result
}

// HasPermission checks if a set of permissions includes a specific one
func HasPermission(perms []string, required string) bool {
	for _, p := range perms {
		if p == PermAdmin || p == required {
			return true
		}
	}
	return false
}
