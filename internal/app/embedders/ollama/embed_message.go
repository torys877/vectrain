package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (o *Ollama) Embed(ctx context.Context, message string) ([]float32, error) {
	reqBody := EmbeddingRequest{
		Model:  "nomic-embed-text",
		Prompt: message,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", o.endpoint, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{
		Timeout: o.MaxResponseTimeoutDuration,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-OK response status: %d, body: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var embedResp EmbeddingResponse
	if err := json.Unmarshal(respBody, &embedResp); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v, body: %s", err, string(respBody))
	}

	return embedResp.Embedding, nil
}
