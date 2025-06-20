package tool

import (
	"context"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/mark3labs/mcp-go/mcp"
)

// Tool 工具接口定义
type Tool interface {
	// GetDescriptor 返回工具描述信息（MCP元数据）
	GetDescriptor() *mcp.Tool
	// Execute 执行工具逻辑
	Execute(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)

	// Name 返回工具唯一标识
	Name() string
}

// Dependencies 公共依赖集合（根据实际需求扩展）
type Dependencies struct {
	ChatModel *openai.ChatModel // 假设需要共享的OpenAI模型
	// 其他公共依赖（如数据库、缓存等）
}

// Constructor 工具构造函数类型（用于依赖注入）
type Constructor func(ctx context.Context, cfg any, deps Dependencies) (Tool, error)
