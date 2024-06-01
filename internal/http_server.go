package syswatch

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	pb "github.com/clwg/syswatch/proto"
)

func StartHTTPServer(s *SysWatchServer, port int) {
	http.HandleFunc("/connections", s.listConnections)
	http.HandleFunc("/send", s.apiSendMessage)
	http.HandleFunc("/broadcast", s.apiBroadcastMessage)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

// This is the HTTP handler that uses getActiveConnections
func (s *SysWatchServer) listConnections(w http.ResponseWriter, r *http.Request) {
	connections := s.getActiveConnections()
	json.NewEncoder(w).Encode(connections)
}

func (s *SysWatchServer) apiSendMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID      string `json:"id"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if req.ID == "" || req.Message == "" {
		http.Error(w, "Missing id or message in request body", http.StatusBadRequest)
		return
	}

	value, ok := s.clients.Load(req.ID)
	if !ok {
		http.Error(w, "Connection ID not found", http.StatusNotFound)
		return
	}

	connStream := value.(*connectionStream)
	out := &pb.ResponseMessage{Payload: req.Message}
	if err := connStream.stream.Send(out); err != nil {
		http.Error(w, "Failed to send message", http.StatusInternalServerError)
		return
	}

	response := struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}{
		Status:  "success",
		Message: "Message sent successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (s *SysWatchServer) apiBroadcastMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Message string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, "Missing message in request body", http.StatusBadRequest)
		return
	}

	// This is terrible logic
	s.directMessage(req.Message, "") // Pass empty senderID to broadcast to all clients

	response := struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}{
		Status:  "success",
		Message: "Broadcast message sent successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
