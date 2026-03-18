package commands

type CommandRegistry interface {
	Register(name string, constructor func() Command)
	GetCommand(name string) (Command, error)
	GetCommands() []Command
}
