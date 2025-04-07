package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"pdf-stuff/helpers"
)

// pdfDirectory is the directory where uploaded PDFs are stored.
// bucketName is the name of the MinIO bucket where PDFs are stored.
const (
	pdfDirectory = "./pdf_directory"
	bucketName   = "pdf-files"
)

// minioClient is the MinIO client instance.
var minioClient *minio.Client

// initMinio initializes the MinIO client and ensures the bucket exists. TODO: AWS S3 Integration goes here
func initMinio() {
	endpoint := "minio:9000"
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

	// Ensure bucket exists
	bucketName := "pdf-files"
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

type FieldData struct {
	Fields []Field `json:"fields"`
}

type Field struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		h.ServeHTTP(w, r)
	})
}

func main() {
	_ = os.MkdirAll(pdfDirectory, 0755)

	initMinio()
	initDynamo()
	r := mux.NewRouter()

	r.HandleFunc("/upload", UploadPDFHandler).Methods("POST")
	r.HandleFunc("/fields/{filename}", ListFieldsHandler).Methods("GET")
	r.HandleFunc("/fill/{filename}", FillFieldsHandler).Methods("POST")
	r.HandleFunc("/download/{filename}", DownloadPDFHandler).Methods("GET")

	fmt.Println("Server started at http://localhost:8080")
	err := http.ListenAndServe(":8080", withCORS(r))
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
}

// UploadPDFHandler handles the PDF upload and saves it to both local storage and MinIO.
func UploadPDFHandler(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Cannot save file", http.StatusInternalServerError)
		return
	}
	defer out.Close()
	_, err = io.Copy(out, file)
	if err != nil {
		http.Error(w, "Error saving", http.StatusInternalServerError)
		return
	}

	// Upload to MinIO
	localFile, _ := os.Open(outputPath)
	defer localFile.Close()
	stat, _ := localFile.Stat()
	_, err = minioClient.PutObject(context.Background(), bucketName, header.Filename, localFile, stat.Size(), minio.PutObjectOptions{ContentType: "application/pdf"})
	if err != nil {
		http.Error(w, "Failed to upload to MinIO: "+err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("⏫ Uploaded %s to bucket %s\n", header.Filename, bucketName)

	fieldsMap, err := helpers.ExtractFormFields(outputPath)
	if err == nil {
		storePDFMeta(header.Filename, len(fieldsMap))
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Uploaded successfully")
}

// ListFieldsHandler lists the form fields in the specified PDF file.
func ListFieldsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileName := vars["filename"]
	pdfPath := filepath.Join(pdfDirectory, fileName)

	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		http.Error(w, "PDF not found", http.StatusNotFound)
		return
	}

	fieldsMap, err := helpers.ExtractFormFields(pdfPath)
	if err != nil {
		http.Error(w, "Error extracting form fields: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var fields []Field
	for name, value := range fieldsMap {
		fields = append(fields, Field{
			Name:  name,
			Value: value,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(fields)
	if err != nil {
		http.Error(w, "Error encoding JSON: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// FillFieldsHandler fills the form fields in the specified PDF file with the provided data.
func FillFieldsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileName := vars["filename"]
	inPath := filepath.Join(pdfDirectory, fileName)

	var fieldData FieldData
	if err := json.NewDecoder(r.Body).Decode(&fieldData); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	ffMap := make(map[string]string)
	for _, f := range fieldData.Fields {
		ffMap[f.Name] = f.Value
	}

	fdfContent := helpers.GenerateFDF(ffMap)
	tmpFDFPath := filepath.Join(pdfDirectory, "temp.fdf")
	if err := os.WriteFile(tmpFDFPath, []byte(fdfContent), fs.ModePerm); err != nil {
		http.Error(w, "Error writing temporary FDF file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	outPath := filepath.Join(pdfDirectory, "filled_"+fileName)
	conf := model.NewDefaultConfiguration()

	if err := api.FillFormFile(inPath, tmpFDFPath, outPath, conf); err != nil {
		http.Error(w, "FillFormFile failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	flattenedPath := filepath.Join(pdfDirectory, "flattened_"+fileName)
	if err := api.LockFormFieldsFile(outPath, flattenedPath, []string{}, conf); err != nil {
		http.Error(w, "LockFormFieldsFile (flatten) failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_ = os.Remove(tmpFDFPath)

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(map[string]string{
		"message":   "Form filled and flattened",
		"filledPDF": "filled_" + fileName,
		"finalPDF":  "flattened_" + fileName,
	})
	if err != nil {
		return
	}
}

// DownloadPDFHandler serves the filled PDF file for download.
func DownloadPDFHandler(w http.ResponseWriter, r *http.Request) {
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
