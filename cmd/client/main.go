package main

import (
	"context"
	"fmt"
	loggerInterface "github.com/col3name/lines/pkg/common/application/logger"
	commonDomain "github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/common/infrastructure/env"
	"github.com/col3name/lines/pkg/common/infrastructure/logrusLogger"
	"io"
	"time"

	pb "github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/transport/grpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	logger := logrusLogger.New()
	kiddyGrpcUrl := env.GetEnvVariable("KIDDY_LINES_PROCESSOR_GRPC_URL", "localhost:50051")

	conn, err := grpc.Dial(kiddyGrpcUrl, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatal("did not connect: ", err)
	}
	defer conn.Close()
	client := pb.NewKiddyLineProcessorClient(conn)

	handleGrpc(logger, client)
}

func handleGrpc(logger loggerInterface.Logger, client pb.KiddyLineProcessorClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	stream, err := client.SubscribeOnSportsLines(ctx)
	if err != nil {
		logger.Fatal("client.RouteChat failed: ", err)
	}

	sports := []string{string(commonDomain.Soccer)}

	handle := clientHandle{stream: stream, logger: logger}

	go handle.subscribeToSports(sports)
	go handle.receiveMessages()
	waitChan := make(chan struct{})
	<-waitChan
}

type clientHandle struct {
	stream pb.KiddyLineProcessor_SubscribeOnSportsLinesClient
	logger loggerInterface.Logger
}

func (c *clientHandle) subscribeToSports(sports []string) {
	subscriptions := []*pb.SubscribeRequest{
		{Sports: sports, IntervalInSecond: 1},
	}

	c.sendSubscribeRequests(subscriptions, 0)

	subscriptions = []*pb.SubscribeRequest{
		{Sports: []string{commonDomain.Soccer.String()}, IntervalInSecond: 1},
	}
	c.sendSubscribeRequests(subscriptions, 10)

	subscriptions = []*pb.SubscribeRequest{
		{
			Sports: []string{
				commonDomain.Soccer.String(),
				commonDomain.Football.String(),
			},
			IntervalInSecond: 2,
		},
	}
	c.sendSubscribeRequests(subscriptions, 10)
	subscriptions = []*pb.SubscribeRequest{
		{
			Sports:           []string{commonDomain.Baseball.String()},
			IntervalInSecond: 1,
		},
	}
	c.sendSubscribeRequests(subscriptions, 10)
}

func (c *clientHandle) receiveMessages() {
	for {
		if !c.receiveMessage() {
			break
		}
	}
}

func (c *clientHandle) receiveMessage() bool {
	recv, err := c.stream.Recv()
	if err == io.EOF {
		c.logger.Println("done", err)
		return false
	}
	if err != nil {
		c.logger.Fatal("client.RouteChat failed:", err)
	}
	c.printSports(recv)
	return true
}

func (c *clientHandle) printSports(recv *pb.SubscribeResponse) {
	for _, sport := range recv.Sports {
		fmt.Println(sport.Type, sport.Line)
	}
}

func (c *clientHandle) sendSubscribeRequests(subscriptions []*pb.SubscribeRequest, sec int) {
	time.Sleep(time.Duration(sec) * time.Second)

	for _, sub := range subscriptions {
		if err := c.stream.Send(sub); err != nil {
			c.logger.Fatalf("client.RouteChat: stream.Send(%v) failed: %v", sub, err)
		}
	}
}
