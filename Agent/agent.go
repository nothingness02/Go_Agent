package agent

import (
	"context"
	"fmt"
	"log"

	"agent/tools"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const (
	DefaultName              string  = "Base_agent"
	DefaultDescription       string  = "The basic for the extended intelligent agent"
	DefaultSystemPrompt      string  = "You are a versatile individual in a general field, and you need to assist clients in completing diverse tasks"
	DefaultReActSystemPrompt string  = "You are a ReAct-style agent. Think step-by-step, decide when to call tools, and respond with final answers after tool use."
	DefaultMaxCircle         int     = 5
	DefaultTemperature       float32 = 0.5
)

type Message struct {
	Role       string       `json:"role"`                   // 角色：system, user, assistant, tool
	Content    string       `json:"content"`                // 消息内容
	Name       string       `json:"name,omitempty"`         // Function/tool name (for tool role)
	ToolCallID string       `json:"tool_call_id,omitempty"` // Tool call ID (for tool role)
	ToolCalls  []tools.Tool `json:"tool_calls,omitempty"`   // Tool calls (for assistant role)
}

type Agent struct {
	Name          string
	Description   string
	client        openai.Client
	model         string
	tools         map[string]tools.Tool
	apiTools      []openai.ChatCompletionToolParam
	promptWrapper PromptWrapper
	systemPrompt  string
	Maxcircle     int
	Temperature   float32
	AllowTools    bool
}

func NewAgent(apiKey string, baseURL string, model string, allow_tools bool) *Agent {
	options := []option.RequestOption{option.WithAPIKey(apiKey)}
	if baseURL != "" {
		options = append(options, option.WithBaseURL(baseURL))
	}
	return &Agent{
		Name:          DefaultName,
		Description:   DefaultDescription,
		client:        openai.NewClient(options...),
		model:         model,
		tools:         map[string]tools.Tool{},
		apiTools:      []openai.ChatCompletionToolParam{},
		promptWrapper: DefaultPromptWrapper(),
		systemPrompt:  DefaultSystemPrompt,
		Maxcircle:     DefaultMaxCircle,
		Temperature:   DefaultTemperature,
		AllowTools:    allow_tools,
	}
}

func (a *Agent) SetName(Name string) {
	a.Name = Name
}

func (a *Agent) SetDescription(Description string) {
	a.Description = Description
}

func (a *Agent) SetSystemPrompt(systemPrompt string) {
	a.systemPrompt = systemPrompt
}

func (a *Agent) SetPromptWrapper(wrapper PromptWrapper) {
	a.promptWrapper = wrapper
}

func (a *Agent) AddSystemPrompt(prompt string) {
	a.promptWrapper.AddSystemPrompt(prompt)
}

func (a *Agent) AddUserPrompt(prompt string) {
	a.promptWrapper.AddUserPrompt(prompt)
}

func (a *Agent) AddMemory(memory string) {
	a.promptWrapper.AddMemory(memory)
}

func (a *Agent) AddToolUsage(toolUsage string) {
	a.promptWrapper.AddToolUsage(toolUsage)
}

// 注册工具
func (a *Agent) ListTools() []tools.Tool {
	items := make([]tools.Tool, 0, len(a.tools))
	for _, tool := range a.tools {
		items = append(items, tool)
	}
	return items
}

func (a *Agent) RegisterTool(tool tools.Tool) {
	if tool.Name == "" {
		return
	}
	if tool.Kind == "" {
		tool.Kind = tools.ToolKindTool
	}
	a.tools[tool.Name] = tool
	functionDef := openai.FunctionDefinitionParam{
		Name: tool.Name,
	}
	if tool.Description != "" {
		functionDef.Description = openai.String(tool.Description)
	}
	if tool.Parameters != nil {
		functionDef.Parameters = openai.FunctionParameters(tool.Parameters)
	}
	a.apiTools = append(a.apiTools, openai.ChatCompletionToolParam{
		Function: functionDef,
	})
}

func (a *Agent) RegisterToolFunc(name string, handler tools.ToolHandler, opts ...tools.Option) {
	a.RegisterTool(tools.New(name, handler, opts...))
}

func (a *Agent) Invoke(ctx context.Context, userQuery string) (string, error) {
	wrapper := a.promptWrapper
	wrapper.AddSystemPrompt(a.systemPrompt)
	wrapper.AddUserPrompt(userQuery)
	messages := wrapper.WrapMessages(a.Name, a.Description)
	for i := 1; i <= a.Maxcircle; i++ {
		req := openai.ChatCompletionNewParams{
			Model:    a.model,
			Messages: messages,
		}
		req.Temperature = openai.Float(float64(a.Temperature))
		if a.AllowTools && len(a.apiTools) > 0 {
			req.Tools = a.apiTools
		}
		resp, err := a.client.Chat.Completions.New(ctx, req)
		if err != nil {
			return "", fmt.Errorf("llm error: %v", err)
		}
		msg := resp.Choices[0].Message
		messages = append(messages, msg.ToParam())
		if !a.AllowTools {
			if len(msg.ToolCalls) > 0 {
				return "", fmt.Errorf("tool calls disabled but received %d tool calls", len(msg.ToolCalls))
			}
			return msg.Content, nil
		}
		if len(msg.ToolCalls) == 0 {
			return msg.Content, nil
		}
		for _, toolCall := range msg.ToolCalls {
			toolName := toolCall.Function.Name
			args := toolCall.Function.Arguments
			tool, exists := a.tools[toolName]
			if !exists {
				log.Printf("Tool %s not found", toolName)
				continue
			}
			log.Printf("Agent calling tool: %s with args: %s", toolName, args)
			result, err := tool.Handler(ctx, args)
			if err != nil {
				result = fmt.Sprintf("Error executing tool: %v", err)
			}
			messages = append(messages, openai.ToolMessage(result, toolCall.ID))
		}
	}
	return "", fmt.Errorf("agent loop limit exceeded")
}
