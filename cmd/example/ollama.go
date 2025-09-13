package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	ollamaEndpoint = "http://localhost:11434/api/embeddings"
)

type EmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type EmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

// GetEmbedding sends a request to Ollama API to get embeddings for the given text
func GetEmbedding(text string) ([]float64, error) {
	// Create request payload
	reqBody := EmbeddingRequest{
		Model:  "nomic-embed-text",
		Prompt: text,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", ollamaEndpoint, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
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

func main() {
	// Example text to get embeddings for
	text := "\"Title: Four Rooms\\nOverview: It's Ted the Bellhop's first night on the job...and the hotel's very unusual guests are about to place him in some outrageous predicaments. It seems that this evening's room service is serving up one unbelievable happening after another.\\nGenres: Comedy\\nTagline: Twelve outrageous guests. Four scandalous requests. And one lone bellhop, in his first day on the job, who's in for the wildest New year's Eve of his life.\\n Keywords: hotel, new year's eve, witch, bet, sperm, hotel room, anthology, los angeles, california, hoodlum, multiple storylines, woman director\\n\""

	fmt.Printf("Getting embeddings for text: %s\n", text)

	// Get embeddings
	embeddings, err := GetEmbedding(text)
	if err != nil {
		fmt.Printf("Error getting embeddings: %v\n", err)
		return
	}

	// Print the first 5 values of the embedding vector (or fewer if less than 5)
	fmt.Println("Embedding vector (first few values):")
	vectorLen := len(embeddings)
	printLen := 50
	if vectorLen < printLen {
		printLen = vectorLen
	}

	for i := 0; i < printLen; i++ {
		fmt.Printf("  [%d]: %f\n", i, embeddings[i])
	}
	fmt.Printf("Total embedding dimensions: %d\n", vectorLen)
}
