package application

import pb "github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/grpc/proto"

type responseSender interface {
	Send(sports []*Sport) error
}

type GrpcResponseSender struct {
	Stream pb.KiddyLineProcessor_SubscribeOnSportsLinesServer
}

func (s *GrpcResponseSender) Send(sports []*Sport) error {
	var list []*pb.Sport
	for _, sport := range sports {
		list = append(list, &pb.Sport{
			Type: sport.Type,
			Line: sport.Line,
		})
	}
	response := &pb.SubscribeResponse{Sports: list}
	return s.Stream.Send(response)
}
