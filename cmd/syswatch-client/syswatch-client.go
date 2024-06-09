package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/clwg/syswatch/data"
	pb "github.com/clwg/syswatch/proto"
	"github.com/clwg/syswatch/utils"
	"github.com/hpcloud/tail"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	tls                = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	caFile             = flag.String("ca_file", "", "The file containing the CA root cert file")
	serverAddr         = flag.String("addr", "localhost:51001", "The server address in the format of host:port")
	serverHostOverride = flag.String("server_host_override", "x.test.example.com", "The server name used to verify the hostname returned by the TLS handshake")
	filelist           = flag.String("filelist", "path/to/your/filelist.txt", "File containing the list of files to tail")
)

func main() {
	flag.Parse()
	// Set up a connection to the server.
	var opts []grpc.DialOption
	if *tls {
		if *caFile == "" {
			*caFile = data.Path("data/x509/ca_cert.pem")
		}
		creds, err := credentials.NewClientTLSFromFile(*caFile, *serverHostOverride)
		if err != nil {
			log.Fatalf("Failed to create TLS credentials: %v", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
		log.Println("TLS connection established")
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		log.Println("Insecure connection established")
	}

	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		log.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := pb.NewSysWatchClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	connUuid, err := client.GenerateUUID(ctx, &pb.Empty{})
	if err != nil {
		log.Fatalf("Could not generate UUID: %v", err)
	}
	connectionID := connUuid.GetUuid()

	stream, err := client.BidirectionalStreamPayload(context.Background())
	if err != nil {
		log.Fatalf("Failed to create stream: %v", err)
	}

	file, err := os.Open(*filelist)
	if err != nil {
		log.Fatalf("Failed to open filelist: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var wg sync.WaitGroup

	for scanner.Scan() {
		filename := scanner.Text()
		wg.Add(1)
		go func(filename string) {
			defer wg.Done()
			tailFileAndSendLogs(filename, connectionID, stream)
		}(filename)
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading filelist: %v", err)
	}

	go receiveServerMessages(stream, connectionID)

	// Block main from exiting
	wg.Wait()
}

func receiveServerMessages(stream pb.SysWatch_BidirectionalStreamPayloadClient, connectionID string) {
	for {
		response, err := stream.Recv()
		if err != nil {
			log.Fatalf("Failed to receive response: %v", err)
		}

		log.Printf("Received server message for connection %s: %s", connectionID, response.GetPayload())

		result, err := utils.ExecuteCommand(response.GetPayload())
		responsePayload := ""
		if err != nil {
			responsePayload = fmt.Sprintf("Error: %s", err)
			log.Println("Error:", err)
		} else {
			encodedResult := base64.StdEncoding.EncodeToString([]byte(result))
			responsePayload = encodedResult
		}

		streamResponse, err := json.Marshal(map[string]string{
			"connection_id":    connectionID,
			"response_payload": responsePayload,
			"source":           "direct",
		})
		if err != nil {
			log.Fatalf("Failed to marshal JSON: %v", err)
		}

		responseMessage := &pb.RequestMessage{
			Payload:      string(streamResponse),
			ConnectionId: connectionID,
			Source:       "direct",
		}

		if err := stream.Send(responseMessage); err != nil {
			log.Printf("Failed to send response message: %v", err)
			return
		}
	}
}

func tailFileAndSendLogs(filename, connectionID string, stream pb.SysWatch_BidirectionalStreamPayloadClient) {
	t, err := tail.TailFile(filename, tail.Config{Follow: true, ReOpen: true, Poll: true})
	if err != nil {
		log.Fatalf("Failed to start tailing file %s: %v", filename, err)
	}

	for line := range t.Lines {
		if line.Err != nil {
			log.Printf("Error reading line from %s: %v", filename, line.Err)
			continue
		}
		jsonMessage := &pb.RequestMessage{
			Payload:      line.Text,
			ConnectionId: connectionID,
			Source:       filename,
		}
		if err := stream.Send(jsonMessage); err != nil {
			log.Printf("Failed to send log message from %s: %v", filename, err)
			return
		}
		//log.Printf("Sent log message from %s: %s", filename, line.Text)
	}
}
