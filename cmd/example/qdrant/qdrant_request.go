package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/qdrant/go-client/qdrant"
)

const (
	ollamaEndpoint = "http://localhost:11434/api/embeddings"
	qdrantHost     = "localhost"
	qdrantPort     = 6334
	collectionName = "production1"
)

type EmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type EmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

// GetEmbedding sends a request to Ollama API to get embeddings for the given text
func GetEmbedding(text string) ([]float32, error) {
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

// SearchSimilarInQdrant searches for similar vectors in Qdrant
func SearchSimilarInQdrant(ctx context.Context, vector []float32, limit uint64) ([]*qdrant.ScoredPoint, error) {
	// Create Qdrant client
	qdrantClientConfig := qdrant.Config{
		Host: qdrantHost,
		Port: qdrantPort,
	}

	client, err := qdrant.NewClient(&qdrantClientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client: %v", err)
	}
	defer client.Close()

	// Create search request
	searchRequest := &qdrant.SearchPoints{
		CollectionName: collectionName,
		Vector:         vector,
		Limit:          limit,
		WithPayload: &qdrant.WithPayloadSelector{
			SelectorOptions: &qdrant.WithPayloadSelector_Enable{
				Enable: true,
			},
		},
		WithVectors: &qdrant.WithVectorsSelector{
			SelectorOptions: &qdrant.WithVectorsSelector_Enable{
				Enable: true,
			},
		},
	}

	// Execute search
	response, err := client.GetPointsClient().Search(ctx, searchRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to search points: %v", err)
	}

	return response.Result, nil
}

// PrintSearchResults prints the search results in a readable format
func PrintSearchResults(results []*qdrant.ScoredPoint) {
	fmt.Printf("Found %d results:\n\n", len(results))

	for i, result := range results {
		fmt.Printf("Result #%d (Score: %.4f):\n", i+1, result.Score)

		// Print payload fields
		fmt.Println("Payload:")
		for fieldName, value := range result.Payload {
			switch v := value.Kind.(type) {
			case *qdrant.Value_StringValue:
				fmt.Printf("  %s: %s\n", fieldName, v.StringValue)
			case *qdrant.Value_IntegerValue:
				fmt.Printf("  %s: %d\n", fieldName, v.IntegerValue)
			case *qdrant.Value_DoubleValue:
				fmt.Printf("  %s: %.4f\n", fieldName, v.DoubleValue)
			case *qdrant.Value_BoolValue:
				fmt.Printf("  %s: %t\n", fieldName, v.BoolValue)
			default:
				fmt.Printf("  %s: <unsupported type>\n", fieldName)
			}
		}
		fmt.Println()
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run qdrant_request.go 'your search query here'")
		os.Exit(1)
	}

	// Get query from command line arguments
	query := os.Args[1]
	fmt.Printf("Search query: %s\n\n", query)

	// Step 1: Get embedding for the query
	fmt.Println("1. Generating embeddings via Ollama...")
	embedding, err := GetEmbedding(query)
	if err != nil {
		fmt.Printf("Error getting embeddings: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Generated embedding with %d dimensions\n\n", len(embedding))

	// Step 2: Search Qdrant for similar vectors
	fmt.Println("2. Searching Qdrant for similar vectors...")
	ctx := context.Background()
	results, err := SearchSimilarInQdrant(ctx, embedding, 30) // Limit to 5 results
	if err != nil {
		fmt.Printf("Error searching Qdrant: %v\n", err)
		os.Exit(1)
	}

	// Step 3: Print results
	fmt.Println("3. Search Results:")
	PrintSearchResults(results)
}
