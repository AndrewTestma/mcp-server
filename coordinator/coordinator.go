package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/mcp"
	"mcp-server/internal/tool"
	"strings"
)

// ToolCallStep 模型生成的工具调用步骤
type ToolCallStep struct {
	ToolName string                 `json:"tool_name"` // 工具名称（如"web_search"）
	Params   map[string]interface{} `json:"params"`    // 工具参数
	Reason   string                 `json:"reason"`    // 调用原因（用于上下文追溯）
}

// ToolCallPlan 模型生成的工具调用计划
type ToolCallPlan struct {
	Steps []ToolCallStep `json:"steps"` // 工具调用步骤列表（按顺序执行）
}

// Coordinator 流程协调器
type Coordinator struct {
	model       *openai.ChatModel // 大模型实例（用于生成工具调用计划）
	toolManager *tool.ToolManager // 工具管理器（用于获取工具实例）
	ctx         context.Context   // 上下文
}

// NewCoordinator 创建协调器实例
func NewCoordinator(ctx context.Context, model *openai.ChatModel, toolManager *tool.ToolManager) *Coordinator {
	return &Coordinator{
		model:       model,
		toolManager: toolManager,
		ctx:         ctx,
	}
}

// Run 执行用户查询的完整处理流程（模型驱动工具调用）
func (c *Coordinator) Run(userQuery string) (string, error) {
	// 1. 生成工具调用计划（通过大模型分析用户查询，决定需要调用的工具及顺序）
	plan, err := c.generateToolCallPlan(userQuery)
	if err != nil {
		return "", fmt.Errorf("生成工具调用计划失败: %v", err)
	}

	// 2. 按顺序执行工具调用步骤
	var finalResult string
	for i, step := range plan.Steps {
		// 2.1 获取工具实例
		tool, err := c.toolManager.GetTool(step.ToolName)
		if err != nil {
			return "", fmt.Errorf("步骤%d: 工具 %s 不存在: %v", i+1, step.ToolName, err)
		}

		// 2.2 构造工具调用请求
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: step.Params,
			},
		}

		// 2.3 执行工具调用
		result, err := tool.Execute(c.ctx, req)
		if err != nil || result.IsError {
			return "", fmt.Errorf("步骤%d: 工具 %s 执行失败: %v / %s", i+1, step.ToolName, err, result.Content)
		}

		// 2.4 记录中间结果（可选：将结果追加到对话上下文中，供模型后续步骤使用）
		content, ok := result.Content[0].(mcp.TextContent)
		if !ok {
			return "", fmt.Errorf("不支持的返回内容类型")
		}
		finalResult = content.Text
	}

	return finalResult, nil
}

// generateToolCallPlan 调用大模型生成工具调用计划
func (c *Coordinator) generateToolCallPlan(userQuery string) (*ToolCallPlan, error) {
	// 构造模型输入（包含工具列表描述，让模型知道可用工具）
	messages := []*schema.Message{
		{
			Role:    schema.System,
			Content: c.buildToolDescriptionPrompt(), // 工具描述提示词（告知模型可用工具的功能和参数）
		},
		{
			Role:    schema.User,
			Content: fmt.Sprintf("用户查询：%s\n请生成一个工具调用计划（JSON格式），描述需要调用哪些工具、参数及原因。", userQuery),
		},
	}

	// 调用大模型生成工具调用计划
	resp, err := c.model.Generate(c.ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("模型调用失败: %v", err)
	}

	// 解析模型输出（假设模型返回JSON格式的工具调用计划）
	if resp == nil || resp.Content == "" {
		return nil, fmt.Errorf("模型返回内容为空")
	}
	plan := &ToolCallPlan{}
	if err := json.Unmarshal([]byte(resp.Content), plan); err != nil {
		return nil, fmt.Errorf("解析工具调用计划失败: %v\n模型输出: %s", err, resp.Content)
	}

	return plan, nil
}

// buildToolDescriptionPrompt 构造工具描述提示词（告知模型可用工具的功能和参数要求）
func (c *Coordinator) buildToolDescriptionPrompt() string {
	var toolDescriptions []string
	c.toolManager.Mu().RLock()
	defer c.toolManager.Mu().RUnlock()

	// 遍历所有已注册工具，生成工具描述
	for _, tool := range c.toolManager.Tools() {
		descriptor := tool.GetDescriptor()
		params := make([]string, 0)

		// 从 InputSchema.Properties 获取参数信息
		for paramName, prop := range descriptor.InputSchema.Properties {
			propMap, ok := prop.(map[string]any)
			if !ok {
				continue
			}

			paramDesc := fmt.Sprintf("%s (%s)",
				paramName,
				propMap["type"])

			if desc, ok := propMap["description"].(string); ok {
				paramDesc += ": " + desc
			}

			params = append(params, paramDesc)
		}

		toolDesc := fmt.Sprintf("- 工具名: %s\n  描述: %s\n  参数要求:\n    %s",
			descriptor.Name,
			descriptor.Description,
			strings.Join(params, "\n    "),
		)
		toolDescriptions = append(toolDescriptions, toolDesc)
	}

	return fmt.Sprintf("你可以使用以下工具来回答用户问题：\n%s\n请根据用户问题生成工具调用计划（JSON格式）。",
		strings.Join(toolDescriptions, "\n\n"))
}
