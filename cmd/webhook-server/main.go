package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	expectedUsername = "admin"
	expectedPassword = "secret"
)

func main() {
	// Open log file
	f, err := os.OpenFile("/tmp/webhooks.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
	}
	defer f.Close()

	logger := log.New(f, "", log.LstdFlags)

	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		// Check method
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Check basic auth
		authHeader := r.Header.Get("Authorization")
		if !validateBasicAuth(authHeader) {
			w.Header().Set("WWW-Authenticate", `Basic realm="Test"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Read body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Log the webhook
		logger.Printf("Received webhook at %s: %s\n", time.Now().Format(time.RFC3339), string(body))

		// Respond with success
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	fmt.Println("Starting webhook server on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func validateBasicAuth(authHeader string) bool {
	if authHeader == "" {
		return false
	}

	// Split "Basic <encoded>"
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Basic" {
		return false
	}

	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	// Split username:password
	credentials := strings.SplitN(string(decoded), ":", 2)
	if len(credentials) != 2 {
		return false
	}

	return credentials[0] == expectedUsername && credentials[1] == expectedPassword
}
