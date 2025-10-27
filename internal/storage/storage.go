// Lab 8: Implement a network video content service (server)

package storage

import (
	"context"
	"log"
	"os"
	"path"
	"strings"
	pb "tritontube/internal/proto"
)

// Implement a network video content service (server)
type NetworkVideoContentServer struct {
	pb.UnimplementedNetworkVideoContentServer
	Dir string
}

func (s *NetworkVideoContentServer) Read(ctx context.Context, readRequest *pb.ReadRequest) (*pb.ReadResponse, error) {
	readData, err := os.ReadFile(path.Join(s.Dir, readRequest.GetFileId()))
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		log.Printf("Error while reading from file: %v", err)
		return nil, err
	}
	return &pb.ReadResponse{Data: readData}, nil
}

func (s *NetworkVideoContentServer) Write(ctx context.Context, writeRequest *pb.WriteRequest) (*pb.WriteResponse, error) {
	dirName := path.Join(s.Dir, strings.Split(writeRequest.GetFileId(), "/")[0])

	err := os.MkdirAll(dirName, 0755)
	if err != nil {
		log.Printf("Error while creating directory: %v", err)
		return nil, err
	}

	err = os.WriteFile(path.Join(s.Dir, writeRequest.GetFileId()), writeRequest.GetData(), 0644)
	if err != nil {
		log.Printf("Error while writing to file: %v", err)
		return nil, err
	}

	return &pb.WriteResponse{}, nil
}

func (s *NetworkVideoContentServer) List(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
	var file_ids []string

	videos, err := os.ReadDir(s.Dir)
	if err != nil {
		log.Printf("Error while reading directory: %v)", err)
		return nil, err
	}

	for _, video := range videos {
		files, err := os.ReadDir(path.Join(s.Dir, video.Name()))
		if err != nil {
			log.Printf("Error while reading directory: %v)", err)
			return nil, err
		}

		for _, file := range files {
			file_ids = append(file_ids, video.Name() + "/" + file.Name())
		}
	}

	return &pb.ListResponse{FileIds: file_ids}, nil
}

func (s *NetworkVideoContentServer) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	err := os.Remove(path.Join(s.Dir, req.GetFileId()))
	if err != nil {
		return nil, err
	}

	return &pb.DeleteResponse{}, nil
}