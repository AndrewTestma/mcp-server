package milvus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudwego/eino/components/embedding"
	"io"
	"net/http"
)

type SiliconFlowEmbedder struct {
	apiKey    string
	modelName string
}

func (e *SiliconFlowEmbedder) EmbedStrings(ctx context.Context, texts []string, opts ...embedding.Option) ([][]float64, error) {
	//TODO implement me
	results := make([][]float64, len(texts))
	for i, text := range texts {
		vec, err := e.Embed(ctx, text)
		if err != nil {
			return nil, err
		}
		// 将float32转为float64
		results[i] = make([]float64, len(vec))
		for j, v := range vec {
			results[i][j] = float64(v)
		}
	}
	return results, nil
}

func NewSiliconFlowEmbedder(apiKey, modelName string) *SiliconFlowEmbedder {
	return &SiliconFlowEmbedder{
		apiKey:    apiKey,
		modelName: modelName,
	}
}

func (e *SiliconFlowEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	requestBody := map[string]interface{}{
		"model":           e.modelName,
		"input":           text,
		"encoding_format": "float",
	}

	jsonBody, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", "https://api.siliconflow.cn/v1/embeddings", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no embedding data in response")
	}

	return result.Data[0].Embedding, nil
}
