package config

import (
	"encoding/json"
	"fmt"
	"mcp-server/internal/tools/browseruse"
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

	return &cfg, nil
}
