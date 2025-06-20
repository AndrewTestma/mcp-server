package toolchain

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
)

// ToolChainService 工具联动服务
type ToolChainService struct {
	searchTool  SearchTool  // 搜索工具（通过接口注入）
	browserTool BrowserTool // 浏览器工具（通过接口注入）
}

// NewToolChainService 创建工具联动服务实例
func NewToolChainService(searchTool SearchTool, browserTool BrowserTool) *ToolChainService {
	return &ToolChainService{
		searchTool:  searchTool,
		browserTool: browserTool,
	}
}

// ExecuteChain 执行工具联动流程（核心方法）

func (s *ToolChainService) ExecuteChain(ctx context.Context, query, extractGoal string) (string, error) {
	// 1. 执行搜索工具
	searchResult, err := s.search(ctx, query)
	if err != nil {
		return "", fmt.Errorf("搜索失败: %w", err)
	}
	// 步骤 2：解析搜索结果，获取目标 URL
	targetURL, err := s.parseSearchResult(searchResult)
	if err != nil {
		return "", fmt.Errorf("解析搜索结果失败: %w", err)
	}
	// 步骤 3：导航到目标 URL
	tabID, err := s.navigateToURL(ctx, targetURL)
	if err != nil {
		return "", fmt.Errorf("导航失败: %w", err)
	}

	// 步骤 4：提取目标内容
	extractedContent, err := s.extractContent(ctx, tabID, extractGoal)
	if err != nil {
		return "", fmt.Errorf("内容提取失败: %w", err)
	}

	return extractedContent, nil

}

func (s *ToolChainService) search(ctx context.Context, query string) (string, error) {
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "web_search",
			Arguments: map[string]interface{}{
				"query": query,
				"limit": 5,
			},
		},
	}
	result, err := s.searchTool.Execute(ctx, req)
	if err != nil {
		return "", fmt.Errorf("搜索工具执行失败: %w", err)
	}
	if result.IsError {
		return "", fmt.Errorf("搜索工具返回错误: %s", result.Content)
	}
	// 处理返回的Content数组
	if len(result.Content) == 0 {
		return "", fmt.Errorf("搜索结果为空")
	}

	// 假设第一个Content是文本类型
	content, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		return "", fmt.Errorf("不支持的返回内容类型")
	}
	// 解析搜索结果
	var searchResults struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Snippet string `json:"snippet"`
		} `json:"results"`
	}

	if err := json.Unmarshal([]byte(content.Text), &searchResults); err != nil {
		return "", fmt.Errorf("解析搜索结果失败: %w", err)
	}

	formattedResults, err := json.Marshal(searchResults)
	if err != nil {
		return "", fmt.Errorf("格式化搜索结果失败: %w", err)
	}

	return string(formattedResults), nil
}

// 解析搜索结果（独立封装）
func (s *ToolChainService) parseSearchResult(searchResult string) (string, error) {
	var results struct {
		Results []struct {
			URL string `json:"url"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(searchResult), &results); err != nil {
		return "", err
	}
	if len(results.Results) == 0 {
		return "", fmt.Errorf("搜索结果为空")
	}
	return results.Results[0].URL, nil
}

// 导航到 URL（独立封装）
func (s *ToolChainService) navigateToURL(ctx context.Context, url string) (string, error) {
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"action": "go_to_url",
				"url":    url,
			},
		},
	}
	result, err := s.browserTool.Execute(ctx, req)
	if err != nil {
		return "", err
	}
	if result.IsError {
		return "", fmt.Errorf("导航失败: %s", result.Content)
	}

	var navigateData struct{ TabID string }
	content, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		return "", fmt.Errorf("不支持的返回内容类型")
	}
	if err := json.Unmarshal([]byte(content.Text), &navigateData); err != nil {
		return "", err
	}
	return navigateData.TabID, nil
}

// 提取内容（独立封装）
func (s *ToolChainService) extractContent(ctx context.Context, tabID, goal string) (string, error) {
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"action": "extract_content",
				"tab_id": tabID,
				"goal":   goal,
			},
		},
	}
	result, err := s.browserTool.Execute(ctx, req)
	if err != nil {
		return "", err
	}
	if result.IsError {
		return "", fmt.Errorf("内容提取失败: %s", result.Content)
	}
	content, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		return "", fmt.Errorf("不支持的返回内容类型")
	}
	return content.Text, nil
}
