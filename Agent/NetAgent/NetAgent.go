package netagent

import (
	"agent"
	"agent/tools"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

type Message struct {
	Role        string       `json:"role"`                   // 角色：system, user, assistant, tool
	Content     string       `json:"content"`                // 消息内容
	Name        string       `json:"name,omitempty"`         // Function/tool name (for tool role)
	Description string       `json:"description,omitempty"`  // Function/tool description (for tool role)
	ToolCallID  string       `json:"tool_call_id,omitempty"` // Tool call ID (for tool role)
	ToolCalls   []tools.Tool `json:"tool_calls,omitempty"`   // Tool calls (for assistant role)
	FromNodeID  string       `json:"from_node_id,omitempty"` // Source node ID
}

type AgentNode struct {
	ID         string
	agent      *agent.Agent
	InputChan  chan []Message
	OutputChan chan []Message
}

func NewAgentNode(id string, agent *agent.Agent) *AgentNode {
	return &AgentNode{
		ID:         id,
		agent:      agent,
		InputChan:  make(chan []Message, 10),
		OutputChan: make(chan []Message, 10),
	}
}

// NetAgent manages a graph of agent nodes and their connections.
type NetAgent struct {
	nodes    map[string]*AgentNode
	inEdges  map[string]map[string]struct{}
	outEdges map[string]map[string]struct{}
	router   RouteFunc
	ctx      context.Context
	cancel   context.CancelFunc
	started  bool
	wg       sync.WaitGroup
	mu       sync.RWMutex
}

// selfID: 当前处理消息的节点 ID
// replyToID: 触发这条消息的发送者 ID (即上一条消息的 FromID)
type RouteFunc func(ctx context.Context, selfID string, replyToID string, input []Message, reply string) (nextIDs []string, outMessages []Message, stop bool)

func NewNetAgent() *NetAgent {
	return &NetAgent{
		nodes:    make(map[string]*AgentNode),
		inEdges:  make(map[string]map[string]struct{}),
		outEdges: make(map[string]map[string]struct{}),
		router:   nil,
	}
}

func (n *NetAgent) AddNode(id string, a *agent.Agent) (*AgentNode, error) {
	if id == "" {
		return nil, fmt.Errorf("node id is required")
	}
	if a == nil {
		return nil, fmt.Errorf("agent is required")
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if _, exists := n.nodes[id]; exists {
		return nil, fmt.Errorf("node %q already exists", id)
	}
	node := NewAgentNode(id, a)
	n.nodes[id] = node
	n.inEdges[id] = make(map[string]struct{})
	n.outEdges[id] = make(map[string]struct{})
	n.attachCommTools(id, a)
	if n.started {
		n.startNodeLoop(node)
	}
	return node, nil
}

func (n *NetAgent) DeleteNode(id string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if _, exists := n.nodes[id]; !exists {
		return
	}
	for from := range n.inEdges[id] {
		delete(n.outEdges[from], id)
	}
	for to := range n.outEdges[id] {
		delete(n.inEdges[to], id)
	}
	delete(n.inEdges, id)
	delete(n.outEdges, id)
	delete(n.nodes, id)
}

func (n *NetAgent) AddEdge(fromID, toID string) error {
	if fromID == "" || toID == "" {
		return fmt.Errorf("fromID and toID are required")
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if _, ok := n.nodes[fromID]; !ok {
		return fmt.Errorf("from node %q not found", fromID)
	}
	if _, ok := n.nodes[toID]; !ok {
		return fmt.Errorf("to node %q not found", toID)
	}
	if n.outEdges[fromID] == nil {
		n.outEdges[fromID] = make(map[string]struct{})
	}
	if n.inEdges[toID] == nil {
		n.inEdges[toID] = make(map[string]struct{})
	}
	n.outEdges[fromID][toID] = struct{}{}
	n.inEdges[toID][fromID] = struct{}{}
	return nil
}

func (n *NetAgent) DeleteEdge(fromID, toID string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if out, ok := n.outEdges[fromID]; ok {
		delete(out, toID)
	}
	if in, ok := n.inEdges[toID]; ok {
		delete(in, fromID)
	}
}

func (n *NetAgent) GetNode(id string) (*AgentNode, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	node, ok := n.nodes[id]
	return node, ok
}

func (n *NetAgent) GetInNodes(id string) ([]*AgentNode, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if _, ok := n.nodes[id]; !ok {
		return nil, false
	}
	ins := n.inEdges[id]
	nodes := make([]*AgentNode, 0, len(ins))
	for from := range ins {
		if node, ok := n.nodes[from]; ok {
			nodes = append(nodes, node)
		}
	}
	return nodes, true
}

func (n *NetAgent) GetOutNodes(id string) ([]*AgentNode, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if _, ok := n.nodes[id]; !ok {
		return nil, false
	}
	outs := n.outEdges[id]
	nodes := make([]*AgentNode, 0, len(outs))
	for to := range outs {
		if node, ok := n.nodes[to]; ok {
			nodes = append(nodes, node)
		}
	}
	return nodes, true
}

func (n *NetAgent) SetRouter(router RouteFunc) {
	n.mu.Lock()
	n.router = router
	n.mu.Unlock()
}

// Start launches background loops for all nodes and keeps them active.
func (n *NetAgent) Start(ctx context.Context) {
	n.mu.Lock()
	if n.started {
		n.mu.Unlock()
		return
	}
	n.started = true
	n.ctx, n.cancel = context.WithCancel(ctx)
	nodes := make([]*AgentNode, 0, len(n.nodes))
	for _, node := range n.nodes {
		nodes = append(nodes, node)
	}
	n.mu.Unlock()

	for _, node := range nodes {
		n.startNodeLoop(node)
	}
}

// Stop stops all node loops.
func (n *NetAgent) Stop() {
	n.mu.Lock()
	if !n.started {
		n.mu.Unlock()
		return
	}
	n.cancel()
	n.mu.Unlock()
	n.wg.Wait()
}

func (n *NetAgent) startNodeLoop(node *AgentNode) {
	n.wg.Add(1)
	go func() {
		defer n.wg.Done()
		n.nodeLoop(node)
	}()
}

func (n *NetAgent) nodeLoop(node *AgentNode) {
	for {
		select {
		case <-n.ctx.Done():
			return
		case msgs := <-node.InputChan:
			if len(msgs) == 0 {
				continue
			}
			lastMsg := msgs[len(msgs)-1]
			senderID := lastMsg.FromNodeID
			content := latestContent(msgs)
			if content == "" {
				continue
			}
			// 注入来源上下文，这对 Agent 决定调度至关重要
			contextualContent := fmt.Sprintf("[Message from %s]: %s", senderID, content)
			fmt.Printf("[PROCESS] %s processing message from %s: %s", node.ID, senderID, content)

			// 调用 Agent 处理消息 等待回复
			reply, err := node.agent.Invoke(n.ctx, contextualContent)
			if err != nil {
				log.Printf("netagent invoke error on %s: %v", node.ID, err)
				continue
			}
			// 路由回复至下游节点
			nextIDs, outMessages, stop := n.route(n.ctx, node.ID, senderID, msgs, reply)
			if stop {
				continue
			}
			if len(outMessages) == 0 {
				outMessages = []Message{{
					Role:       "user", // 发给下游时，角色通常转为 user (即下游Agent看到的输入)
					Content:    reply,
					FromNodeID: node.ID, // 标记我自己是发送者
				}}
			} else {
				// 确保路由返回的消息也标记了 FromID
				for i := range outMessages {
					if outMessages[i].FromNodeID == "" {
						outMessages[i].FromNodeID = node.ID
					}
				}
			}
			if _, err := n.send(node.ID, nextIDs, outMessages); err != nil {
				log.Printf("netagent send error from %s: %v", node.ID, err)
			}
		}
	}
}

func (n *NetAgent) route(ctx context.Context, selfID string, replyToID string, input []Message, reply string) ([]string, []Message, bool) {
	n.mu.RLock()
	router := n.router
	n.mu.RUnlock()

	// 如果没有设置 router，使用默认的
	if router == nil {
		router = defaultRoute
	}

	// 调用具体的路由策略
	nextIDs, outMessages, stop := router(ctx, selfID, replyToID, input, reply)

	// 如果策略决定停止，或者已经指定了目标 ID，直接返回
	if stop || len(nextIDs) > 0 {
		return nextIDs, outMessages, stop
	}

	// === 兜底逻辑 (Broadcast) ===
	// 只有当 Router 既没有喊停，也没有给出目标时，才执行广播
	// 在 SmartRoute 模式下，这个逻辑很少触发，除非你希望保留“群发”功能
	n.mu.RLock()
	for id := range n.outEdges[selfID] {
		nextIDs = append(nextIDs, id)
	}
	n.mu.RUnlock()

	if len(nextIDs) == 0 {
		return nil, outMessages, true
	}
	return nextIDs, outMessages, false
}

// SmartRoute 实现智能调度
func SmartRouter(ctx context.Context, selfID string, replyToID string, input []Message, reply string) (nextIDs []string, outMessages []Message, stop bool) {
	trimmedReply := strings.TrimSpace(reply)

	// === 情况 A：工具接管 (显式调度) ===
	// 如果 Agent 调用了 "send" 工具，Invoke 返回的结果通常是工具的执行日志（如 "delivered to..."）
	// 或者，如果 Agent 只是想思考而不说话，可能返回空。
	// 在这些情况下，我们停止默认的路由转发，防止把 "delivered to node B" 这句话发给别人。
	if strings.Contains(reply, "delivered to") || trimmedReply == "" {
		return nil, nil, true // Stop! 停止转发
	}

	// === 情况 B：普通回复 (隐式调度/恢复) ===
	// Agent 没有用工具，只是说了一句话。这时候默认认为它是回复给“发消息的人”的。
	if replyToID != "" && replyToID != selfID {
		// 构造发给对方的消息
		outMsg := Message{
			Role:       "user", // 对接收者来说，这是用户输入
			Content:    reply,
			FromNodeID: selfID, // 标记是我发的
		}

		// 路由回发送者 (Reply-To-Sender)
		return []string{replyToID}, []Message{outMsg}, false
	}

	// === 情况 C：无主的广播 ===
	// 既没有用 send 工具，也不知道发给谁 (replyToID 为空)。
	// 这种通常是自启动任务或 Cron 任务。
	// 返回 nil, nil, false 会触发 route 方法里的“广播给所有邻居”的兜底逻辑
	// 如果你想禁止广播，这里返回 true
	return nil, nil, false
}

func defaultRoute(_ context.Context, selfID string, _ string, _ []Message, reply string) ([]string, []Message, bool) {
	if strings.TrimSpace(reply) == "" {
		return nil, nil, true
	}
	// 默认行为：不指定 nextIDs (触发广播)，只包装消息
	outMsg := Message{
		Role:       "user",
		Content:    reply,
		FromNodeID: selfID,
	}
	return nil, []Message{outMsg}, false
}

func latestContent(msgs []Message) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if strings.TrimSpace(msgs[i].Content) != "" {
			return msgs[i].Content
		}
	}
	return ""
}

func (n *NetAgent) attachCommTools(nodeID string, a *agent.Agent) {
	if a == nil {
		return
	}
	a.RegisterToolFunc(
		"send",
		func(ctx context.Context, args string) (string, error) {
			var input struct {
				ToID     string    `json:"to_id"`
				ToIDs    []string  `json:"to_ids"`
				Messages []Message `json:"messages"`
				Content  string    `json:"content"`
			}
			if err := json.Unmarshal([]byte(args), &input); err != nil {
				return "", fmt.Errorf("parse args: %w", err)
			}
			targets := input.ToIDs
			if input.ToID != "" {
				targets = append(targets, input.ToID)
			}
			msgs := input.Messages
			if len(msgs) == 0 && input.Content != "" {
				msgs = []Message{{Role: "user", Content: input.Content}}
			}
			if len(msgs) == 0 {
				return "", fmt.Errorf("messages or content is required")
			}
			delivered, err := n.send(nodeID, targets, msgs)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("delivered to %d node(s)", delivered), nil
		},
		tools.WithDescription("Send messages to connected nodes."),
		tools.WithParameters(tools.ObjectSchema(map[string]any{
			"to_id":   tools.StringProperty("Target node id."),
			"to_ids":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"content": tools.StringProperty("Shortcut text content."),
			"messages": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"role":    tools.StringProperty("Message role."),
						"content": tools.StringProperty("Message content."),
					},
				},
			},
		})),
	)

	// a.RegisterToolFunc(
	// 	"recv",
	// 	func(ctx context.Context, args string) (string, error) {
	// 		var input struct {
	// 			Max       int `json:"max"`
	// 			TimeoutMS int `json:"timeout_ms"`
	// 		}
	// 		if err := json.Unmarshal([]byte(args), &input); err != nil {
	// 			return "", fmt.Errorf("parse args: %w", err)
	// 		}
	// 		if input.Max <= 0 {
	// 			input.Max = 1
	// 		}
	// 		msgs, err := n.recv(nodeID, input.Max, time.Duration(input.TimeoutMS)*time.Millisecond)
	// 		if err != nil {
	// 			return "", err
	// 		}
	// 		data, err := json.Marshal(msgs)
	// 		if err != nil {
	// 			return "", fmt.Errorf("marshal messages: %w", err)
	// 		}
	// 		return string(data), nil
	// 	},
	// 	tools.WithDescription("Receive pending messages for this node. And deliver them to the agent for processing."),
	// 	tools.WithParameters(tools.ObjectSchema(map[string]any{
	// 		"max":        tools.IntProperty("Maximum number of messages to receive."),
	// 		"timeout_ms": tools.IntProperty("Wait time in milliseconds for the first message."),
	// 	})),
	//
}

