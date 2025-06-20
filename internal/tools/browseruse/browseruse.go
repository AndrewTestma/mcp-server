package browseruse

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudwego/eino-ext/components/tool/browseruse"
	"github.com/mark3labs/mcp-go/mcp"
	"log"
	"mcp-server/internal/tool"
)

// BrowserTool 浏览器工具实现
type BrowserTool struct {
	impl *browseruse.Tool
	cfg  *Config
}

// Config 浏览器工具配置
type Config struct {
	Headless          bool     `json:"headless"`
	DisableSecurity   bool     `json:"disable_security"`
	ExtraChromiumArgs []string `json:"extra_chromium_args"`
}

func NewBrowseruseTool(ctx context.Context, cfg any, deps tool.Dependencies) (tool.Tool, error) {
	config, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("invalid config type for browser tool")
	}

	impl, err := browseruse.NewBrowserUseTool(ctx, &browseruse.Config{
		Headless:          config.Headless,
		DisableSecurity:   config.DisableSecurity,
		ExtraChromiumArgs: config.ExtraChromiumArgs,
		DDGSearchTool:     nil,
		ExtractChatModel:  deps.ChatModel,
		Logf:              log.Printf,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create browser tool: %v", err)
	}
	return &BrowserTool{
		impl: impl,
		cfg:  config,
	}, nil

}

// GetDescriptor 实现工具接口
func (t *BrowserTool) GetDescriptor() *mcp.Tool {
	tool := mcp.NewTool("browseruse",
		mcp.WithDescription("真实浏览器操作工具（导航、内容提取等）"),
		mcp.WithString("action", mcp.Required(), mcp.Description("操作类型（go_to_url, click_element, extract_content等")),
		mcp.WithString("url", mcp.Description("目标URL（用于导航或打开新标签）")),
		mcp.WithString("goal", mcp.Description("提取目标（用于内容提取）")),
		// 将整数参数改为字符串类型
		mcp.WithString("index", mcp.Description("元素索引")),
		mcp.WithString("scroll_amount", mcp.Description("滚动像素数")),
		mcp.WithString("tab_id", mcp.Description("标签页ID")),
		mcp.WithString("query", mcp.Description("搜索查询")),
	)
	return &tool
}

// Execute 实现工具接口
func (t *BrowserTool) Execute(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var param browseruse.Param
	switch arg := req.Params.Arguments.(type) {
	case string:
		if err := json.Unmarshal([]byte(arg), &param); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("参数解析失败: %v", err)), nil
		}
	case map[string]interface{}:
		argJSON, _ := json.Marshal(arg)
		if err := json.Unmarshal(argJSON, &param); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("参数解析失败（map）: %v", err)), nil
		}
	default:
		return mcp.NewToolResultError(fmt.Sprintf("不支持的参数类型: %T", arg)), nil
	}

	result, err := t.impl.Execute(&param)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("操作失败: %v", err)), nil
	}

	output, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(output)), nil
}

// Name 实现工具接口
func (t *BrowserTool) Name() string {
	return "browseruse"
}
