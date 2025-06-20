package main

import (
	"context"
	"fmt"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/mark3labs/mcp-go/server"
	"log"
	"mcp-server/config"
	"mcp-server/internal/tool"
	"mcp-server/internal/tools/browseruse"
	"mcp-server/internal/tools/search"
)

func main() {
	// 1. 加载配置
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 2. 初始化公共依赖（如OpenAI模型）
	chatModel, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		APIKey:  "sk-14d8e0db18214302a2ae13e493094d98", // 从配置/环境变量获取
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		Model:   "qwen-plus",
	})
	if err != nil {
		log.Fatalf("创建聊天模型失败: %v", err)
	}

	// 3. 初始化工具管理器
	toolManager := tool.NewToolManager(tool.Dependencies{
		ChatModel: chatModel,
	})

	// 4. 注册工具构造函数（新增工具只需添加这一行）
	toolManager.Register("browseruse", browseruse.NewBrowseruseTool)
	toolManager.Register("web_search", search.NewSearchTool)

	// 5. 初始化所有工具
	if err := toolManager.InitTools(context.Background(), cfg.Tools); err != nil {
		log.Fatalf("初始化工具失败: %v", err)
	}

	// 6. 启动MCP服务器
	svr := server.NewMCPServer("multi-tool-service", "1.0")
	toolManager.RegisterToServer(svr)

	// 7. 启动SSE服务
	sseServer := server.NewSSEServer(svr, server.WithBaseURL(fmt.Sprintf("http://localhost:%s", cfg.ServerPort)))
	if err := sseServer.Start(fmt.Sprintf("localhost:%s", cfg.ServerPort)); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}

	log.Printf("MCP服务启动成功，监听端口: %s", cfg.ServerPort)
	select {} // 阻塞主线程
}
