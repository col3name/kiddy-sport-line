package main

import (
	"context"
	"flag"
	"fmt"
	commonDomain "github.com/col3name/lines/pkg/common/domain"
	"io"
	"log"
	"time"

	pb "github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/grpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	addr = flag.String("addr", "localhost:50051", "the address to connect to")
)

func main() {
	flag.Parse()
	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewKiddyLineProcessorClient(conn)

	handleGrpc(client)
}

func handleGrpc(client pb.KiddyLineProcessorClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	stream, err := client.SubscribeOnSportsLines(ctx)
	if err != nil {
		log.Fatalf("client.RouteChat failed: %v", err)
	}

	sports := []string{string(commonDomain.Baseball)}

	handle := clientHandle{stream: stream}

	go handle.subscribeForSport(sports)
	go handle.receiveMessage()
	waitc := make(chan struct{})
	<-waitc
}

type clientHandle struct {
	stream pb.KiddyLineProcessor_SubscribeOnSportsLinesClient
}

func (c *clientHandle) subscribeForSport(sports []string) {
	subscriptions := []*pb.SubscribeRequest{
		{Sports: sports, IntervalInSecond: 1},
	}

	c.sendSubs(subscriptions, 0)

	subscriptions = []*pb.SubscribeRequest{
		{Sports: []string{commonDomain.Soccer.String()}, IntervalInSecond: 1},
	}
	c.sendSubs(subscriptions, 10)

	subscriptions = []*pb.SubscribeRequest{
		{Sports: []string{
			commonDomain.Soccer.String(),
			commonDomain.Football.String(),
		}, IntervalInSecond: 2},
	}
	c.sendSubs(subscriptions, 10)
	subscriptions = []*pb.SubscribeRequest{
		{Sports: []string{
			commonDomain.Baseball.String(),
		}, IntervalInSecond: 1},
	}
	c.sendSubs(subscriptions, 10)
}

func (c *clientHandle) receiveMessage() {
	for {
		recv, err := c.stream.Recv()
		if err == io.EOF {
			log.Println("done", err)
		} else if err != nil {
			log.Fatalf("client.RouteChat failed: %v", err)
		}
		for _, sport := range recv.Sports {
			fmt.Println(sport.Type, sport.Line)
		}
		log.Println(recv.Sports)
	}
}

func (c *clientHandle) sendSubs(subscriptions []*pb.SubscribeRequest, sec int) {
	time.Sleep(time.Duration(sec) * time.Second)

	for _, sub := range subscriptions {
		if err := c.stream.Send(sub); err != nil {
			log.Fatalf("client.RouteChat: stream.Send(%v) failed: %v", sub, err)
		}
	}
}
