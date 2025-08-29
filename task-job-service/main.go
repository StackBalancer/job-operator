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
	defaultProcessing = 5      // Default job processing delay in seconds
	counter           = 0
)

func init() {
	// Read environment variables for configuration
	if port, exists := os.LookupEnv("SERVERPORT"); exists {
		serverPort = port
	}
	if delay, exists := os.LookupEnv("JOBPROCESSINGTIME"); exists {
		if parsedDelay, err := strconv.Atoi(delay); err == nil {
			defaultProcessing = parsedDelay
		}
	}
}

func jobHandler(w http.ResponseWriter, r *http.Request) {
	// Get job parameters from query string
	jobID := r.URL.Query().Get("param1")           // Job ID
	processingStr := r.URL.Query().Get("param2")  // Processing time for this job

	if jobID == "" {
		http.Error(w, "Missing job parameter: param1", http.StatusBadRequest)
		return
	}

	// Convert processing time to int, fallback to default
	processingTime := defaultProcessing
	if processingStr != "" {
		if pt, err := strconv.Atoi(processingStr); err == nil {
			processingTime = pt
		}
	}

	log.Printf("Executing job %s for %d seconds", jobID, processingTime)
	time.Sleep(time.Duration(processingTime) * time.Second)
	fmt.Fprintf(w, "Job with ID '%s' completed at %v", jobID, time.Now())
}

func heartbeat() {
	for {
		counter++
		log.Printf("Heartbeat %d: service running...", counter)
		time.Sleep(10 * time.Second)
	}
}

func main() {
	go heartbeat() // Start periodic task in background

	http.HandleFunc("/task-job", jobHandler)

	log.Printf("Starting task job server on port %s...", serverPort)
	log.Fatal(http.ListenAndServe(":"+serverPort, nil))
}
