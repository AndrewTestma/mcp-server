package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
	"mcp-server/config"
	"mcp-server/coordinator"
	"mcp-server/internal/log"
	"mcp-server/internal/tool"
	"mcp-server/internal/tools/milvus"
	"net/http"
)

func main() {
	// 初始化日志
	log.Init()
	logger := log.GetLogger()

	// 1. 加载配置
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		logger.Error("加载配置失败", zap.Error(err))
		return
	}
	logger.Info("配置加载成功")

	// 2. 初始化公共依赖（如OpenAI模型）
	chatModel, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		APIKey:  "sk-14d8e0db18214302a2ae13e493094d98", // 从配置/环境变量获取
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		Model:   "qwen-plus",
	})
	if err != nil {
		logger.Error("创建聊天模型失败: %v", zap.Error(err))
	}

	// 3. 初始化工具管理器
	toolManager := tool.NewToolManager(tool.Dependencies{
		ChatModel: chatModel,
	})

	//toolManager.Register("browseruse", browseruse.NewBrowseruseTool)
	//toolManager.Register("web_search", search.NewSearchTool)
	toolManager.Register("milvus", milvus.NewMilvusTool) // 新增Milvus工具注册
	// 4. 初始化协调器
	coor := coordinator.NewCoordinator(context.Background(), chatModel, toolManager)
	// 5. 初始化所有工具
	if err := toolManager.InitTools(context.Background(), cfg.Tools); err != nil {
		logger.Error("初始化工具失败: %v", zap.Error(err))
	}

	// 6. 启动MCP服务器
	svr := server.NewMCPServer("multi-tool-service", "1.0")
	toolManager.RegisterToServer(svr)

	// 7. 启动SSE服务
	sseServer := server.NewSSEServer(svr, server.WithBaseURL(fmt.Sprintf("http://localhost:%s", cfg.ServerPort)))
	// 设置消息处理路由
	http.Handle("/messages", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 解析客户端发送的消息
		var clientRequest struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&clientRequest); err != nil {
			logger.Error("解析客户端消息失败", zap.Error(err))
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		clientID := r.URL.Query().Get("sessionId")
		logger.Info("收到客户端查询",
			zap.String("client", clientID),
			zap.String("query", clientRequest.Query))

		// 调用协调器处理查询
		result, err := coor.Run(clientRequest.Query)
		if err != nil {
			logger.Error("处理失败", zap.Error(err))
			http.Error(w, fmt.Sprintf("处理失败: %v", err), http.StatusInternalServerError)
			return
		}
		// 返回成功响应
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"result": result,
		})
	}))

	// 设置SSE路由
	http.Handle("/sse", sseServer.SSEHandler())
	if err := sseServer.Start(fmt.Sprintf("localhost:%s", cfg.ServerPort)); err != nil {
		logger.Error("启动服务器失败", zap.Error(err))
	}

	logger.Info("MCP服务启动成功，监听端口: %s", zap.String("port", cfg.ServerPort))
	select {} // 阻塞主线程
}
