package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"agent"
	"agent/tools/buildin"
)

func main() {
	cfg, err := agent.LoadAgentConfig(".")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if cfg.APIKey == "" {
		log.Fatal("missing API key; set AGENT_API_KEY or agent.yaml")
	}
	if cfg.Model == "" {
		cfg.Model = "gpt-4o-mini"
	}

	var chatAgent interface {
		Invoke(context.Context, string) (string, error)
	}
	var base *agent.Agent

	if cfg.ReAct.Enabled {
		reactAgent := agent.NewReActAgent(cfg.APIKey, cfg.BaseURL, cfg.Model)
		reactAgent.Temperature = cfg.Temperature
		reactAgent.Maxcircle = cfg.MaxCircle
		base = reactAgent.Agent
		chatAgent = reactAgent
	} else {
		baseAgent := agent.NewAgent(cfg.APIKey, cfg.BaseURL, cfg.Model, cfg.AllowTools)
		baseAgent.Temperature = cfg.Temperature
		baseAgent.Maxcircle = cfg.MaxCircle
		base = baseAgent
		chatAgent = baseAgent
	}

	registerTools(base)

	fmt.Println("Simple chat agent with tools. Type 'exit' to quit.")
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("You> ")
		if !scanner.Scan() {
			break
		}
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		if text == "exit" || text == "quit" {
			break
		}
		reply, err := chatAgent.Invoke(context.Background(), text)
		if err != nil {
			log.Printf("agent error: %v", err)
			continue
		}
		fmt.Printf("Agent> %s\n", reply)
		if base != nil {
			base.AddMemory(reply)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("input error: %v", err)
	}
}

func registerTools(a *agent.Agent) {
	if a == nil {
		return
	}
	a.RegisterTool(buildin.NewWebSearchTool())
	a.RegisterTool(buildin.NewGetWeatherTool())
	a.RegisterTool(buildin.NewGetCurrentTimeTool())
}
