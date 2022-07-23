package grpc

import (
	"context"
	"fmt"
	commonDomain "github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/kiddy-line-processor/domain"
	pb "github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/grpc/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"io"
	"net"
	"sync"
)

// Peer contains the information of the peer for an RPC, such as the address
// and authentication information.
type Peer struct {
	// Addr is the peer address.
	Addr net.Addr
	// AuthInfo is the authentication information of the transport.
	// It is nil if there is no transport security being used.
	AuthInfo credentials.AuthInfo
}

type peerKey struct{}

// NewContext creates a new context with peer information attached.
func NewContext(ctx context.Context, p *Peer) context.Context {
	return context.WithValue(ctx, peerKey{}, p)
}

// FromContext returns the peer information in ctx if it exists.
func FromContext(ctx context.Context) (p *Peer, ok bool) {
	value := ctx.Value(peerKey{})
	fmt.Println(value)
	p, ok = value.(*Peer)
	return
}

// server is used to implement helloworld.GreeterServer.
type Server struct {
	pb.UnimplementedKiddyLineProcessorServer
	mu         sync.Mutex
	routeNotes map[string]map[commonDomain.SportType]float32
	sportRepo  domain.SportRepo
}

func serialize(point []string) string {
	return point[0]
}

func parseSportRequest(sports []string) []commonDomain.SportType {
	res := make([]commonDomain.SportType, 0)
	for _, sport := range sports {
		switch sport {
		case string(commonDomain.Football):
			res = append(res, commonDomain.Football)
		case string(commonDomain.Baseball):
			res = append(res, commonDomain.Baseball)
		case string(commonDomain.Soccer):
			res = append(res, commonDomain.Soccer)
		}
	}
	return res
}

// SayHello implements helloworld.GreeterServer
func (s *Server) SubscribeOnSportsLines(stream pb.KiddyLineProcessor_SubscribeOnSportsLinesServer) error {
	for {
		ctx := stream.Context()
		p, _ := peer.FromContext(ctx)
		ip := p.Addr.String()
		fmt.Println(ip)
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		//key := serialize(in.Sports)
		sportRequest := parseSportRequest(in.Sports)

		isNeedDelta := false
		s.mu.Lock()

		oldValue, exist := s.routeNotes[ip]
		if exist {
			if len(oldValue) != len(in.Sports) {
				isNeedDelta = false
			} else {
				for _, sportType := range sportRequest {
					_, ok := s.routeNotes[ip][sportType]
					if !ok {
						isNeedDelta = false
						break
					}
				}
				isNeedDelta = true
			}
		} else {
			isNeedDelta = false
		}
		if !isNeedDelta {
			s.routeNotes[ip] = make(map[commonDomain.SportType]float32, 0)
			for _, sportType := range sportRequest {
				s.routeNotes[ip][sportType] = 1.0
			}
		}
		//TODO сделать проверку наличия запроса с ip address в map
		// Note: this copy prevents blocking other clients while serving this one.
		// We don't need to do a deep copy, because elements in the slice are
		// insert-only and never modified.

		lines, err := s.sportRepo.GetSportLines(sportRequest)
		if err != nil {
			s.mu.Unlock()
			log.Println(err)
		} else {
			s.mu.Unlock()
			var sports []*pb.Sport
			for _, line := range lines {
				sportType := line.Type
				score := line.Score
				if isNeedDelta {
					s.mu.Lock()
					score = s.routeNotes[ip][sportType] - score
					s.mu.Unlock()
				} else {
					s.mu.Lock()
					s.routeNotes[ip][sportType] = score
					s.mu.Unlock()
				}
				resp := &pb.Sport{
					Type: sportType.String(),
					Line: score,
				}
				sports = append(sports, resp)
			}
			response := pb.SubscribeResponse{Sports: sports}
			if err = stream.Send(&response); err != nil {
				log.Println(err)
				return err
			}

			//stream.Send
			//rn := make([]*pb.SubscribeRequest, len(s.routeNotes[key]))
			//copy(rn, s.routeNotes[key])
			//s.mu.Unlock()
			//
			//for _, note := range rn {
			//	if err := stream.Send(note); err != nil {
			//		return err
			//	}
			//}
		}
	}
}

func NewServer(sportRepo domain.SportRepo) *Server {
	s := &Server{
		routeNotes: make(map[string]map[commonDomain.SportType]float32),
		sportRepo:  sportRepo,
	}
	return s
}
