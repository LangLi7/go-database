package auth

import (
	"testing"
)

// roleLoader returns a role from a static map (simulates DB lookup)
func mapLoader(roles map[string]Role) func(id string) (*Role, bool) {
	return func(id string) (*Role, bool) {
		r, ok := roles[id]
		if !ok {
			return nil, false
		}
		return &r, true
	}
}

func TestLuckpermsInheritance(t *testing.T) {
	roles := map[string]Role{
		"super":  {ID: "super", Permissions: []string{PermConnectionsExec}, DBAccess: []string{"db_super"}},
		"dev":    {ID: "dev", Parent: "super", Permissions: []string{PermConnectionsQuery}, DBAccess: []string{"db_dev"}},
		"junior": {ID: "junior", Parent: "dev", Permissions: []string{PermConnectionsList}, DBAccess: []string{"db_junior"}},
	}
	loader := mapLoader(roles)

	// junior inherits from dev <- super
	perms := GetEffectivePerms("junior", loader, nil)
	if !HasPermission(perms, PermConnectionsExec) {
		t.Fatalf("junior should inherit exec from super via dev, got %v", perms)
	}
	if !HasPermission(perms, PermConnectionsList) {
		t.Fatalf("junior should have own list perm")
	}

	db := GetEffectiveDBAccess("junior", loader, nil)
	want := map[string]bool{"db_super": true, "db_dev": true, "db_junior": true}
	for _, d := range db {
		if !want[d] {
			t.Fatalf("unexpected db_access %s in %v", d, db)
		}
	}
	if len(db) != 3 {
		t.Fatalf("expected 3 inherited db_access, got %v", db)
	}
}

func TestDBAccessDenyWins(t *testing.T) {
	roles := map[string]Role{
		"dev": {ID: "dev", Permissions: []string{PermConnectionsQuery}, DBAccess: []string{"db_shared"}},
	}
	loader := mapLoader(roles)
	// extra deny on db_shared overrides role grant
	db := GetEffectiveDBAccess("dev", loader, []string{"-db_shared"})
	for _, d := range db {
		if d == "db_shared" {
			t.Fatalf("deny should remove db_shared, got %v", db)
		}
	}
}
