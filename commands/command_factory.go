package commands

type CommandFactory interface {
	GetCommand(name string) (Command, error)
}
