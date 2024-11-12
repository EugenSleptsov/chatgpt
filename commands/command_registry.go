package commands

type CommandRegistry interface {
	GetCommands() []Command
}
