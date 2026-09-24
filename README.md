# storage-service

Project-agnostic blob storage with S3-style multipart uploads. Metadata is event-sourced; bytes live in a pluggable backend (filesystem or in-memory).

The service has no notion of end users. Each calling service holds an API key that maps to a **namespace**, and every upload and file is confined to the namespace of the key that created it. Deciding which end user may see a file is the caller's responsibility (e.g. reporting-service checks job ownership before proxying a download).

## API

All routes live under `/storage/v1` and require `Authorization: Bearer <api key>`.

| Method | Path | Description |
|---|---|---|
| `POST` | `/uploads` | Start an upload: `{"key": "reports/job-1/report.html", "content_type": "text/html"}` |
| `PUT` | `/uploads/{upload_id}/parts/{part_number}` | Upload a part (≤ 5 MB, 1-based) |
| `POST` | `/uploads/{upload_id}/complete` | Assemble parts into a file |
| `GET` | `/files/{file_id}` | Download (supports range requests) |

Keys are relative paths within the namespace; `..` segments are rejected. Resources in other namespaces return `404`.

Go callers use [`pkg/storageservice`](pkg/storageservice) — `storageservice.UploadFile` handles chunking.

## Configuration

| Variable | Default | Description |
|---|---|---|
| `STORAGE_CLIENTS_B64_JSON` | required | base64 JSON `{"<namespace>": "<sha256 hex of api key>"}` |
| `STORAGE_BACKEND` | `INMEMORY` | `INMEMORY` or `FILESYSTEM` |
| `STORAGE_FILESYSTEM_DIRECTORY` | `./tmp/storage` | Root directory for the filesystem backend |
| `STORAGE_EVENT_LOG_FACTORY` | `INMEMORY` | `INMEMORY` or `REDIS` (+ `STORAGE_EVENT_LOG_REDIS_ADDRESS`) |
| `STORAGE_LEGACY_NAMESPACE` | empty | Namespace assigned to uploads recorded before namespaces existed |
| `PORT` | `8083` | Listen port |

Client side (`storageservice.ClientFromEnv`): `STORAGE_SERVICE_CLIENT_IMPLEMENTATION=HTTP`, `STORAGE_SERVICE_URL`, `STORAGE_SERVICE_API_KEY`, optional `STORAGE_SERVICE_HTTP_CLIENT_TIMEOUT`.

## Issuing a key

```sh
go run ./cmd/api-key-generator -namespace example-service
```

Give the raw key to the calling service; add the hash to `STORAGE_CLIENTS_B64_JSON`.
