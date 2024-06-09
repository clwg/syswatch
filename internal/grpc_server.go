package syswatch

import (
	"context"
	"log"
	"sync"

	logwriter "github.com/clwg/go-rotating-logger"
	pb "github.com/clwg/syswatch/proto"
	"github.com/google/uuid"
)

type connectionStream struct {
	stream pb.SysWatch_BidirectionalStreamPayloadServer
	active bool
}

type SysWatchServer struct {
	pb.UnimplementedSysWatchServer
	clients sync.Map
	stopCh  chan struct{}
	logger  *logwriter.Logger
}

func InitializeSysWatchServer(logger *logwriter.Logger) *SysWatchServer {
	return &SysWatchServer{
		stopCh: make(chan struct{}),
		logger: logger,
	}
}

func (s *SysWatchServer) GenerateUUID(context.Context, *pb.Empty) (*pb.UUIDResponse, error) {
	uuid := uuid.New().String()
	return &pb.UUIDResponse{Uuid: uuid}, nil
}

func (s *SysWatchServer) BidirectionalStreamPayload(stream pb.SysWatch_BidirectionalStreamPayloadServer) error {
	var connID string

	for {
		in, err := stream.Recv()
		if err != nil {
			log.Printf("Failed to receive a message: %v", err)
			break
		}

		if connID == "" {
			connID = in.GetConnectionId()
			s.clients.Store(connID, &connectionStream{stream: stream, active: true})
			log.Printf("Server registered new client with connection ID: %s", connID)
		}

		source := in.GetSource()
		logData := connID + " | " + source + " | " + in.GetPayload()

		s.logger.Log(logData)
	}

	s.clients.Delete(connID)
	log.Printf("Client disconnected with connection ID: %s", connID)
	return nil
}

func (s *SysWatchServer) directMessage(payload, senderID string) {
	s.clients.Range(func(key, value interface{}) bool {
		connID := key.(string)
		connStream := value.(*connectionStream)

		if connID != senderID && connStream.active {
			out := &pb.ResponseMessage{Payload: payload}
			if err := connStream.stream.Send(out); err != nil {
				log.Printf("Failed to send a message to connection ID %s: %v", connID, err)
				connStream.active = false // Mark as inactive on failure
			}
		}
		return true
	})
}

func (s *SysWatchServer) getActiveConnections() []string {
	var connections []string
	s.clients.Range(func(key, value interface{}) bool {
		connID := key.(string)
		connStream := value.(*connectionStream)
		if connStream.active {
			connections = append(connections, connID)
		}
		return true
	})
	return connections
}
