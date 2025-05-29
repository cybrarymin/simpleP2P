package node

import (
	"context"
	"time"

	"github.com/cybrarymin/simpleP2P/protogen/pb"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Add peer will get a new id and address of the peer and it to the peers of the node
func (n *Node) addPeer(id string, addr string) error {
	n.PeersMutex.Lock()
	defer n.PeersMutex.Unlock()

	if _, exists := n.Peers[id]; exists {
		n.logger.Debug().
			Str("peer_id", id).
			Str("peer_address", addr).
			Msg("ignores adding new peer. peer already exists")
		return nil
	}

	// skip adding the peer if the address is your same address
	if addr == n.Address {
		return nil
	}

	// configuring the grpcClient
	nConn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		n.logger.Error().Err(err).
			Str("peer_id", id).
			Str("peer_address", addr).
			Msg("failed to connect to the peer")
		return err
	}
	grpcClient := pb.NewChatServiceClient(nConn)

	puuid, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	nPeer := &Peer{
		ID:      puuid,
		Address: addr,
		Conn:    nConn,
		client:  grpcClient,
	}

	n.Peers[id] = nPeer
	return nil
}

// dicoverPeers periodically will run in the background to discover new peers joining to p2p network
func (n *Node) discoverPeers(ctx context.Context) {
	defer n.wg.Done()
	ctx, span := otel.Tracer("discoverPeers.tracer").Start(ctx, "discoverPeers.span")
	defer span.End()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			n.logger.Debug().Msg("discovery goroutine closed")
			return
		case <-ticker.C:
			n.performDiscovery(ctx)
		}
	}
}

// Will send query to all the known peers for their peer list
func (n *Node) performDiscovery(ctx context.Context) {
	ctx, span := otel.Tracer("performDiscovery.tracer").Start(ctx, "performDiscovery.span")
	defer span.End()

	n.PeersMutex.RLock()
	defer n.PeersMutex.RUnlock()

	// TSHOOT
	tmp := make(map[string]Peer)
	for k, v := range n.Peers {
		tmp[k] = *v
	}
	n.logger.Debug().Msgf("current peers %+v", tmp)

	for _, peer := range n.Peers {
		if peer.Address != n.Address {
			pResponse, err := peer.client.Discover(ctx, &pb.DiscoveryRequest{
				Peer: &pb.PeerInfo{
					Id:      n.ID.String(),
					Address: n.Address,
				},
			})
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "failed to receive discovery response from peer")
				span.SetAttributes(
					attribute.String("peer_id", peer.ID.String()),
					attribute.String("peer_address", peer.Address),
				)
				n.logger.Info().
					Str("peer_id", peer.ID.String()).
					Str("peer_address", peer.Address).
					Msg("failed to receive discovery response from the peer deleting the peer from list")
				delete(n.Peers, peer.ID.String())
				continue
			}
			for _, rPeer := range pResponse.Peers {
				if rPeer.Id != n.ID.String() && rPeer.Address != n.Address {
					err := n.addPeer(rPeer.Id, rPeer.Address)
					if err != nil {
						span.RecordError(err)
						span.SetStatus(codes.Error, "failed to add peer to peer lists")
						span.SetAttributes(
							attribute.String("peer_id", rPeer.Id),
							attribute.String("peer_address", rPeer.Address),
						)
						n.logger.Info().
							Str("peer_id", rPeer.Id).
							Str("peer_address", rPeer.Address).
							Msg("failed to add peer to peer lists")
						continue
					}
				}
			}
		}
	}
}
