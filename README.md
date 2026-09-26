# storage-service

Project-agnostic blob storage with S3-style multipart uploads. Metadata is event-sourced; bytes live in a pluggable storage backend (currently the filesystem).

The service has no notion of end users. Each calling service holds an API key that maps to a **namespace**, and every upload and file is confined to the namespace of the key that created it. Deciding which end user may see a file is the caller's responsibility (e.g. reporting-service checks job ownership before proxying a download).

## API

All routes live under `/storage/v1` and require `Authorization: Bearer <api key>`.

| Method | Path | Description |
|---|---|---|
| `POST` | `/uploads` | Start an upload: `{"key": "reports/job-1/report.html", "content_type": "text/html"}` |
| `GET` | `/uploads/{upload_id}` | Get an in-progress upload and the parts received so far |
| `PUT` | `/uploads/{upload_id}/parts/{part_number}` | Upload a part (≤ 5 MB, 1-based) |
| `POST` | `/uploads/{upload_id}/complete` | Assemble parts into a file |
| `POST` | `/uploads/{upload_id}/abort` | Discard an upload and its parts |
| `GET` | `/files?prefix=&limit=&cursor=` | List files by key; `limit` defaults to 100 (max 1000), pass `next_cursor` back as `cursor` for the next page |
| `GET` | `/files/{file_id}` | Download; send a `Range` header for part of the file (`206`, or `416` if out of bounds) |
| `GET` | `/files/{file_id}/metadata` | Get a file's metadata without downloading it |
| `POST` | `/files/{file_id}/move` | Rename or move a file: `{"key": "archive/report.html"}`; the file keeps its ID |
| `DELETE` | `/files/{file_id}` | Delete a file |

Keys are relative paths within the namespace; `..` segments are rejected. Completing another upload to an existing key adds another file rather than replacing it. File IDs are UUIDv7, so list returns files ordered by key, then oldest first. Resources in other namespaces return `404`.

Go callers use [`pkg/storageservice`](pkg/storageservice) — `storageservice.UploadFile` handles chunking.

## Configuration

| Variable | Default | Description |
|---|---|---|
| `STORAGE_CLIENTS_B64_JSON` | required | base64 JSON `{"<namespace>": "<sha256 hex of api key>"}` |
| `STORAGE` | `FILESYSTEM` | Storage backend; `FILESYSTEM` is the only one |
| `STORAGE_FILESYSTEM_DIRECTORY` | `./tmp/storage` | Root directory for the filesystem backend |
| `STORAGE_EVENT_LOG_FACTORY` | `INMEMORY` | `INMEMORY` or `REDIS` (+ `STORAGE_EVENT_LOG_REDIS_ADDRESS`); uploads and files use the `storage:uploads` and `storage:files` logs |
| `PORT` | `8083` | Listen port |

Client side (`storageservice.ClientFromEnv`): `STORAGE_SERVICE_CLIENT_IMPLEMENTATION=HTTP`, `STORAGE_SERVICE_URL`, `STORAGE_SERVICE_API_KEY`, optional `STORAGE_SERVICE_HTTP_CLIENT_TIMEOUT`.

## Issuing a key

```sh
go run ./cmd/api-key-generator -namespace example-service
```

Give the raw key to the calling service; add the hash to `STORAGE_CLIENTS_B64_JSON`.
