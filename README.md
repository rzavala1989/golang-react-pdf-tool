# 📄 PDF Manager (Go + React + MinIO + LocalStack Style)

This is a fullstack local simulation of a production PDF management pipeline using:

- ✅ Go backend (PDF uploads, form detection, filling, and flattening)
- ✅ React + Vite frontend playground
- ✅ MinIO (S3-compatible bucket)
- ✅ Docker Compose for local orchestration

---

## 🚀 Quick Start

```bash
git clone https://github.com/yourname/pdf-manager
cd pdf-manager

# Start everything
docker-compose up --build
```

Then visit:

- Frontend: http://localhost:5173
- Backend API: http://localhost:8080
- MinIO Console: http://localhost:9090
- user: minioadmin
- pass: minioadmin

## Teardown

```bash
docker-compose down
```

## cURL Test

```bash
curl -X POST http://localhost:8080/upload \
  -F "pdf=@./some_form.pdf"
```