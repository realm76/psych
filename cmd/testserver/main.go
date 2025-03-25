package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/", randomStatusHandler)
	fmt.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Error starting server:", err)
	}
}

func randomStatusHandler(w http.ResponseWriter, r *http.Request) {
	statusCodes := []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusAccepted,
		http.StatusNoContent,
		http.StatusMovedPermanently,
		http.StatusFound,
		http.StatusNotModified,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusInternalServerError,
		http.StatusNotImplemented,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
	}

	rand.New(rand.NewSource(time.Now().UnixNano()))
	randomStatus := statusCodes[rand.Intn(len(statusCodes))]

	randomDelay := time.Duration(rand.Intn(5)) * time.Second
	time.Sleep(randomDelay)

	w.WriteHeader(randomStatus)
}
