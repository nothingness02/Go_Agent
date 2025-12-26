package netagent

import (
	"agent"
	"context"
	"log"
	"testing"
	"time"
)

func TestCommunication(t *testing.T) {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	net := NewNetAgent()
	net.ctx = context.Background()
	net.SetRouter(SmartRouter)

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

	a1 := agent.NewAgent(cfg.APIKey, cfg.BaseURL, cfg.Model, true)
	a1.AddSystemPrompt("You are node A. Reply in the format: A_ACK: <content>.")
	a2 := agent.NewAgent(cfg.APIKey, cfg.BaseURL, cfg.Model, true)
	a2.AddSystemPrompt("You are node B. Reply in the format: B_ACK: <content>.")
	a3 := agent.NewAgent(cfg.APIKey, cfg.BaseURL, cfg.Model, true)
	a3.AddSystemPrompt("You are node C. Reply in the format: C_ACK: <content>.")

	if _, err := net.AddNode("A", a1); err != nil {
		t.Fatalf("AddNode A: %v", err)
	}
	if _, err := net.AddNode("B", a2); err != nil {
		t.Fatalf("AddNode B: %v", err)
	}
	if _, err := net.AddNode("C", a3); err != nil {
		t.Fatalf("AddNode C: %v", err)
	}
	if err := net.AddEdge("A", "B"); err != nil {
		t.Fatalf("AddEdge A->B: %v", err)
	}
	if err := net.AddEdge("B", "A"); err != nil {
		t.Fatalf("AddEdge B->A: %v", err)
	}
	if err := net.AddEdge("B", "C"); err != nil {
		t.Fatalf("AddEdge B->C: %v", err)
	}
	if err := net.AddEdge("C", "B"); err != nil {
		t.Fatalf("AddEdge C->B: %v", err)
	}
	if err := net.AddEdge("A", "C"); err != nil {
		t.Fatalf("AddEdge A->C: %v", err)
	}
	if err := net.AddEdge("C", "A"); err != nil {
		t.Fatalf("AddEdge C->A: %v", err)
	}
	net.Start(net.ctx)

	nodeA, ok := net.GetNode("A")
	if !ok {
		t.Fatal("node A not found")
	}
	nodeA.InputChan <- []Message{{
		Role:       "user",
		Content:    "你要开启一场有关人工智能对话，将你的回复传递给你想要交流的节点。",
		FromNodeID: "seed",
	}}
	time.Sleep(time.Minute * 5)
	net.Stop()
}
