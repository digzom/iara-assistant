package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type ChromaDBClient struct {
	baseURL    string
	httpClient *http.Client
}

type Collection struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Metadata map[string]string `json:"metadata"`
}

type CreateCollectionRequest struct {
	Name     string            `json:"name"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type AddRequest struct {
	IDs        []string                 `json:"ids"`
	Embeddings [][]float32              `json:"embeddings"`
	Documents  []string                 `json:"documents"`
	Metadatas  []map[string]interface{} `json:"metadatas,omitempty"`
}

type QueryRequest struct {
	QueryEmbeddings [][]float32 `json:"query_embeddings"`
	NResults        int         `json:"n_results,omitempty"`
}

type QueryResponse struct {
	IDs       [][]string                 `json:"ids"`
	Distances [][]float32                `json:"distances"`
	Documents [][]string                 `json:"documents"`
	Metadatas [][]map[string]interface{} `json:"metadatas"`
}

func (c *ChromaDBClient) Heartbeat() error {
	url := fmt.Sprintf("%s/api/v2/heartbeat", c.baseURL)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to make heartbeat request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("chromadb not ready, heartbeat status: %s", resp.Status)
	}
	return nil
}

func NewChromaDBClientWithRetry(baseURL string, maxRetries int, retryDelay time.Duration) (*ChromaDBClient, error) {
	var client *ChromaDBClient
	var err error

	log.Println("Attempting to connect to ChromaDB...")

	for attempt := 1; attempt <= maxRetries; attempt++ {
		c := &ChromaDBClient{
			baseURL: baseURL,
			httpClient: &http.Client{
				Timeout: 10 * time.Second,
			},
		}

		err = c.Heartbeat()
		if err == nil {
			log.Println("Successfully connected to ChromaDB!")
			client = c
			break
		}

		log.Printf("Failed to connect (attempt %d/%d): %v", attempt, maxRetries, err)
		if attempt < maxRetries {
			log.Printf("Retrying in %v...", retryDelay)
			time.Sleep(retryDelay)
		}
	}

	if client == nil {
		return nil, fmt.Errorf("could not connect to ChromaDB at %s after %d attempts", baseURL, maxRetries)
	}

	return client, nil
}

func (c *ChromaDBClient) CreateCollection(name string) error {
	reqBody := CreateCollectionRequest{
		Name: name,
		Metadata: map[string]string{
			"description": "Iara assistant facts storage",
		},
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	url := fmt.Sprintf("%s/api/v2/collections", c.baseURL)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		return fmt.Errorf("failed to create collection, status: %s", resp.Status)
	}
	return nil
}

func (c *ChromaDBClient) AddDocument(collectionName, id, document string, embedding []float32, metadata map[string]interface{}) error {
	reqBody := AddRequest{
		IDs:        []string{id},
		Embeddings: [][]float32{embedding},
		Documents:  []string{document},
		Metadatas:  []map[string]interface{}{metadata},
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	// A API para adicionar documentos é /api/v2/collections/{collection_name}/add
	url := fmt.Sprintf("%s/api/v2/collections/%s/add", c.baseURL, collectionName)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to add document, status: %s", resp.Status)
	}
	return nil
}

func (c *ChromaDBClient) QuerySimilar(collectionName string, queryEmbedding []float32, nResults int) (*QueryResponse, error) {
	if nResults == 0 {
		nResults = 3
	}
	reqBody := QueryRequest{
		QueryEmbeddings: [][]float32{queryEmbedding},
		NResults:        nResults,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v2/collections/%s/query", c.baseURL, collectionName)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to query collection, status: %s", resp.Status)
	}
	var queryResp QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&queryResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &queryResp, nil
}
