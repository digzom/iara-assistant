package handlers

import (
	"encoding/json"
	"iara-assistant/services"
	"log"
	"net/http"
	"os"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

var ragService *services.RAGService

func init() {
	googleAPIKey := os.Getenv("GOOGLE_API_KEY")
	chromaDBURL := os.Getenv("CHROMADB_URL")
	if chromaDBURL == "" {
		chromaDBURL = "http://chromadb:8000"
	}

	ragService = services.NewRAGService(googleAPIKey, chromaDBURL)
}

func MessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req services.MessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding message request: %v", err)
		sendError(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	response, err := ragService.ProcessMessage(req)
	if err != nil {
		log.Printf("Error processing message: %v", err)
		sendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if !response.Success {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	json.NewEncoder(w).Encode(response)
}

func LearnHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req services.LearnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding learn request: %v", err)
		sendError(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	response, err := ragService.LearnFact(req)
	if err != nil {
		log.Printf("Error learning fact: %v", err)
		sendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if !response.Success {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	json.NewEncoder(w).Encode(response)
}

func sendError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
	}

	json.NewEncoder(w).Encode(response)
}

