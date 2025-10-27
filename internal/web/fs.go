// Lab 7: Implement a local filesystem video content service

package web

import (
	"log"
	"os"
	"path"
)

// FSVideoContentService implements VideoContentService using the local filesystem.
type FSVideoContentService struct{
	FSDir string
}

func (s FSVideoContentService) Read(videoId string, filename string) ([]byte, error) {
	readData, err := os.ReadFile(path.Join(s.FSDir, videoId, filename))
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		log.Printf("Error while reading from file: %v", err)
		return nil, err
	}
	return readData, nil
}

func (s FSVideoContentService) Write(videoId string, filename string, data []byte) error {
	dirName := path.Join(s.FSDir, videoId)

	err := os.MkdirAll(dirName, 0755)
	if err != nil {
		log.Printf("Error while creating directory: %v", err)
		return err
	}

	err = os.WriteFile(path.Join(dirName, filename), data, 0644)
	if err != nil {
		log.Printf("Error while writing to file: %v", err)
		return err
	}

	return nil
}

// Uncomment the following line to ensure FSVideoContentService implements VideoContentService
var _ VideoContentService = (*FSVideoContentService)(nil)
