# React + TypeScript + Vite

* INSTALL pdfcpu GLOBALLY

```bash
go install github.com/pdfcpu/pdfcpu/cmd/pdfcpu@latest 
```

* SIMULATE S3 BUCKET LOCALLY


```bash

# Install on macOS 
brew install minio/stable/mc

# Install on Linux (MinIO server binary)
wget https://dl.min.io/server/minio/release/linux-amd64/minio
chmod +x minio
sudo mv minio /usr/local/bin

# Creds
export MINIO_ROOT_USER=minioadmin
export MINIO_ROOT_PASSWORD=minioadmin

```

Start MinIO server in Docker (make sure Docker is running)

```bash
docker run -p 9000:9000 -p 9090:9090 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  quay.io/minio/minio server /data --console-address ":9090"
```

Configure mc Client and Create Bucket

```bash
mc alias set local http://localhost:9000 minioadmin minioadmin
mc mb local/pdf-files
mc ls local
```

Configure mock DynamoDB

```bash
docker run -p 8000:8000 amazon/dynamodb-local
```

