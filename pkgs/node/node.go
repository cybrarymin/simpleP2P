package node

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/cybrarymin/simpleP2P/protogen/pb"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Node struct {
	logger         *zerolog.Logger
	ID             uuid.UUID // id of the snode
	Address        string    // address of the node
	BootstrapAddr  string    // bootstrap node address
	IsBootstrap    bool      // is this node a boostrap node or not
	Server         *grpc.Server
	Peers          map[string]*Peer
	PeersMutex     sync.RWMutex
	MessageChannel chan *pb.ChatMessage
	ctx            context.Context
	cancelFunc     context.CancelFunc
	wg             sync.WaitGroup
}

type Peer struct {
	ID      uuid.UUID
	Address string
	Conn    *grpc.ClientConn
	client  pb.ChatServiceClient
}

func NewNode(logger *zerolog.Logger, addr string, bootstrapAddr string, isBootstrap bool) *Node {
	ctx, cancelFunc := context.WithCancel(context.Background())
	return &Node{
		logger:         logger,
		ID:             uuid.New(),
		Address:        addr,
		IsBootstrap:    isBootstrap,
		BootstrapAddr:  bootstrapAddr,
		Peers:          make(map[string]*Peer),
		MessageChannel: make(chan *pb.ChatMessage),
		ctx:            ctx,
		cancelFunc:     cancelFunc,
	}
}

func (n *Node) Start(ctx context.Context) error {
	// start the grpc server of each p2p node
	go n.startServer()

	// If this node is not a bootstrap node send a connect to the bootstrap node to get the list of peers
	if !n.IsBootstrap && n.BootstrapAddr != "" {
		if err := n.connectToBoostrap(ctx); err != nil {
			return err
		}
	}

	// start the peer discovery process
	go n.discoverPeers(ctx)
	// start message processing
	go n.processMessages()

	return nil
}

func (n *Node) Stop(ctx context.Context) error {
	ctx, cancelFunc := context.WithTimeout(ctx, time.Second*5)
	defer cancelFunc()

	n.cancelFunc()

	n.PeersMutex.Lock()
	for _, p := range n.Peers {
		if p.Conn != nil {
			n.logger.Info().Msg("closing peer outbound connection")
			p.Conn.Close()
		}
	}
	n.PeersMutex.Unlock()

	n.logger.Info().Msg("stopping grpc server")
	if n.Server != nil {
		n.Server.Stop()
	}

	done := make(chan struct{})
	go func() {
		n.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (n *Node) startServer() {
	n.wg.Add(1)
	defer n.wg.Done()

	n.Server = grpc.NewServer()

	pb.RegisterChatServiceServer(n.Server, NewChatServiceServer(n))
	pb.RegisterNodeServiceServer(n.Server, NewNodeServiceServer(n))
	reflection.Register(n.Server)

	listener, err := net.Listen("tcp4", n.Address)
	if err != nil {
		n.logger.Panic().Err(err).Send()
	}

	err = n.Server.Serve(listener)
	if err != nil {
		n.logger.Panic().Err(err).Send()
	}
}

// TODO
func (n *Node) processMessages() {
	n.wg.Add(1)
	defer n.wg.Done()
	for {
		<-n.MessageChannel
		n.logger.Info().Msgf("message processed succefully")
	}
}

// send message will send message to all the peers by adding it to the message channel
func (n *Node) sendMessage(content string) {
	message := &pb.ChatMessage{
		Sender:  n.Address,
		Content: content,
		Time:    timestamppb.New(time.Now()),
	}
	n.MessageChannel <- message
}

// connectToBoostrap will use addPeer function to get connected to the boostrap node and add it to the peer list
func (n *Node) connectToBoostrap(ctx context.Context) error {
	n.wg.Add(1)
	defer n.wg.Done()
	_, span := otel.Tracer("connectToBoostrap.tracer").Start(ctx, "connectToBoostrap.span")
	defer span.End()

	// fetching the boostrap node id to add that to the list of peers
	n.logger.Debug().
		Msgf("getting connected to the boostrap node on %s to fetch id", n.BootstrapAddr)

	nConn, err := grpc.NewClient(n.BootstrapAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	grpcClient := pb.NewNodeServiceClient(nConn)
	resp, err := grpcClient.FetchID(ctx, &pb.BootstrapIDReq{})
	if err != nil {
		return err
	}
	nConn.Close()

	n.logger.Debug().
		Str("boostrap_id", resp.Id).
		Msg("fetched the boostrap node id")

	// adding the bootstrap node of the peer list to make peer discovery process to start after timer to get the full list of peers.
	err = n.addPeer(resp.Id, n.BootstrapAddr)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to add bootstrap node to peer list")
		span.SetAttributes(
			attribute.String("bootstrap_addr", n.BootstrapAddr),
		)
		n.logger.Error().Err(err).Send()
		return err
	}

	return nil
}
