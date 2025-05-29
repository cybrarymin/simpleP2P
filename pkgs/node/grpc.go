package node

import (
	"context"
	"strings"
	"time"

	"github.com/cybrarymin/simpleP2P/protogen/pb"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/grpc"
	grpcCode "google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ChatServiceServer struct {
	pb.ChatServiceServer
	node *Node
}

type NodeServiceServer struct {
	pb.NodeServiceServer
	node *Node
}

func NewChatServiceServer(node *Node) *ChatServiceServer {
	return &ChatServiceServer{
		node: node,
	}
}

func NewNodeServiceServer(node *Node) *NodeServiceServer {
	return &NodeServiceServer{
		node: node,
	}
}

// Discover will add the requesting peer to the list of peers if it is not already in the list and returns the list of our known peers to the requesting peer
func (s *ChatServiceServer) Discover(ctx context.Context, req *pb.DiscoveryRequest) (*pb.DiscoveryResponse, error) {
	_, span := otel.Tracer("grpc.discover.tracer").Start(ctx, "grpc.discover.span")
	defer span.End()
	// adding the requesting peer to our peer list
	if req.Peer != nil && req.Peer.Address != s.node.Address {
		err := s.node.addPeer(strings.ToLower(req.Peer.Id), req.Peer.Address)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to add the requesting peer to the list of peers")
			return nil, status.Error(grpcCode.Internal, "failed to add the requesting peer to the list of peers")
		}
	}

	// return the list of our known peers
	response := &pb.DiscoveryResponse{
		Peers: make([]*pb.PeerInfo, 0),
	}
	s.node.PeersMutex.RLock()
	for id, peer := range s.node.Peers {
		response.Peers = append(response.Peers, &pb.PeerInfo{
			Id:      id,
			Address: peer.Address,
		})
	}
	s.node.PeersMutex.RUnlock()

	return response, nil
}

// SendMessage will send the message to all the peers by adding it to the message queue
func (s *ChatServiceServer) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	_, span := otel.Tracer("grpc.sendMessage.tracer").Start(ctx, "grpc.sendMessage.span")
	defer span.End()

	s.node.sendMessage(req.Message.Content)
	resp := &pb.SendMessageResponse{
		Message: &pb.ChatMessage{
			Sender:  s.node.Address,
			Content: "Message received",
			Time:    timestamppb.New(time.Now()),
		},
	}
	return resp, nil
}

func (s *ChatServiceServer) Subscrible(req *pb.SubscribeRequest, res grpc.ServerStreamingServer[pb.SendMessageResponse]) error {
	return nil
}

// FetchID will send the server uuid string back to the client requesting it.
func (s *NodeServiceServer) FetchID(ctx context.Context, req *pb.BootstrapIDReq) (*pb.BootstrapIDResp, error) {
	_, span := otel.Tracer("grpc.FetchID.Tracer").Start(ctx, "grpc.FetchID.Span")
	defer span.End()

	resp := &pb.BootstrapIDResp{
		Id: s.node.ID.String(),
	}
	return resp, nil
}
