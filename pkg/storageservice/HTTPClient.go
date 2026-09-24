package storageservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kduong-dev/goutil/fatal"
)

type HTTPClient struct {
	baseURL    url.URL
	apiKey     string
	httpClient *http.Client
}

type NewHTTPClientInput struct {
	Timeout time.Duration
	BaseURL url.URL
	APIKey  string
}

func NewHTTPClient(input NewHTTPClientInput) *HTTPClient {
	return &HTTPClient{
		baseURL:    input.BaseURL,
		apiKey:     input.APIKey,
		httpClient: &http.Client{Timeout: input.Timeout},
	}
}

type initialiseUploadRequestBody struct {
	Key         string `json:"key"`
	ContentType string `json:"content_type"`
}

func (client *HTTPClient) InitialiseUpload(ctx context.Context, key string, contentType string) (output *Upload, err error) {
	requestBody := fatal.UnlessMarshal(initialiseUploadRequestBody{Key: key, ContentType: contentType})
	request := client.newRequest(ctx, http.MethodPost, "/storage/v1/uploads", bytes.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	err = client.doJSON(request, http.StatusCreated, &output)
	return
}

func (client *HTTPClient) UploadPart(ctx context.Context, uploadID string, partNumber int, body io.Reader) (output *UploadPartResponse, err error) {
	path := fmt.Sprintf("/storage/v1/uploads/%s/parts/%d", url.PathEscape(uploadID), partNumber)
	request := client.newRequest(ctx, http.MethodPut, path, body)
	request.Header.Set("Content-Type", "application/octet-stream")
	err = client.doJSON(request, http.StatusOK, &output)
	return
}

func (client *HTTPClient) CompleteUpload(ctx context.Context, uploadID string) (output *File, err error) {
	path := fmt.Sprintf("/storage/v1/uploads/%s/complete", url.PathEscape(uploadID))
	request := client.newRequest(ctx, http.MethodPost, path, nil)
	err = client.doJSON(request, http.StatusCreated, &output)
	return
}

func (client *HTTPClient) AbortUpload(ctx context.Context, uploadID string) error {
	path := fmt.Sprintf("/storage/v1/uploads/%s/abort", url.PathEscape(uploadID))
	request := client.newRequest(ctx, http.MethodPost, path, nil)
	response, err := client.do(request, http.StatusNoContent)
	if err != nil {
		return err
	}
	return response.Body.Close()
}

func (client *HTTPClient) DownloadFile(ctx context.Context, fileID string) (output *DownloadFileResponse, err error) {
	path := fmt.Sprintf("/storage/v1/files/%s", url.PathEscape(fileID))
	request := client.newRequest(ctx, http.MethodGet, path, nil)
	response, err := client.do(request, http.StatusOK)
	if err != nil {
		return
	}
	output = &DownloadFileResponse{
		ContentType:        response.Header.Get("Content-Type"),
		ContentDisposition: response.Header.Get("Content-Disposition"),
		Body:               response.Body,
	}
	return
}

func (client *HTTPClient) ListFiles(ctx context.Context, input ListFilesInput) (output *ListFilesResponse, err error) {
	query := url.Values{}
	if input.Prefix != "" {
		query.Set("prefix", input.Prefix)
	}
	if input.Cursor != "" {
		query.Set("cursor", input.Cursor)
	}
	if input.Limit != 0 {
		query.Set("limit", strconv.Itoa(input.Limit))
	}
	request := client.newRequest(ctx, http.MethodGet, "/storage/v1/files", nil)
	request.URL.RawQuery = query.Encode()
	err = client.doJSON(request, http.StatusOK, &output)
	return
}

func (client *HTTPClient) newRequest(ctx context.Context, method string, path string, body io.Reader) *http.Request {
	target := client.baseURL.JoinPath(path)
	request, err := http.NewRequestWithContext(ctx, method, target.String(), body)
	fatal.OnError(err)
	request.Header.Set("Authorization", "Bearer "+client.apiKey)
	return request
}

// do sends the request and maps any unexpected status to an error. On
// success the caller owns the response body.
func (client *HTTPClient) do(request *http.Request, expectedStatusCode int) (*http.Response, error) {
	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != expectedStatusCode {
		defer response.Body.Close()
		return nil, mapResponseError(response)
	}
	return response, nil
}

func (client *HTTPClient) doJSON(request *http.Request, expectedStatusCode int, output any) error {
	response, err := client.do(request, expectedStatusCode)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return json.NewDecoder(response.Body).Decode(output)
}

func mapResponseError(response *http.Response) error {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	message := strings.TrimSpace(string(body))
	switch response.StatusCode {
	case http.StatusBadRequest, http.StatusRequestEntityTooLarge:
		return fmt.Errorf("%w: %s", ErrBadRequest, message)
	case http.StatusUnauthorized:
		return fmt.Errorf("%w: %s", ErrUnauthorized, message)
	case http.StatusNotFound:
		if strings.Contains(response.Request.URL.Path, "/uploads/") {
			return fmt.Errorf("%w: %s", ErrUploadNotFound, message)
		}
		return fmt.Errorf("%w: %s", ErrFileNotFound, message)
	default:
		return fmt.Errorf("%w: %s", ErrServerError, message)
	}
}
