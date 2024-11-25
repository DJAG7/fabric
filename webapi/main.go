package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
)

// CommandRequest represents the structure of the incoming JSON request
type CommandRequest struct {
	Input       string `json:"input"`       // Input text or file path
	PatternName string `json:"pattern_name"` // Pattern name for Fabric
	Flag        string `json:"flag,omitempty"` // Optional flag (e.g., -y for YouTube)
	Model       string `json:"model,omitempty"` // Optional model (e.g., DALL-E)
}

// CommandResponse represents the response structure
type CommandResponse struct {
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
}

// isFile checks if the provided path is a file
func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// runFabricCommand executes the Fabric command based on the input
func runFabricCommand(req CommandRequest) (string, error) {
	var cmd *exec.Cmd

	// Construct the command based on input
	if req.Flag == "" && req.Model == "" {
		// Basic command
		cmd = exec.Command("fabric", req.Input, "-p", req.PatternName)
	} else if req.Flag != "" {
		// Command with a flag
		cmd = exec.Command("fabric", req.Flag, req.Input, "-p", req.PatternName)
	} else if req.Model != "" {
		// Command with a model
		cmd = exec.Command("fabric", req.Input, "-p", req.PatternName, "-m", req.Model)
	} else {
		// Multiline input using echo
		cmd = exec.Command("sh", "-c", fmt.Sprintf("echo \"%s\" | fabric -p %s", req.Input, req.PatternName))
	}

	// Log the command being executed
	log.Println("Executing command:", cmd.Args)

	// Execute the command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error: %v, output: %s", err, string(output))
	}

	return string(output), nil
}

// withCORS is a middleware to handle CORS
func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Allow all origins (for development purposes)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// handleRunCommand handles POST requests to run Fabric commands
func handleRunCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Parse the JSON body
	var req CommandRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Input == "" || req.PatternName == "" {
		http.Error(w, "Missing required fields: input and pattern_name", http.StatusBadRequest)
		return
	}

	// Run the Fabric command
	output, err := runFabricCommand(req)
	var response CommandResponse
	if err != nil {
		response = CommandResponse{
			Output: "",
			Error:  err.Error(),
		}
	} else {
		response = CommandResponse{
			Output: output,
			Error:  "",
		}
	}

	// Send the response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// main sets up the HTTP server
func main() {
	log.Println("Starting API server...")
	http.HandleFunc("/run-command", withCORS(handleRunCommand)) // Apply CORS middleware

	port := ":8080"
	log.Printf("API server is starting on port %s\n", port)

	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatalf("Failed to start server: %v\n", err)
	}
}

