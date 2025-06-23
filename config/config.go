package config

import (
	"encoding/json"
	"fmt"
	"mcp-server/internal/tools/browseruse"
	"mcp-server/internal/tools/milvus"
	"mcp-server/internal/tools/search"
	"os"
)

// AppConfig 应用整体配置
type AppConfig struct {
	ServerPort string         `json:"server_port"`
	Tools      map[string]any `json:"tools"` // 工具配置（动态类型）
}

// LoadConfig 加载配置文件
func LoadConfig(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file failed: %w", err)
	}

	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config failed: %w", err)
	}

	// 类型转换工具配置（根据实际工具补充）
	browserCfg := &browseruse.Config{}
	if err := json.Unmarshal(data, browserCfg); err != nil {
		return nil, fmt.Errorf("parse browser config failed: %w", err)
	}
	cfg.Tools["browseruse"] = browserCfg

	searchCfg := &search.Config{}
	if err := json.Unmarshal(data, searchCfg); err != nil {
		return nil, fmt.Errorf("parse search config failed: %w", err)
	}
	cfg.Tools["web_search"] = searchCfg
	// 添加Milvus配置解析
	var rawConfig map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawConfig); err != nil {
		return nil, fmt.Errorf("parse raw config failed: %w", err)
	}

	if toolsRaw, ok := rawConfig["tools"]; ok {
		var toolsMap map[string]json.RawMessage
		if err := json.Unmarshal(toolsRaw, &toolsMap); err != nil {
			return nil, fmt.Errorf("parse tools config failed: %w", err)
		}

		// 单独解析milvus配置
		if milvusRaw, ok := toolsMap["milvus"]; ok {
			milvusCfg := &milvus.Config{}
			if err := json.Unmarshal(milvusRaw, milvusCfg); err != nil {
				return nil, fmt.Errorf("parse milvus config failed: %w", err)
			}
			cfg.Tools["milvus"] = milvusCfg
		}
	}

	return &cfg, nil
}
