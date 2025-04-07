package main

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const (
	pdfDirectory = "./pdf_directory"
	bucketName   = "pdf-files"
)

var minioClient *minio.Client

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️  No .env file found, continuing without it")
	} else {
		log.Println("✅ Loaded .env file")
	}

	_ = os.MkdirAll(pdfDirectory, 0755)

	initMinio()
	initDynamo()

	r := mux.NewRouter()
	r.HandleFunc("/upload", uploadPDFHandler).Methods("POST")
	r.HandleFunc("/download/{filename}", downloadPDFHandler).Methods("GET")

	fmt.Println("Server started at http://localhost:8080")
	if err := http.ListenAndServe(":8080", withCORS(r)); err != nil {
		log.Fatalln("Error starting server:", err)
	}
}

// initMinio initializes the MinIO client and ensures the bucket exists.
func initMinio() {
	endpoint := "localhost:9000"
	accessKeyID := "minioadmin"
	secretAccessKey := "minioadmin"
	useSSL := false

	var err error
	minioClient, err = minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalln("❌ MinIO init failed:", err)
	}

	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		log.Fatalln("❌ Bucket check failed:", err)
	}
	if !exists {
		err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: "us-east-1"})
		if err != nil {
			log.Fatalln("❌ Bucket creation failed:", err)
		}
		log.Println("✅ Bucket created:", bucketName)
	}
}

// withCORS attaches CORS headers for dev
func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		h.ServeHTTP(w, r)
	})
}

// uploadPDFHandler → receives a multipart form-file named "pdf", saves locally, uploads to MinIO, then stores metadata in DynamoDB
func uploadPDFHandler(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("pdf")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Save to disk
	outputPath := filepath.Join(pdfDirectory, header.Filename)
	out, err := os.Create(outputPath)
	if err != nil {
		http.Error(w, "Cannot create file on server", http.StatusInternalServerError)
		return
	}
	defer out.Close()

	if _, err = io.Copy(out, file); err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}

	// Upload to MinIO
	if err := uploadToMinio(outputPath, header.Filename); err != nil {
		http.Error(w, "Failed to upload to MinIO: "+err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("⏫ Uploaded %s to bucket %s\n", header.Filename, bucketName)

	// Store metadata in DynamoDB (call your storePDFMeta function!)
	storePDFMeta(header.Filename, 0)

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Uploaded successfully")
}

// uploadToMinio is a helper function to store the file in MinIO
func uploadToMinio(localPath, objectName string) error {
	localFile, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer localFile.Close()

	stat, err := localFile.Stat()
	if err != nil {
		return err
	}

	_, err = minioClient.PutObject(
		context.Background(),
		bucketName,
		objectName,
		localFile,
		stat.Size(),
		minio.PutObjectOptions{ContentType: "application/pdf"},
	)
	return err
}

// downloadPDFHandler → returns the file to the client (if it exists on local disk)
func downloadPDFHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileName := vars["filename"]
	pdfPath := filepath.Join(pdfDirectory, fileName)

	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	http.ServeFile(w, r, pdfPath)
}
