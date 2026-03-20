package service

import "GPTBot/util"

// Auth provides user authorization and admin checks.
type Auth struct {
	AdminID           int64
	AuthorizedUserIDs []int64
}

// IsAuthorized returns true if the user is allowed (empty list = everyone allowed).
func (a *Auth) IsAuthorized(userID int64) bool {
	return len(a.AuthorizedUserIDs) == 0 || util.IsIdInList(userID, a.AuthorizedUserIDs)
}

// IsAdmin returns true if the user is the bot administrator.
func (a *Auth) IsAdmin(userID int64) bool {
	return a.AdminID != 0 && userID == a.AdminID
}
