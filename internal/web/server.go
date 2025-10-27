// Lab 7: Implement a web server

package web

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type server struct {
	Addr string
	Port int

	metadataService VideoMetadataService
	contentService  VideoContentService

	mux *http.ServeMux
}

func NewServer(
	metadataService VideoMetadataService,
	contentService VideoContentService,
) *server {
	return &server{
		metadataService: metadataService,
		contentService:  contentService,
	}
}

func (s *server) Start(lis net.Listener) error {
	s.mux = http.NewServeMux()
	s.mux.HandleFunc("/upload", s.handleUpload)
	s.mux.HandleFunc("/videos/", s.handleVideo)
	s.mux.HandleFunc("/content/", s.handleVideoContent)
	s.mux.HandleFunc("/", s.handleIndex)

	return http.Serve(lis, s.mux)
}

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	metadata, err := s.metadataService.List()
	if err != nil {
		msg := fmt.Sprintf("Error while fetching metadata: %v", err)
		log.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	type IndexTmplData struct{
		Id string
		UploadTime string
		EscapedId string
	}
	var data []IndexTmplData
	for _, val := range metadata {
		data = append(data, IndexTmplData{
			Id: val.Id,
			UploadTime: val.UploadedAt.Format("2006-01-02 15:04:05"),
			EscapedId: url.PathEscape(val.Id),
		})
	}

	tmpl, err := template.New("index").Parse(indexHTML)
	if err != nil {
		msg := fmt.Sprintf("Error while parsing template: %v", err)
		log.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

func (s *server) handleUpload(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		msg := fmt.Sprintf("Error while parsing form: %v", err)
		log.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	upload_file, upload_header, err := r.FormFile("file")
	if err != nil {
		msg := fmt.Sprintf("Error while loading file from form: %v", err)
		log.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	defer upload_file.Close()

	videoId := strings.Split(path.Base(upload_header.Filename), ".")[0]

	metadata, err := s.metadataService.Read(videoId)
	if err != nil {
		msg := fmt.Sprintf("Error while reading metadata: %v", err)
		log.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	if metadata != nil {
		http.Error(w, "File with same videoId already exists!", http.StatusConflict)
		return
	}

	temp_file, err := os.CreateTemp("", upload_header.Filename)
	if err != nil {
		msg := fmt.Sprintf("Error while creating temp file: %v", err)
		log.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	defer os.Remove(temp_file.Name())

	data, err := io.ReadAll(upload_file)
	if err != nil {
		msg := fmt.Sprintf("Error while reading file data: %v", err)
		log.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	temp_file.Write(data)

	tempDir, err := os.MkdirTemp("", upload_header.Filename)
	if err != nil {
		msg := fmt.Sprintf("Error while creating temp directory: %v", err)
		log.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	
	manifestPath := filepath.Join(tempDir, "manifest.mpd")

	cmd := exec.Command("ffmpeg",
			"-i", temp_file.Name(), // input file
			"-c:v", "libx264", // video codec
			"-c:a", "aac", // audio codec
			"-bf", "1", // max 1 b-frame
			"-keyint_min", "120", // minimum keyframe interval
			"-g", "120", // keyframe every 120 frames
			"-sc_threshold", "0", // scene change threshold
			"-b:v", "3000k", // video bitrate
			"-b:a", "128k", // audio bitrate
			"-f", "dash", // dash format
			"-use_timeline", "1", // use timeline
			"-use_template", "1", // use template
			"-init_seg_name", "init-$RepresentationID$.m4s", // init segment naming
			"-media_seg_name", "chunk-$RepresentationID$-$Number%05d$.m4s", // media segment naming
			"-seg_duration", "4", // segment duration in seconds
			manifestPath) // output file
	cmd.Run()

	
	mpegDashFiles, err := os.ReadDir(tempDir)
	if err != nil {
		msg := fmt.Sprintf("Error while reading temp directory: %v", err)
		log.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	for _, file := range mpegDashFiles {
		data, err := os.ReadFile(path.Join(tempDir, file.Name()))
		if err != nil {
			msg := fmt.Sprintf("Error while reading temp file: %v", err)
			log.Println(msg)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}
		err = s.contentService.Write(videoId, file.Name(), data)
		if err != nil {
			msg := fmt.Sprintf("Error while writing file to content service: %v", err)
			log.Println(msg)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}
	}
	
	err = s.metadataService.Create(videoId, time.Now())
	if err != nil {
		msg := fmt.Sprintf("Error while creating metadata entry: %v", err)
		log.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *server) handleVideo(w http.ResponseWriter, r *http.Request) {
	videoId := r.URL.Path[len("/videos/"):]

	metadata, err := s.metadataService.Read(videoId)
	if err != nil {
		msg := fmt.Sprintf("Error while reading metadata: %v", err)
		log.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	if metadata == nil {
		http.Error(w, "No such videoId!", http.StatusNotFound)
		return
	}
	
	type VideoTmplData struct {
		Id string
		UploadedAt string
	}

	tmpl := template.Must(template.New("index").Parse(videoHTML))
	tmpl.Execute(w, VideoTmplData{videoId, metadata.UploadedAt.Format("2006-01-02 15:04:05")})
}

func (s *server) handleVideoContent(w http.ResponseWriter, r *http.Request) {
	// parse /content/<videoId>/<filename>
	videoId := r.URL.Path[len("/content/"):]
	parts := strings.Split(videoId, "/")
	if len(parts) != 2 {
		http.Error(w, "Invalid content path", http.StatusBadRequest)
		return
	}
	videoId = parts[0]
	filename := parts[1]

	file, err := s.contentService.Read(videoId, filename)
	if err != nil {
		msg := fmt.Sprintf("Error while reading file from content service: %v", err)
		log.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	if file == nil {
		http.Error(w, "Content not found!", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/octet-stream")
	_, err = w.Write(file)
	if err != nil {
		// Ignore errors of the type "write tcp 127.0.0.1:8080->127.0.0.1:36822: write: broken pipe"
		// Caused by the client closing the connection before the server finishes writing
		if errors.Is(err, syscall.EPIPE) {
			return
		}
		msg := fmt.Sprintf("Error while sending data: %v", err)
		log.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
}