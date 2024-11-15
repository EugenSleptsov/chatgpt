package commands

type CommandFactory interface {
	GetCommand(name string) (Command, error)
	Register(name string, constructor func() Command)
}
