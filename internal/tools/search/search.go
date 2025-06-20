package search

import (
	"context"
	"fmt"
	"github.com/cloudwego/eino-ext/components/tool/duckduckgo" // 假设使用DuckDuckGo搜索
	"github.com/cloudwego/eino-ext/components/tool/duckduckgo/ddgsearch"
	"github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/mcp"
	tol "mcp-server/internal/tool"
	"time"
)

// SearchTool 搜索工具实现
type SearchTool struct {
	impl tool.InvokableTool
	cfg  *Config
}

// Config 搜索工具配置
type Config struct {
	CacheDuration string `json:"cache_duration"` // 缓存时长（如"5m"）
}

// NewSearchTool 构造函数（实现ToolConstructor）
func NewSearchTool(ctx context.Context, cfg any, deps tol.Dependencies) (tol.Tool, error) {
	searchCfg, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("invalid config type for search tool")
	}

	// 实际应使用具体搜索工具的配置
	impl, err := duckduckgo.NewTool(ctx, &duckduckgo.Config{
		ToolName:   "web_search",
		ToolDesc:   "网页搜索工具（获取实时信息）",
		MaxResults: 5, // 默认结果数量
		DDGConfig: &ddgsearch.Config{
			Timeout: parseDuration(searchCfg.CacheDuration),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create search tool failed: %w", err)
	}

	return &SearchTool{
		impl: impl,
		cfg:  searchCfg,
	}, nil
}

func parseDuration(duration string) time.Duration {
	if duration == "" {
		return time.Minute // 默认缓存1分钟
	}
	dur, err := time.ParseDuration(duration)
	if err != nil {
		return time.Minute // 解析失败，返回默认值
	}
	return dur
}

// GetDescriptor 实现工具接口
func (t *SearchTool) GetDescriptor() *mcp.Tool {
	tol := mcp.NewTool("web_search",
		mcp.WithDescription("网页搜索工具（获取实时信息）"),
		mcp.WithString("query", mcp.Required(), mcp.Description("搜索关键词")),
		mcp.WithString("limit", mcp.Description("结果数量限制")),
	)
	return &tol
}

// Execute 实现工具接口
func (t *SearchTool) Execute(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 从请求参数中提取值
	args := req.GetArguments()
	if args == nil {
		return mcp.NewToolResultError("参数格式错误: 需要是map类型"), nil
	}

	query, ok := args["query"].(string)
	if !ok {
		return mcp.NewToolResultError("缺少或无效的query参数"), nil
	}
	//var limit int
	//if limitVal, ok := args["limit"]; ok {
	//	switch v := limitVal.(type) {
	//	case int:
	//		limit = v
	//	case float64:
	//		limit = int(v)
	//	default:
	//		return mcp.NewToolResultError("limit参数必须是数字"), nil
	//	}
	//} else {
	//	limit = 5 // 默认值
	//}

	// 调用 InvokableTool 接口
	result, err := t.impl.InvokableRun(ctx, fmt.Sprintf(`{"query":"%s","limit":%d}`, query, 10))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("搜索失败: %v", err)), nil
	}

	return mcp.NewToolResultText(result), nil
}

// Name 实现工具接口
func (t *SearchTool) Name() string {
	return "web_search"
}
