package auth

import (
	"strings"
)

// Resource permission format: "action:resource" with wildcard support
// Examples:
//
//	"query:conn_sales"          — query connection conn_sales
//	"query:conn_sales.customers" — query customers table in conn_sales
//	"query:*"                    — query any connection
//	"list:connection.*"          — list all connections
//	"-query:conn_sales"          — deny query on conn_sales (always wins)
//	"connections:query"          — flat perm (mapped to "query:connection.*")

// flatPermToResource maps old flat permissions to resource-based format
var flatPermToResource = map[string]string{
	PermConnectionsList:   "list:connection.*",
	PermConnectionsCreate: "create:connection.*",
	PermConnectionsDelete: "delete:connection.*",
	PermConnectionsQuery:  "query:connection.*",
	PermConnectionsExec:   "execute:connection.*",
	PermUsersList:         "list:users.*",
	PermUsersCreate:       "create:users.*",
	PermUsersEdit:         "edit:users.*",
	PermUsersDelete:       "delete:users.*",
	PermSettingsRead:      "read:settings.*",
	PermSettingsWrite:     "write:settings.*",
	PermJobsList:          "list:jobs.*",
	PermJobsCreate:        "create:jobs.*",
	PermJobsCancel:        "cancel:jobs.*",
	PermBackupCreate:      "create:backup.*",
	PermBackupRestore:     "restore:backup.*",
	PermTrafficView:       "read:traffic.*",
	PermAPIKeysManage:     "manage:apikeys.*",
	PermRolesManage:       "manage:roles.*",
}

// HasResourcePermission checks if permissions allow an action on a resource.
// Supports deny (prefix "-") and wildcard ("*") matching.
func HasResourcePermission(perms []string, action, resource string) bool {
	granted := false
	for _, p := range perms {
		if p == PermAdmin {
			return true
		}

		deny := false
		perm := p
		if strings.HasPrefix(p, "-") {
			deny = true
			perm = p[1:]
		}

		if matchesResource(perm, action, resource) {
			if deny {
				return false // deny always wins
			}
			granted = true
		}
	}
	return granted
}

// matchesResource checks if a permission string matches an action:resource pair
func matchesResource(perm, action, resource string) bool {
	// Try flat permission mapping first
	if mapped, ok := flatPermToResource[perm]; ok {
		perm = mapped
	}

	parts := strings.SplitN(perm, ":", 2)
	if len(parts) != 2 {
		return perm == PermAdmin
	}

	permAction := parts[0]
	permResource := parts[1]

	// Check action match
	if permAction != "*" && permAction != action {
		// Try reverse flat mapping (action:resource.* → flat perm check)
		if action == "" {
			return false
		}
		return false
	}

	// Check resource match with wildcard
	return matchPattern(permResource, resource)
}

// matchPattern checks if a pattern matches a resource string.
// Pattern supports: "*" (match all), "prefix.*" (prefix match)
func matchPattern(pattern, resource string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, ".*") {
		prefix := pattern[:len(pattern)-2]
		return strings.HasPrefix(resource, prefix)
	}
	return pattern == resource
}

// CheckDBAccess checks if a user can access a specific connection.
// Returns true if the user has global connection:* permission or the connection is in their DBAccess list.
func CheckDBAccess(perms []string, dbAccess []string, connID string) bool {
	// Admin can access everything
	for _, p := range perms {
		if p == PermAdmin {
			return true
		}
	}

	// Global connection permissions grant access to all connections
	for _, p := range perms {
		if p == PermConnectionsList || p == PermConnectionsQuery || p == PermConnectionsExec {
			return true
		}
		if mapped, ok := flatPermToResource[p]; ok {
			if strings.HasPrefix(mapped, "list:connection.") ||
				strings.HasPrefix(mapped, "query:connection.") ||
				strings.HasPrefix(mapped, "execute:connection.") {
				return true
			}
		}
	}

	// Check explicit deny in DBAccess
	for _, d := range dbAccess {
		if d == "-"+connID {
			return false
		}
	}

	// Check if connection ID is in DBAccess list
	for _, d := range dbAccess {
		if d == connID || d == "*" {
			return true
		}
	}

	return false
}

// GetEffectiveDBAccess accumulates db_access up the role parent chain and
// merges with the user's extra_db_access. Extra db_access with "-" prefix
// denies access (always wins). parent roles' db_access are resolved via loadRole.
func GetEffectiveDBAccess(roleName string, loadRole func(id string) (*Role, bool), extraDBAccess []string) []string {
	// accumulate role db_access (child + parents)
	var roleDBAccess []string
	var walk func(id string)
	walk = func(id string) {
		var r *Role
		if loadRole != nil {
			if loaded, ok := loadRole(id); ok {
				r = loaded
			}
		}
		if r == nil {
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
		roleDBAccess = append(roleDBAccess, r.DBAccess...)
		if r.Parent != "" {
			walk(r.Parent)
		}
	}
	walk(roleName)

	seen := make(map[string]bool)
	var result []string

	// Add role db_access
	for _, d := range roleDBAccess {
		if !seen[d] {
			seen[d] = true
			result = append(result, d)
		}
	}

	// Add/override with extra db_access
	for _, d := range extraDBAccess {
		if d == "" {
			continue
		}
		if strings.HasPrefix(d, "-") {
			// Deny: add deny entry and remove any allow for same conn
			connID := d[1:]
			for i, existing := range result {
				if existing == connID {
					result = append(result[:i], result[i+1:]...)
					break
				}
			}
			if !seen[d] {
				seen[d] = true
				result = append(result, d)
			}
		} else {
			// Allow: remove any deny for this conn
			denyKey := "-" + d
			for i, existing := range result {
				if existing == denyKey {
					result = append(result[:i], result[i+1:]...)
					break
				}
			}
			if !seen[d] {
				seen[d] = true
				result = append(result, d)
			}
		}
	}

	return result
}
