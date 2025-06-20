package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino-ext/components/tool/browseruse"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"log"
)

func main1() {
	startMCPServer()
	select {} // 阻塞主线程，保持程序运行
}

func startMCPServer() {
	svr := server.NewMCPServer("browser-service", mcp.LATEST_PROTOCOL_VERSION)

	chatModel, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		APIKey:  "sk-14d8e0db18214302a2ae13e493094d98",
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		Model:   "qwen-plus",
	})
	if err != nil {
		log.Fatalf("创建聊天模型失败: %v", err)
	}

	//bingTool, err := bingsearch.NewTool(context.Background(), &bingsearch.Config{
	//	APIKey: "sk-14d8e0db18214302a2ae13e493094d98",
	//	Cache:  5 * time.Minute,
	//})

	browserTool, err := browseruse.NewBrowserUseTool(context.Background(), &browseruse.Config{
		Headless:          false,
		DisableSecurity:   true,
		ExtraChromiumArgs: []string{"--start-maximized"},
		DDGSearchTool:     nil,
		ExtractChatModel:  chatModel,
		Logf:              log.Printf,
	})
	if err != nil {
		log.Fatalf("初始化浏览器工具失败: %v", err)
	}

	svr.AddTool(
		mcp.NewTool("browseruse",
			mcp.WithDescription("真实浏览器操作工具（导航、内容提取等）"),
			mcp.WithString("action", mcp.Required(), mcp.Description("操作类型（go_to_url, click_element, extract_content等")),
			mcp.WithString("url", mcp.Description("目标URL（用于导航或打开新标签）")),
			mcp.WithString("goal", mcp.Description("提取目标（用于内容提取）")),
			// 将整数参数改为字符串类型
			mcp.WithString("index", mcp.Description("元素索引")),
			mcp.WithString("scroll_amount", mcp.Description("滚动像素数")),
			mcp.WithString("tab_id", mcp.Description("标签页ID")),
			mcp.WithString("query", mcp.Description("搜索查询")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// 步骤 1：将参数反序列化为结构体（兼容 string 或 map 类型）
			var param browseruse.Param
			switch arg := request.Params.Arguments.(type) {
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
			// 步骤 2：执行真实浏览器操作（示例：导航到 URL）
			result, err := browserTool.Execute(&param)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("浏览器操作失败: %v", err)), nil
			}
			// 步骤 3：返回结果（示例：返回提取的内容）
			if result == nil {
				return mcp.NewToolResultText("操作成功，但未返回结果"), nil
			}
			// 将结果转换为MCP要求的格式
			output, _ := json.Marshal(result)
			return mcp.NewToolResultText(string(output)), nil
		})
	go func() {
		sseServer := server.NewSSEServer(svr, server.WithBaseURL("http://localhost:12345"))
		err := sseServer.Start("localhost:12345")
		if err != nil {
			log.Fatalf("启动SSE服务器失败: %v", err)
		}
		log.Println("MCP服务端启动成功，监听端口12345")
	}()
}
