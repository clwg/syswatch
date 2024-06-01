package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/clwg/syswatch/data"
	filewriter "github.com/clwg/syswatch/logging"

	syswatch "github.com/clwg/syswatch/internal"
	pb "github.com/clwg/syswatch/proto"
)

var (
	tls            = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	certFile       = flag.String("cert_file", "", "The TLS cert file")
	keyFile        = flag.String("key_file", "", "The TLS key file")
	port           = flag.Int("port", 51001, "The server port")
	httpPort       = flag.Int("http_port", 8084, "The HTTP server port")
	filenamePrefix = flag.String("log_filename_prefix", "syswatch", "The prefix for the log file name")
	logDir         = flag.String("log_dir", "./logs", "The directory for the log files")
	maxLines       = flag.Int("log_max_lines", 1000, "The maximum number of lines per log file")
	rotationTime   = flag.Duration("log_rotation_time", 600*time.Second, "The rotation time for the log files")
)

func main() {
	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	LoggingConfig := filewriter.LoggerConfig{
		FilenamePrefix: *filenamePrefix,
		LogDir:         *logDir,
		MaxLines:       *maxLines,
		RotationTime:   *rotationTime,
	}

	fileLogger, err := filewriter.NewLogger(LoggingConfig)
	if err != nil {
		panic(err)
	}

	var opts []grpc.ServerOption
	if *tls {
		if *certFile == "" {
			*certFile = data.Path("data/x509/server_cert.pem")
		}
		if *keyFile == "" {
			*keyFile = data.Path("data/x509/server_key.pem")
		}
		creds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
		if err != nil {
			log.Fatalf("Failed to generate credentials: %v", err)
		}
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	}

	grpcServer := grpc.NewServer(opts...)
	server := syswatch.InitializeSysWatchServer(fileLogger)

	pb.RegisterSysWatchServer(grpcServer, server)

	go syswatch.StartHTTPServer(server, *httpPort)

	log.Printf("Server listening at %v", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
