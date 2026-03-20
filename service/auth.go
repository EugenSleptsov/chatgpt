package service

import "GPTBot/util"

// Auth provides user authorization checks.
type Auth struct {
	AuthorizedUserIDs []int64
}

// IsAuthorized returns true if the user is allowed (empty list = everyone allowed).
func (a *Auth) IsAuthorized(userID int64) bool {
	return len(a.AuthorizedUserIDs) == 0 || util.IsIdInList(userID, a.AuthorizedUserIDs)
}
