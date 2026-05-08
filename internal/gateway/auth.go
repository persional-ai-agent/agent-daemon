package gateway

import "strings"

func CheckAuthorization(allowedUsers, userID string) bool {
	if strings.TrimSpace(allowedUsers) == "" {
		return false
	}
	for _, uid := range strings.Split(allowedUsers, ",") {
		if strings.TrimSpace(uid) == userID {
			return true
		}
	}
	return false
}