func (n *NetAgent) send(fromID string, toIDs []string, messages []Message) (int, error) {
	targets, err := n.resolveTargets(fromID, toIDs)
	if err != nil {
		return 0, err
	}
	for i := range messages {
		messages[i].FromNodeID = fromID
	}
	delivered := 0
	for _, node := range targets {
		select {
		case node.InputChan <- messages:
			delivered++
			for _, msg := range messages {
				log.Printf("[SEND] %s → %s: %s", fromID, node.ID, msg.Content)
			}
		default:
			return delivered, fmt.Errorf("input channel full for node %q", node.ID)
		}
	}
	return delivered, nil
}

func (n *NetAgent) recv(nodeID string, max int, timeout time.Duration) ([]Message, error) {
	n.mu.RLock()
	node, ok := n.nodes[nodeID]
	n.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("node %q not found", nodeID)
	}

	var first []Message
	if timeout > 0 {
		select {
		case first = <-node.InputChan:
		case <-time.After(timeout):
			return nil, nil
		}
	} else {
		select {
		case first = <-node.InputChan:
		default:
			return nil, nil
		}
	}

	results := make([]Message, 0, max)
	results = append(results, first...)
	for len(results) < max {
		select {
		case next := <-node.InputChan:
			results = append(results, next...)
		default:
			return results, nil
		}
	}
	for _, msg := range results {
		log.Printf("[RECV] %s ← (from network): %s", nodeID, msg.Content)
	}

	return results, nil
}

func (n *NetAgent) resolveTargets(fromID string, toIDs []string) ([]*AgentNode, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if _, ok := n.nodes[fromID]; !ok {
		return nil, fmt.Errorf("from node %q not found", fromID)
	}
	targetSet := make(map[string]struct{})
	if len(toIDs) == 0 {
		for id := range n.outEdges[fromID] {
			targetSet[id] = struct{}{}
		}
	} else {
		for _, id := range toIDs {
			if id == "" {
				continue
			}
			if _, ok := n.outEdges[fromID][id]; !ok {
				return nil, fmt.Errorf("no edge from %q to %q", fromID, id)
			}
			targetSet[id] = struct{}{}
		}
	}
	if len(targetSet) == 0 {
		return nil, fmt.Errorf("no target nodes available")
	}
	targets := make([]*AgentNode, 0, len(targetSet))
	for id := range targetSet {
		if node, ok := n.nodes[id]; ok {
			targets = append(targets, node)
		}
	}
	return targets, nil
}
