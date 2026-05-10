package tools

import (
	"context"
	"fmt"
	"smithai/src/agent/availability"
	"smithai/src/agent/protocol"
)


// Dispatcher manages available tools and handles executing them.
type Dispatcher interface {
	// Dispatch executes a tool by name with the given arguments.
	Dispatch(ctx context.Context, call protocol.ToolCall) (string, error)
	// Definitions returns the list of available tools.
	Definitions() []protocol.ToolDef
	// Register adds a tool to the dispatcher.
	Register(def protocol.ToolDef, handler Handler)
}

// Handler is a function that executes a tool.
type Handler func(ctx context.Context, args any) (string, error)

// BasicDispatcher implements Dispatcher.
type BasicDispatcher struct {
	tools    map[string]protocol.ToolDef
	handlers map[string]Handler
}

// NewBasicDispatcher creates a new dispatcher.
func NewBasicDispatcher() *BasicDispatcher {
	return &BasicDispatcher{
		tools:    make(map[string]protocol.ToolDef),
		handlers: make(map[string]Handler),
	}
}

// Register adds a tool to the dispatcher.
func (d *BasicDispatcher) Register(def protocol.ToolDef, handler Handler) {
	d.tools[def.Name] = def
	d.handlers[def.Name] = handler
	// Mark as available when registered
	availability.MarkAvailable(def.Name, "tool", "registered")
}

// Dispatch executes a registered tool.
func (d *BasicDispatcher) Dispatch(ctx context.Context, call protocol.ToolCall) (string, error) {
	handler, ok := d.handlers[call.Name]
	if !ok {
		return "", fmt.Errorf("tool not found: %s", call.Name)
	}
	return handler(ctx, call.Arguments)
}

// Definitions returns all registered tool definitions that are currently available.
func (d *BasicDispatcher) Definitions() []protocol.ToolDef {
	var defs []protocol.ToolDef
	for _, def := range d.tools {
		if availability.IsAvailable(def.Name) {
			defs = append(defs, def)
		}
	}
	return defs
}

