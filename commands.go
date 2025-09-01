package main

import "fmt"

// commands holds all registered CLI command handlers
type commands struct {
	handlers map[string]func(*state, command) error
}

// register adds a command handler
func (c *commands) register(name string, f func(*state, command) error) {
	if c.handlers == nil {
		c.handlers = make(map[string]func(*state, command) error)
	}
	c.handlers[name] = f
}

// run executes a command if it exists
func (c *commands) run(s *state, cmd command) error {
	if handler, ok := c.handlers[cmd.name]; ok {
		return handler(s, cmd)
	}
	return fmt.Errorf("unknown command: %s", cmd.name)
}
