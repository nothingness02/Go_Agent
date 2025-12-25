package agent

import (
	"context"
)

// ReActAgent wraps a base Agent with ReAct-style prompting.
type ReActAgent struct {
	*Agent
}

// NewReActAgent creates an agent configured for ReAct prompting.
func NewReActAgent(apiKey string, baseURL string, model string) *ReActAgent {
	base := NewAgent(apiKey, baseURL, model, true)
	base.SetSystemPrompt(DefaultReActSystemPrompt)
	base.SetPromptWrapper(ReActPromptWrapper())
	return &ReActAgent{Agent: base}
}

// Invoke runs the ReAct agent, defaulting to allow tool calls.
func (r *ReActAgent) Invoke(ctx context.Context, userQuery string) (string, error) {
	return r.Agent.Invoke(ctx, userQuery)
}

// BaseAgent is the plain agent without ReAct prompting.
type BaseAgent struct {
	*Agent
}

// NewBaseAgent creates a plain base agent.
func NewBaseAgent(apiKey string, baseURL string, model string) *BaseAgent {
	return &BaseAgent{Agent: NewAgent(apiKey, baseURL, model, false)}
}
