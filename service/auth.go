package service

import (
	"GPTBot/util"
	"sync"
)

// Auth provides thread-safe user authorization and admin checks.
type Auth struct {
	mu                sync.RWMutex
	AdminID           int64
	authorizedUserIDs []int64
}

// NewAuth creates an Auth instance. Safe for concurrent use.
func NewAuth(adminID int64, authorizedUserIDs []int64) *Auth {
	cp := make([]int64, len(authorizedUserIDs))
	copy(cp, authorizedUserIDs)
	return &Auth{AdminID: adminID, authorizedUserIDs: cp}
}

// IsAuthorized returns true if the user is allowed (empty list = everyone allowed).
func (a *Auth) IsAuthorized(userID int64) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.authorizedUserIDs) == 0 || util.IsIdInList(userID, a.authorizedUserIDs)
}

// IsAdmin returns true if the user is the bot administrator.
// AdminID is set once at startup and never mutated — no lock needed.
func (a *Auth) IsAdmin(userID int64) bool {
	return a.AdminID != 0 && userID == a.AdminID
}

// SetAuthorizedUsers atomically replaces the authorized user list.
func (a *Auth) SetAuthorizedUsers(ids []int64) {
	cp := make([]int64, len(ids))
	copy(cp, ids)
	a.mu.Lock()
	defer a.mu.Unlock()
	a.authorizedUserIDs = cp
}

// GetAuthorizedUsers returns a snapshot copy of the current authorized user list.
func (a *Auth) GetAuthorizedUsers() []int64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	cp := make([]int64, len(a.authorizedUserIDs))
	copy(cp, a.authorizedUserIDs)
	return cp
}
