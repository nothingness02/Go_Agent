package agent

import (
	"fmt"
	"strings"

	"github.com/openai/openai-go"
)

// PromptWrapper stores prompt segments for system and user messages.
type PromptWrapper struct {
	Memory        []string
	ToolUsage     []string
	systemPrompts []string
	userPrompts   []string
}

func DefaultPromptWrapper() PromptWrapper {
	return PromptWrapper{}
}

func ReActPromptWrapper() PromptWrapper {
	wrapper := PromptWrapper{}
	wrapper.AddSystemPrompt("You are a ReAct-style agent.")
	wrapper.AddToolUsage("Use tools when needed. Think about whether a tool is required, call it with structured arguments, then produce the final answer.")
	return wrapper
}

// AddSystemPrompt appends an extra system-role prompt segment.
func (w *PromptWrapper) AddSystemPrompt(prompt string) {
	if strings.TrimSpace(prompt) == "" {
		return
	}
	w.systemPrompts = append(w.systemPrompts, prompt)
}

// AddUserPrompt appends an extra user-role prompt segment.
func (w *PromptWrapper) AddUserPrompt(prompt string) {
	if strings.TrimSpace(prompt) == "" {
		return
	}
	w.userPrompts = append(w.userPrompts, prompt)
}

// AddMemory appends a memory segment for the system prompt.
func (w *PromptWrapper) AddMemory(memory string) {
	if strings.TrimSpace(memory) == "" {
		return
	}
	w.Memory = append(w.Memory, memory)
}

// AddToolUsage appends a tool-usage segment for the system prompt.
func (w *PromptWrapper) AddToolUsage(toolUsage string) {
	if strings.TrimSpace(toolUsage) == "" {
		return
	}
	w.ToolUsage = append(w.ToolUsage, toolUsage)
}

// WrapMessages builds chat messages from the stored prompt segments.
func (w *PromptWrapper) WrapMessages(name, desc string) []openai.ChatCompletionMessageParamUnion {
	systemParts := make([]string, 0, 8)
	if name != "" || desc != "" {
		systemParts = append(systemParts, fmt.Sprintf("Agent Name: %s\nAgent Description: %s", name, desc))
	}
	if len(w.Memory) > 0 {
		systemParts = append(systemParts, fmt.Sprintf("Memory:\n%s", strings.Join(w.Memory, "\n")))
	}
	if len(w.ToolUsage) > 0 {
		systemParts = append(systemParts, fmt.Sprintf("Tool Usage:\n%s", strings.Join(w.ToolUsage, "\n")))
	}
	if len(w.systemPrompts) > 0 {
		systemParts = append(systemParts, w.systemPrompts...)
	}

	userParts := make([]string, 0, 4)
	if len(w.userPrompts) > 0 {
		userParts = append(userParts, w.userPrompts...)
	}
	systemMessage := strings.TrimSpace(strings.Join(systemParts, "\n\n"))
	userMessage := strings.TrimSpace(strings.Join(userParts, "\n\n"))

	messages := make([]openai.ChatCompletionMessageParamUnion, 0, 2)
	if systemMessage != "" {
		messages = append(messages, openai.SystemMessage(systemMessage))
	}
	if userMessage != "" {
		messages = append(messages, openai.UserMessage(userMessage))
	}
	return messages
}
