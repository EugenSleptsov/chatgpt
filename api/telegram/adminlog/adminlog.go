package adminlog

type AdminLogger interface {
	Log(message string) error
}
