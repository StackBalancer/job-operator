package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

var (
	serverPort        = "8080" // Default port
	jobProcessingTime = 5      // Default job processing delay in seconds
)

func init() {
	// Read environment variables for configuration
	if port, exists := os.LookupEnv("SERVERPORT"); exists {
		serverPort = port
	}
	if delay, exists := os.LookupEnv("JOBPROCESSINGTIME"); exists {
		if parsedDelay, err := strconv.Atoi(delay); err == nil {
			jobProcessingTime = parsedDelay
		}
	}
}

func jobHandler(w http.ResponseWriter, r *http.Request) {
	// Get job parameters from query string
	param1 := r.URL.Query().Get("param1") // e.g., job ID
	param2 := r.URL.Query().Get("param2") // e.g., additional parameter

	// If param1 is missing, return an error
	if param1 == "" {
		http.Error(w, "Missing job parameter: param1", http.StatusBadRequest)
		return
	}
	// Log the received parameters (for debugging)
	log.Printf("Received job with param1: %s and param2: %s", param1, param2)

	// Simulate job processing delay
	time.Sleep(time.Duration(jobProcessingTime) * time.Second)

	// Respond with job completion message
	fmt.Fprintf(w, "HPC Job with ID '%s' completed at %v", param1, time.Now())
}

func main() {
	http.HandleFunc("/hpc-job", jobHandler)

	log.Printf("Starting mock HPC job server on port %s...", serverPort)
	log.Fatal(http.ListenAndServe(":"+serverPort, nil))
}
