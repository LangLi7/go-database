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

// Role defines a named set of permissions. A role may inherit from a parent
// role (Luckperms-style) — effective perms/db_access accumulate up the chain.
type Role struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Parent      string   `json:"parent,omitempty"` // parent role ID to inherit from
	Permissions []string `json:"permissions"`
	DBAccess    []string `json:"db_access"` // connection IDs this role can access
}

// User represents an account in the system.
type User struct {
	ID            string   `json:"id"`
	Username      string   `json:"username"`
	PasswordHash  string   `json:"-"`
	Role          string   `json:"role"`
	ExtraPerm     []string `json:"extra_perm,omitempty"`
	ExtraDBAccess []string `json:"extra_db_access,omitempty"`
	Email         string   `json:"email,omitempty"`
	PublicKey     string   `json:"public_key,omitempty"` // ssh-ed25519 ... for passwordless login
}

// DefaultRoles returns the three built-in roles
func DefaultRoles() []Role {
	return []Role{
		{
			ID:          "admin",
			Name:        "Administrator",
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

// GetEffectivePerms returns the effective permissions for a user given a role
// name, an optional role loader (for parent inheritance), and extra perms.
// Parent roles' permissions accumulate (Luckperms-style). A role with admin (*)
// grants everything.
func GetEffectivePerms(roleName string, loadRole func(id string) (*Role, bool), extraPerms []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	var walk func(id string)
	walk = func(id string) {
		var r *Role
		if loadRole != nil {
			if loaded, ok := loadRole(id); ok {
				r = loaded
			}
		}
		if r == nil {
			// built-in?
			for _, b := range DefaultRoles() {
				if b.ID == id {
					r = &b
					break
				}
			}
		}
		if r == nil {
			return
		}
		// admin short-circuit
		for _, p := range r.Permissions {
			if p == PermAdmin {
				result = []string{PermAdmin}
				return
			}
		}
		// prepend so parent perms come first; avoid dupes
		for _, p := range r.Permissions {
			if !seen[p] {
				seen[p] = true
				result = append(result, p)
			}
		}
		if r.Parent != "" {
			walk(r.Parent)
		}
	}
	walk(roleName)

	// merge extra perms last
	for _, p := range extraPerms {
		if !seen[p] {
			seen[p] = true
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return []string{}
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
