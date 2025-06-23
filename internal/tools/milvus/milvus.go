package milvus

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudwego/eino-ext/components/retriever/milvus"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
	"mcp-server/internal/tool"
)

// MilvusTool 实现MCP工具接口
type MilvusTool struct {
	retriever *milvus.Retriever
	cfg       *Config
}

// Config 工具配置
type Config struct {
	Address           string  `json:"address"`
	SiliconFlowAPIKey string  `json:"silicon_flow_api_key"`
	ModelName         string  `json:"model_name"`
	Collection        string  `json:"collection"`
	VectorField       string  `json:"vector_field"`
	TopK              int     `json:"top_k"`
	ScoreThreshold    float64 `json:"score_threshold"`
	MetricType        string  `json:"metric_type"` // 添加度量类型字段
}

func NewMilvusTool(ctx context.Context, cfg any, deps tool.Dependencies) (tool.Tool, error) {
	config, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("invalid config type for milvus tool")
	}
	if config.Address == "" {
		return nil, fmt.Errorf("milvus address is required")
	}
	// 设置默认度量类型
	metricType := entity.L2
	if config.MetricType == "IP" {
		metricType = entity.IP
	} else if config.MetricType == "HAMMING" {
		metricType = entity.HAMMING
	} else if config.MetricType == "JACCARD" {
		metricType = entity.JACCARD
	}

	cli, err := client.NewClient(ctx, client.Config{
		Address: config.Address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to milvus: %v", err)
	}

	emb := NewSiliconFlowEmbedder(config.SiliconFlowAPIKey, config.ModelName)
	// 检查集合是否存在
	exists, err := cli.HasCollection(ctx, config.Collection)
	if err != nil {
		return nil, fmt.Errorf("failed to check collection existence: %v", err)
	}
	if !exists {
		return nil, fmt.Errorf("collection %s does not exist", config.Collection)
	}
	// 加载集合(异步加载)
	err = cli.LoadCollection(ctx, config.Collection, false)
	if err != nil {
		return nil, fmt.Errorf("failed to load collection: %v", err)
	}
	// 检查加载状态
	loadState, err := cli.GetLoadState(ctx, config.Collection, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get load state: %v", err)
	}
	if loadState != entity.LoadStateLoaded {
		return nil, fmt.Errorf("collection %s is not loaded, current state: %v", config.Collection, loadState)
	}

	retriever, err := milvus.NewRetriever(ctx, &milvus.RetrieverConfig{
		Client:         cli,
		Collection:     config.Collection,
		VectorField:    config.VectorField,
		TopK:           config.TopK,
		ScoreThreshold: config.ScoreThreshold,
		Embedding:      emb,
		MetricType:     metricType,
		VectorConverter: func(ctx context.Context, vectors [][]float64) ([]entity.Vector, error) {
			// 将float64转换为float32
			result := make([]entity.Vector, len(vectors))
			for i, v := range vectors {
				float32Vec := make([]float32, len(v))
				for j, val := range v {
					float32Vec[j] = float32(val)
				}
				result[i] = entity.FloatVector(float32Vec)
			}
			return result, nil
		},
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
	tool := mcp.NewTool("vector_search",
		mcp.WithDescription("搜索模块功能及其对应的入口URL（如门票、酒店预订等），输入用户问题即可返回匹配的功能入口URL"),
		mcp.WithString("query", mcp.Required(), mcp.Description("检索查询文本，系统将分析意图并匹配最相关的资源")),
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

	// 使用SearchQueryOptFn指定输出字段
	// 移除反射方式，直接调用检索
	documents, err := t.retriever.Retrieve(ctx, query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("检索失败: %v", err)), nil
	}

	// 处理结果，提取需要的字段
	var results []map[string]interface{}
	for _, doc := range documents {
		result := make(map[string]interface{})
		// 假设文档有Content和MetaData字段
		result["content"] = doc.Content
		if doc.MetaData != nil {
			// 提取需要的元数据字段
			if name, ok := doc.MetaData["name"]; ok {
				result["name"] = name
			}
			if url, ok := doc.MetaData["url"]; ok {
				result["url"] = url
			}
		}
		results = append(results, result)
	}

	output, _ := json.Marshal(documents)
	return mcp.NewToolResultText(string(output)), nil
}

// Name 实现工具接口
func (t *MilvusTool) Name() string {
	return "milvus"
}
