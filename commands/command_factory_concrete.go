package commands

import "fmt"

type ConcreteCommandFactory struct {
	registry map[string]func() Command
}

func NewCommandFactory() *ConcreteCommandFactory {
	return &ConcreteCommandFactory{
		registry: make(map[string]func() Command),
	}
}

func (f *ConcreteCommandFactory) GetCommands() []Command {
	var commands []Command
	for _, constructor := range f.registry {
		commands = append(commands, constructor())
	}
	return commands
}

func (f *ConcreteCommandFactory) Register(name string, constructor func() Command) {
	f.registry[name] = constructor
}

func (f *ConcreteCommandFactory) GetCommand(name string) (Command, error) {
	if constructor, ok := f.registry[name]; ok {
		return constructor(), nil
	}
	return nil, fmt.Errorf("command " + name + " not found")
}
