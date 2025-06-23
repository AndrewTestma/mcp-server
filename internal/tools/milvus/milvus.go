package milvus

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudwego/eino-ext/components/retriever/milvus"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"mcp-server/internal/tool"
)

// MilvusTool 实现MCP工具接口
type MilvusTool struct {
	retriever *milvus.Retriever
	cfg       *Config
}

// Config 工具配置
type Config struct {
	Address           string
	Username          string
	Password          string
	SiliconFlowAPIKey string `json:"silicon_flow_api_key"` // SiliconFlow API密钥
	ModelName         string `json:"model_name"`           // 模型名称
	Collection        string
	VectorField       string
	TopK              int
	ScoreThreshold    float64
}

func NewMilvusTool(ctx context.Context, cfg any, deps tool.Dependencies) (tool.Tool, error) {
	config, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("invalid config type for milvus tool")
	}
	cli, err := client.NewClient(ctx, client.Config{
		Address:  config.Address,
		Username: config.Username,
		Password: config.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to milvus: %v", err)
	}
	emb := NewSiliconFlowEmbedder(config.SiliconFlowAPIKey, config.ModelName)

	retriever, err := milvus.NewRetriever(ctx, &milvus.RetrieverConfig{
		Client:         cli,
		Collection:     config.Collection,
		VectorField:    config.VectorField,
		TopK:           config.TopK,
		ScoreThreshold: config.ScoreThreshold,
		Embedding:      emb,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create milvus retriever: %v", err)
	}
	return &MilvusTool{
		retriever: retriever,
		cfg:       config,
	}, nil
}

// GetDescriptor 实现工具接口
func (t *MilvusTool) GetDescriptor() *mcp.Tool {
	tool := mcp.NewTool("milvus",
		mcp.WithDescription("Milvus向量检索工具"),
		mcp.WithString("query", mcp.Required(), mcp.Description("检索查询文本")),
	)
	return &tool
}

// Execute 实现工具接口
func (t *MilvusTool) Execute(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var query string
	switch arg := req.Params.Arguments.(type) {
	case string:
		query = arg
	case map[string]interface{}:
		if q, ok := arg["query"].(string); ok {
			query = q
		} else {
			return mcp.NewToolResultError("参数中缺少query字段"), nil
		}
	default:
		return mcp.NewToolResultError(fmt.Sprintf("不支持的参数类型: %T", arg)), nil
	}

	documents, err := t.retriever.Retrieve(ctx, query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("检索失败: %v", err)), nil
	}

	output, _ := json.Marshal(documents)
	return mcp.NewToolResultText(string(output)), nil
}

// Name 实现工具接口
func (t *MilvusTool) Name() string {
	return "milvus"
}
