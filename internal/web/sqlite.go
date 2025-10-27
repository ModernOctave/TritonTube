// Lab 7: Implement a SQLite video metadata service

package web

import (
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteVideoMetadataService struct{
	DBPath string
}

func (s SQLiteVideoMetadataService) OpenDB() (*sql.DB, error) {
	var makeTable bool

	// Check if DB exists
	_, err := os.Stat(s.DBPath)
	if os.IsNotExist(err) {
		makeTable = true
	} else if err != nil {
		log.Printf("Error while checking if SQLite database exists: %v", err)
		return nil, err
	}

	// Open DB
	db, err := sql.Open("sqlite3", s.DBPath)
	if err != nil {
		log.Printf("Error while opening SQLite database: %v", err)
		return nil, err
	}

	// If DB didn't exist before, create the table
	if makeTable {
		_, err = db.Exec("CREATE TABLE metadata (videoID TEXT NOT NULL PRIMARY KEY, uploadedAt TIMESTAMP);")
		if err != nil {
			log.Printf("Error while trying to create table: %v", err)
			return nil, err
		}
	}

	return db, nil
}

func (s SQLiteVideoMetadataService) Read(id string) (*VideoMetadata, error) {
	db, err := s.OpenDB()
	if err != nil {
		log.Printf("Error while opening SQLite database: %v", err)
		return nil, err
	}
	defer db.Close()

	row := db.QueryRow("SELECT uploadedAt FROM metadata WHERE videoID = ?", id)
	
	var uploadedAt time.Time
	err = row.Scan(&uploadedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		log.Printf("Error while parsing rows (read): %v", err)
		return nil, err
	}
	return &VideoMetadata{
			Id: id,
			UploadedAt: uploadedAt,
		}, nil
}

func (s SQLiteVideoMetadataService) List() ([]VideoMetadata, error) {
	db, err := s.OpenDB()
	if err != nil {
		log.Printf("Error while opening SQLite database: %v", err)
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query("SELECT videoID, uploadedAt FROM metadata")
	if err != nil {
		log.Printf("Error while querying metadata (list): %v", err)
		return nil, err
	}
	defer rows.Close()
	
	var retSlice []VideoMetadata
	for rows.Next() {
		var videoID string
		var uploadedAt time.Time
		err := rows.Scan(&videoID, &uploadedAt)
		if err != nil {
			log.Printf("Error while parsing rows (list): %v", err)
			return nil, err
		}
		retSlice = append(retSlice, VideoMetadata{
			Id: videoID,
			UploadedAt: uploadedAt,
		})
	}

	return retSlice, nil
}

func (s SQLiteVideoMetadataService) Create(videoId string, uploadedAt time.Time) error {
	db, err := s.OpenDB()
	if err != nil {
		log.Printf("Error while opening SQLite database: %v", err)
		return err
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO metadata (videoID, uploadedAt) VALUES (?, ?)", videoId, uploadedAt)
	if err != nil {
		log.Printf("Error while inserting metadata: %v", err)
		return err
	}

	return nil
}

// Uncomment the following line to ensure SQLiteVideoMetadataService implements VideoMetadataService
var _ VideoMetadataService = (*SQLiteVideoMetadataService)(nil)
