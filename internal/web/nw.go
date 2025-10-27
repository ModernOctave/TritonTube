// Lab 8: Implement a network video content service (client using consistent hashing)

package web

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"log"
	"net"
	"slices"
	"sort"
	"strings"

	pb "tritontube/internal/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func hashStringToUint64(s string) uint64 {
	sum := sha256.Sum256([]byte(s))
	return binary.BigEndian.Uint64(sum[:8])
}

// NetworkVideoContentService implements VideoContentService using a network of nodes.
type Node struct {
	hash uint64
	id string
}

type VideoContentAdminServer struct {
	pb.UnimplementedVideoContentAdminServiceServer
	nw *NetworkVideoContentService
}

func (s *VideoContentAdminServer) AddNode(ctx context.Context, req *pb.AddNodeRequest) (*pb.AddNodeResponse, error) {
	// Add node to hash ring
	s.nw.StorageServers = append(s.nw.StorageServers, req.GetNodeAddress())
	s.nw.initHashRing()

	// Find nextNode
	var nextNode Node
	for idx, node := range s.nw.Nodes {
		if node.id == req.GetNodeAddress() {
			if idx + 1 == len(s.nw.Nodes) {
				nextNode = s.nw.Nodes[0]
			} else {
				nextNode = s.nw.Nodes[idx+1]
			}
			break
		}
	}

	// Open connection to nextNode
	conn, err := grpc.NewClient(nextNode.id, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	client := pb.NewNetworkVideoContentClient(conn)

	// Find out which files need to be displaced
	response, err := client.List(context.Background(), &pb.ListRequest{})
	if err != nil {
		return nil, err
	}
	filesOnNextNode := response.GetFileIds()
	var displacedFiles []string
	for _, file := range filesOnNextNode {
		if s.nw.getNWLocation(strings.Split(file, "/")[0], strings.Split(file, "/")[1]) == req.GetNodeAddress() {
			displacedFiles = append(displacedFiles, file)
		}
	}

	// Move displaced files to new hash ring
	for _, file := range displacedFiles {
		// Read file
		response, err := client.Read(context.Background(), &pb.ReadRequest{
			FileId: file,
		})
		if err != nil {
			return nil, err
		}

		// Write file to NW
		s.nw.Write(strings.Split(file, "/")[0], strings.Split(file, "/")[1], response.GetData())

		// Delete file from old node
		_, err = client.Delete(context.Background(), &pb.DeleteRequest{FileId: file})
		if err != nil {
			return nil, err
		}
	}

	return &pb.AddNodeResponse{MigratedFileCount: int32(len(displacedFiles))}, nil
}

func (s *VideoContentAdminServer) RemoveNode(ctx context.Context, req *pb.RemoveNodeRequest) (*pb.RemoveNodeResponse, error) {
	// Find node index
	var nodeIdx int
	for idx, storageServer := range s.nw.StorageServers {
		if storageServer == req.GetNodeAddress() {
			nodeIdx = idx
			break
		}
	} 
	// Remove node from hash ring
	s.nw.StorageServers = slices.Delete(s.nw.StorageServers, nodeIdx, nodeIdx + 1)
	s.nw.initHashRing()

	// Open connection to node
	conn, err := grpc.NewClient(req.GetNodeAddress(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	client := pb.NewNetworkVideoContentClient(conn)

	// Find out which files need to be displaced
	response, err := client.List(context.Background(), &pb.ListRequest{})
	if err != nil {
		return nil, err
	}
	displacedFiles := response.GetFileIds()

	// Move displaced files to new hash ring
	for _, file := range displacedFiles {
		// Read file
		response, err := client.Read(context.Background(), &pb.ReadRequest{
			FileId: file,
		})
		if err != nil {
			return nil, err
		}

		// Write file to NW
		s.nw.Write(strings.Split(file, "/")[0], strings.Split(file, "/")[1], response.GetData())

		// Delete file from old node
		_, err = client.Delete(context.Background(), &pb.DeleteRequest{FileId: file})
		if err != nil {
			return nil, err
		}
	}
	
	return &pb.RemoveNodeResponse{MigratedFileCount: int32(len(displacedFiles))}, nil
}

func (s *VideoContentAdminServer) ListNodes(ctx context.Context, req *pb.ListNodesRequest) (*pb.ListNodesResponse, error) {
	return &pb.ListNodesResponse{Nodes: s.nw.StorageServers}, nil
}

type NetworkVideoContentService struct{
	initialized bool
	AdminServer string
	StorageServers []string
	Nodes []Node
	Directory map[string][]string
}

func (s *NetworkVideoContentService) initHashRing() {
	s.Nodes = []Node{}

	for _, nodeId := range s.StorageServers {
		s.Nodes = append(s.Nodes, Node{
			hash: hashStringToUint64(nodeId),
			id: nodeId,
		})
	}

	sort.Slice(s.Nodes, func(i, j int) bool {
		return s.Nodes[i].hash < s.Nodes[j].hash
	})
}

func (s *NetworkVideoContentService) initAdminServer() {
	lis, err := net.Listen("tcp", s.AdminServer)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	gs := grpc.NewServer()
	pb.RegisterVideoContentAdminServiceServer(gs, &VideoContentAdminServer{nw: s})
	
	go func() {
		if err := gs.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()
}

func (s *NetworkVideoContentService) getNWLocation(videoId string, filename string) string {
	videoHash := hashStringToUint64(videoId + "/" + filename)

	// Find the target node (smallest hash where video_hash < node_hash)
	var targetNodeId string
	for _, node := range s.Nodes {
		if videoHash < node.hash {
			targetNodeId = node.id
			break
		}
	}
	if targetNodeId == "" {
		targetNodeId = s.Nodes[0].id
	}

	return targetNodeId
}

func (s *NetworkVideoContentService) openNWClient(videoId string, filename string) (pb.NetworkVideoContentClient, error) {
	if !s.initialized {
		s.initHashRing()
		s.initAdminServer()
		s.initialized = true
	}
	
	targetNodeId := s.getNWLocation(videoId, filename)
	
	conn, err := grpc.NewClient(targetNodeId, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	client := pb.NewNetworkVideoContentClient(conn)

	return client, nil
}

func (s *NetworkVideoContentService) Read(videoId string, filename string) ([]byte, error) {
	client, err := s.openNWClient(videoId, filename)
	if err != nil {
		return nil, err
	}

	response, err := client.Read(context.Background(), &pb.ReadRequest{
		FileId: videoId + "/" + filename,
	})

	return response.GetData(), err
}

func (s *NetworkVideoContentService) Write(videoId string, filename string, data []byte) error {
	client, err := s.openNWClient(videoId, filename)
	if err != nil {
		return err
	}

	_, err = client.Write(context.Background(), &pb.WriteRequest{
		FileId: videoId + "/" + filename,
		Data: data,
	})
	if err != nil {
		return err
	}

	return err
}

// Uncomment the following line to ensure NetworkVideoContentService implements VideoContentService
var _ VideoContentService = (*NetworkVideoContentService)(nil)
