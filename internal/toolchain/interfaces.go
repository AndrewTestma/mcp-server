package toolchain

import (
	"context"
	"github.com/mark3labs/mcp-go/mcp"
)

// SearchTool 定义搜索工具接口（抽象 web_search 行为）
type SearchTool interface {
	Execute(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)
}

// BrowserTool 定义浏览器工具接口（抽象 browseruse 行为）
type BrowserTool interface {
	Execute(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)
}
