package tools

import "context"

// ToolHandler defines the tool handler signature.
type ToolHandler func(ctx context.Context, args string) (string, error)

type ToolKind string

const (
	ToolKindTool     ToolKind = "tool"
	ToolKindFunction ToolKind = "function"
)

type Tool struct {
	Name        string
	Description string
	Parameters  map[string]any
	Handler     ToolHandler
	Kind        ToolKind
}

type Option func(*Tool)

func New(name string, handler ToolHandler, opts ...Option) Tool {
	t := Tool{
		Name:    name,
		Handler: handler,
		Kind:    ToolKindTool,
	}
	for _, opt := range opts {
		opt(&t)
	}
	return t
}

func WithDescription(description string) Option {
	return func(t *Tool) {
		t.Description = description
	}
}

func WithParameters(parameters map[string]any) Option {
	return func(t *Tool) {
		t.Parameters = parameters
	}
}

func WithKind(kind ToolKind) Option {
	return func(t *Tool) {
		t.Kind = kind
	}
}

func ObjectSchema(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func StringProperty(description string) map[string]any {
	prop := map[string]any{
		"type": "string",
	}
	if description != "" {
		prop["description"] = description
	}
	return prop
}

func IntProperty(description string) map[string]any {
	prop := map[string]any{
		"type": "integer",
	}
	if description != "" {
		prop["description"] = description
	}
	return prop
}
