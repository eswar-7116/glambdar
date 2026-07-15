package auth

var rolePermissions = map[Role]map[string][]string{
	RoleAdmin: {
		"deploy": {"POST"},
		"invoke": {"GET", "POST", "PUT", "PATCH", "DELETE"},
		"delete": {"DELETE"},
		"config": {"POST"},
		"info":   {"GET"},
		"logs":   {"GET"},
		"auth":   {"GET", "POST", "PATCH", "DELETE"},
	},
	RoleDeployer: {
		"deploy": {"POST"},
		"invoke": {"GET", "POST", "PUT", "PATCH", "DELETE"},
		"delete": {"DELETE"},
		"config": {"POST"},
		"info":   {"GET"},
		"logs":   {"GET"},
	},
	RoleInvoker: {
		"invoke": {"GET", "POST", "PUT", "PATCH", "DELETE"},
		"info":   {"GET"},
		"logs":   {"GET"},
	},
	RoleViewer: {
		"info": {"GET"},
		"logs": {"GET"},
	},
}

// HasPermission checks if a role can perform an action with a specific HTTP method
func HasPermission(role Role, action string, method string) bool {
	actions, ok := rolePermissions[role]
	if !ok {
		return false
	}
	methods, ok := actions[action]
	if !ok {
		return false
	}
	for _, m := range methods {
		if m == method {
			return true
		}
	}
	return false
}
