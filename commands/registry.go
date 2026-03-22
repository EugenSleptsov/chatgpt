package commands

import "fmt"

// Registry stores bot commands by name.
// Use Add() to register and Get()/All() to look up.
type Registry struct {
	commands map[string]Command
}

func NewRegistry() *Registry {
	return &Registry{commands: make(map[string]Command)}
}

// Add registers a command. Panics on duplicate names (programming error).
func (r *Registry) Add(cmd Command) {
	if _, exists := r.commands[cmd.Name()]; exists {
		panic(fmt.Sprintf("duplicate command: %s", cmd.Name()))
	}
	r.commands[cmd.Name()] = cmd
}

// Get looks up a command by name.
func (r *Registry) Get(name string) (Command, error) {
	if cmd, ok := r.commands[name]; ok {
		return cmd, nil
	}
	return nil, fmt.Errorf("command %s not found", name)
}

// All returns every registered command (order is not guaranteed).
func (r *Registry) All() []Command {
	out := make([]Command, 0, len(r.commands))
	for _, cmd := range r.commands {
		out = append(out, cmd)
	}
	return out
}
