package app

import (
	"strings"

	"github.com/martin3zra/forge/foundation"
)

var rolePermissionsCache = map[string]map[string]bool{}

// groupedPermissions maps a role to its allowed action:resource pairs. "*" means
// full access. Extend this for your application's roles and abilities.
var groupedPermissions = map[string]map[string][]string{
	"admin": {"*": {"*"}},
	"user": {
		"view": {"dashboard"},
	},
}

func permissions(role string) map[string]bool {
	if cached, ok := rolePermissionsCache[role]; ok {
		return cached
	}

	flat := map[string]bool{}
	for action, resources := range groupedPermissions[role] {
		for _, resource := range resources {
			if action == "*" && resource == "*" {
				flat["*"] = true
				continue
			}
			flat[action+":"+resource] = true
		}
	}

	rolePermissionsCache[role] = flat
	return flat
}

// Can reports whether the user's role grants the given action:resource ability.
func Can(user foundation.Authenticatable, actionModule string) bool {
	perms := permissions(user.GetRole())

	if perms["*"] {
		return true
	}
	if actionModule == "*" {
		return perms["*"]
	}
	if perms[actionModule] {
		return true
	}

	idx := strings.Index(actionModule, ":")
	if idx < 0 {
		return false
	}
	action, module := actionModule[:idx], actionModule[idx+1:]
	return perms[action+":*"] || perms["*:"+module]
}
