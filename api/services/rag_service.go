package services

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	"iara-assistant/clients"
)

const (
	CollectionName = "facts"
	MaxContextDocs = 3
)

type RAGService struct {
	googleClient *clients.GoogleAIClient
	chromaClient *clients.ChromaDBClient
}

type LearnRequest struct {
	Text   string `json:"text"`
	UserID string `json:"user_id,omitempty"`
}

type MessageRequest struct {
	Text   string `json:"text"`
	UserID string `json:"user_id,omitempty"`
}

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

func NewRAGService(googleAPIKey, chromaDBURL string) *RAGService {
	// Use retry client to wait for ChromaDB to be ready
	chromaClient, err := clients.NewChromaDBClientWithRetry(chromaDBURL, 10, 5*time.Second)
	if err != nil {
		log.Fatalf("Failed to connect to ChromaDB: %v", err)
	}

	service := &RAGService{
		googleClient: clients.NewGoogleAIClient(googleAPIKey),
		chromaClient: chromaClient,
	}

	if err := service.initializeCollection(); err != nil {
		log.Printf("Warning: Failed to initialize collection: %v", err)
	} else {
		log.Printf("Yeey: habemus collections!")
	}

	return service
}

func (s *RAGService) initializeCollection() error {
	return s.chromaClient.CreateCollection(CollectionName)
}

func (s *RAGService) LearnFact(req LearnRequest) (*Response, error) {
	if req.Text == "" {
		return &Response{
			Success: false,
			Error:   "Text cannot be empty",
		}, nil
	}

	embedding, err := s.googleClient.GenerateEmbedding(req.Text)
	if err != nil {
		return &Response{
			Success: false,
			Error:   "Failed to generate embedding",
		}, fmt.Errorf("embedding generation failed: %w", err)
	}

	docID := s.generateDocID(req.Text)
	metadata := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"user_id":   req.UserID,
		"type":      "fact",
	}

	err = s.chromaClient.AddDocument(CollectionName, docID, req.Text, embedding, metadata)
	if err != nil {
		return &Response{
			Success: false,
			Error:   "Failed to store fact",
		}, fmt.Errorf("document storage failed: %w", err)
	}

	return &Response{
		Success: true,
		Message: "Fact learned successfully!",
	}, nil
}

func (s *RAGService) ProcessMessage(req MessageRequest) (*Response, error) {
	if req.Text == "" {
		return &Response{
			Success: false,
			Error:   "Message cannot be empty",
		}, nil
	}

	queryEmbedding, err := s.googleClient.GenerateEmbedding(req.Text)
	if err != nil {
		return &Response{
			Success: false,
			Error:   "Failed to process message",
		}, fmt.Errorf("query embedding generation failed: %w", err)
	}

	similarDocs, err := s.chromaClient.QuerySimilar(CollectionName, queryEmbedding, MaxContextDocs)
	if err != nil {
		log.Printf("Warning: Failed to query similar documents: %v", err)
		return s.generateResponseWithoutContext(req.Text)
	}

	if len(similarDocs.Documents) == 0 || len(similarDocs.Documents[0]) == 0 {
		return s.generateResponseWithoutContext(req.Text)
	}

	contextDocs := similarDocs.Documents[0]
	augmentedPrompt := s.buildAugmentedPrompt(req.Text, contextDocs)

	response, err := s.googleClient.GenerateText(augmentedPrompt)
	if err != nil {
		return &Response{
			Success: false,
			Error:   "Failed to generate response",
		}, fmt.Errorf("text generation failed: %w", err)
	}

	return &Response{
		Success: true,
		Message: response,
	}, nil
}

func (s *RAGService) generateResponseWithoutContext(userQuery string) (*Response, error) {
	prompt := fmt.Sprintf(`You are Iara, a helpful personal AI assistant. The user is asking: "%s"

Please respond helpfully, but let them know that you don't have specific information about this topic in your personal knowledge base yet. You can suggest they teach you facts using the /learn command.`, userQuery)

	response, err := s.googleClient.GenerateText(prompt)
	if err != nil {
		return &Response{
			Success: false,
			Error:   "Failed to generate response",
		}, fmt.Errorf("text generation failed: %w", err)
	}

	return &Response{
		Success: true,
		Message: response,
	}, nil
}

func (s *RAGService) buildAugmentedPrompt(userQuery string, contextDocs []string) string {
	context := strings.Join(contextDocs, "\n\n")

	prompt := fmt.Sprintf(`You are Iara, a helpful personal AI assistant. You have access to the user's personal knowledge base.

CONTEXT FROM KNOWLEDGE BASE:
%s

USER QUESTION: %s

Please answer the user's question using the information from the knowledge base above. If the information is not sufficient to answer the question completely, be honest about what you know and don't know. Be conversational and helpful.`, context, userQuery)

	return prompt
}

func (s *RAGService) generateDocID(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text + time.Now().Format("2006-01-02")))
	return hex.EncodeToString(hasher.Sum(nil))
}

