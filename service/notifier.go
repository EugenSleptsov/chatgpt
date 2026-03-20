package service

import (
	"GPTBot/api/logger"
	"GPTBot/api/telegram/adminlog"
	"GPTBot/util"
)

// Notifier combines console logging with optional admin notifications
// via a separate Telegram bot. It is NOT part of the transport layer.
type Notifier struct {
	Log             logger.Log
	AdminLog        adminlog.AdminLogger // may be nil
	IgnoreReportIDs []int64
}

// Notify logs the message to console and sends it to the admin bot (if configured).
func (n *Notifier) Notify(message string) {
	n.Log.Log(message)
	if n.AdminLog == nil {
		return
	}
	_ = n.AdminLog.Log(message)
}

// Logf logs a formatted message to the console only (no admin notification).
func (n *Notifier) Logf(format string, v ...interface{}) {
	n.Log.Logf(format, v...)
}

// LogError logs an error to the console if non-nil.
func (n *Notifier) LogError(err error) {
	if err != nil {
		n.Log.Logf("Error: %v", err)
	}
}

// ReportAdmin sends a notification unless the user is in the ignore list.
func (n *Notifier) ReportAdmin(userID int64, message string) {
	if !util.IsIdInList(userID, n.IgnoreReportIDs) {
		n.Notify(message)
	}
}
