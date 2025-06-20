package tool

import (
	"context"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"sync"
)

// ToolManager 工具管理器
type ToolManager struct {
	tools        map[string]Tool
	constructors map[string]Constructor
	deps         Dependencies
	mu           sync.RWMutex
}

// NewToolManager 创建工具管理器实例
func NewToolManager(deps Dependencies) *ToolManager {
	return &ToolManager{
		tools:        make(map[string]Tool),
		constructors: make(map[string]Constructor),
		deps:         deps,
	}
}

// Register 注册工具构造函数（启动时调用
func (m *ToolManager) Register(name string, constructor Constructor) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.constructors[name] = constructor
}

// InitTools 初始化所有注册的工具（从配置加载)
func (m *ToolManager) InitTools(ctx context.Context, toolCfgs map[string]any) error {
	for toolName, constructor := range m.constructors {
		cfg, ok := toolCfgs[toolName]
		if !ok {
			return fmt.Errorf("missing config for tool: %s", toolName)
		}
		tool, err := constructor(ctx, cfg, m.deps)
		if err != nil {
			return fmt.Errorf("failed to initialize tool %s: %v", toolName, err)
		}
		m.tools[toolName] = tool
	}
	return nil
}

// RegisterToServer 将所有工具注册到MCP服务器
func (m *ToolManager) RegisterToServer(svr *server.MCPServer) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, tool := range m.tools {
		svr.AddTool(
			*tool.GetDescriptor(),
			func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return tool.Execute(ctx, request)
			},
		)
	}
}
