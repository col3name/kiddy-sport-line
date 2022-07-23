package main

import (
	"flag"
	"fmt"
	pb "github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/grpc/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"io"
	"net"
	"sync"
)

var (
	port = flag.Int("port", 50051, "The server port")
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedKiddyLineProcessorServer
	mu         sync.Mutex
	routeNotes map[string][]*pb.SubscribeRequest
}

func serialize(point []string) string {
	return point[0]
}

// SayHello implements helloworld.GreeterServer
func (s *server) SubscribeOnSportsLines(stream pb.KiddyLineProcessor_SubscribeOnSportsLinesServer) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		key := serialize(in.Sports)

		s.mu.Lock()
		s.routeNotes[key] = append(s.routeNotes[key], in)
		// Note: this copy prevents blocking other clients while serving this one.
		// We don't need to do a deep copy, because elements in the slice are
		// insert-only and never modified.
		rn := make([]*pb.SubscribeRequest, len(s.routeNotes[key]))
		copy(rn, s.routeNotes[key])
		s.mu.Unlock()

		for _, note := range rn {
			if err := stream.Send(note); err != nil {
				return err
			}
		}
	}
}
func newServer() *server {
	s := &server{routeNotes: make(map[string][]*pb.SubscribeRequest)}
	return s
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterKiddyLineProcessorServer(s, newServer())
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
