package auth

type PermissionGroup struct {
	Name        string            `json:"name"`
	DisplayName string            `json:"display_name"`
	Icon        string            `json:"icon"`
	Children    []PermissionEntry `json:"children"`
}

type PermissionEntry struct {
	Key         string `json:"key"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
}

func PermissionGroups() []PermissionGroup {
	return []PermissionGroup{
		{
			Name: "connection", DisplayName: "Connection", Icon: "🔌",
			Children: []PermissionEntry{
				{PermConnectionsList, "List Connections", "View list of database connections"},
				{PermConnectionsCreate, "Create Connections", "Add new database connections"},
				{PermConnectionsDelete, "Delete Connections", "Remove database connections"},
				{PermConnectionsQuery, "Query Data", "Run SELECT queries (all connections)"},
				{PermConnectionsExec, "Execute", "Run INSERT/UPDATE/DELETE (all connections)"},
				{"query:connection.*", "Query (per-connection)", "Query specific connections via granular permissions"},
				{"execute:connection.*", "Execute (per-connection)", "Execute on specific connections"},
				{"list:connection.*", "List (per-connection)", "View specific connections"},
			},
		},
		{
			Name: "admin", DisplayName: "Administration", Icon: "⚙️",
			Children: []PermissionEntry{
				{PermUsersList, "List Users", "View user list"},
				{PermUsersCreate, "Create Users", "Add new users"},
				{PermUsersEdit, "Edit Users", "Modify existing users"},
				{PermUsersDelete, "Delete Users", "Remove users"},
				{PermRolesManage, "Manage Roles", "Create/edit/delete roles"},
				{PermTrafficView, "View Traffic", "View API traffic stats"},
				{PermSettingsRead, "Read Settings", "View system settings"},
				{PermSettingsWrite, "Write Settings", "Modify system settings"},
				{PermAPIKeysManage, "Manage API Keys", "Create/revoke API keys"},
			},
		},
		{
			Name: "jobs", DisplayName: "Jobs", Icon: "📋",
			Children: []PermissionEntry{
				{PermJobsList, "List Jobs", "View background jobs"},
				{PermJobsCreate, "Create Jobs", "Start new jobs"},
				{PermJobsCancel, "Cancel Jobs", "Stop running jobs"},
			},
		},
		{
			Name: "backup", DisplayName: "Backup", Icon: "💾",
			Children: []PermissionEntry{
				{PermBackupCreate, "Create Backups", "Create database backups"},
				{PermBackupRestore, "Restore Backups", "Restore from backups"},
			},
		},
	}
}

func AllPermissionKeys() []string {
	var keys []string
	for _, group := range PermissionGroups() {
		for _, entry := range group.Children {
			keys = append(keys, entry.Key)
		}
	}
	return keys
}
