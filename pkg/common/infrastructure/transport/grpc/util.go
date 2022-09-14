package grpc

import (
	loggerInterface "github.com/col3name/lines/pkg/common/application/logger"
	"google.golang.org/grpc"
	"net"
)

func RunGrpcServer(logger loggerInterface.Logger, url string, srv *grpc.Server) {
	lis, err := net.Listen("tcp", url)
	if err != nil {
		logger.Fatalf("failed to listen: %v", err)
	}
	logger.Info("server listening at", lis.Addr().String())
	if err = srv.Serve(lis); err != nil {
		logger.Fatal("failed to serve: ", err)
	}
}
