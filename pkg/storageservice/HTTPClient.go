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

	"github.com/ansel1/merry"
	"github.com/kduong-dev/goutil/fatal"
	"github.com/kduong-dev/goutil/httpx"
)

type HTTPClient struct {
	baseURL    *url.URL
	apiKey     string
	httpClient *http.Client
}

type NewHTTPClientInput struct {
	Timeout time.Duration
	BaseURL *url.URL
	APIKey  string
}

func NewHTTPClient(input NewHTTPClientInput) *HTTPClient {
	return &HTTPClient{
		baseURL: input.BaseURL,
		apiKey:  input.APIKey,
		httpClient: &http.Client{
			Timeout: input.Timeout,
		},
	}
}

func (client *HTTPClient) InitialiseUpload(ctx context.Context, input InitialiseUploadInput) (output *Upload, err error) {
	requestBody := fatal.UnlessMarshal(input)
	request := client.newRequest(ctx, http.MethodPost, "/storage/v1/uploads", bytes.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	err = client.doJSON(request, http.StatusCreated, &output)
	return
}

func (client *HTTPClient) UploadPart(ctx context.Context, input UploadPartInput) (output *UploadPartOutput, err error) {
	path := fmt.Sprintf("/storage/v1/uploads/%s/parts/%d", url.PathEscape(input.UploadID), input.PartNumber)
	request := client.newRequest(ctx, http.MethodPut, path, input.Body)
	request.Header.Set("Content-Type", "application/octet-stream")
	err = client.doJSON(request, http.StatusOK, &output)
	return
}

func (client *HTTPClient) CompleteUpload(ctx context.Context, input CompleteUploadInput) (output *File, err error) {
	path := fmt.Sprintf("/storage/v1/uploads/%s/complete", url.PathEscape(input.UploadID))
	request := client.newRequest(ctx, http.MethodPost, path, nil)
	err = client.doJSON(request, http.StatusCreated, &output)
	return
}

func (client *HTTPClient) AbortUpload(ctx context.Context, input AbortUploadInput) error {
	path := fmt.Sprintf("/storage/v1/uploads/%s/abort", url.PathEscape(input.UploadID))
	request := client.newRequest(ctx, http.MethodPost, path, nil)
	response, err := client.do(request, http.StatusNoContent)
	if err != nil {
		return err
	}
	return response.Body.Close()
}

func (client *HTTPClient) DownloadFile(ctx context.Context, input DownloadFileInput) (output *DownloadFileResponse, err error) {
	path := fmt.Sprintf("/storage/v1/files/%s", url.PathEscape(input.FileID))
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
		return nil, responseError(response)
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

// responseError tags the error httpx.ResponseError builds with the matching
// sentinel, keeping its HTTP status code and the server's user message.
func responseError(response *http.Response) error {
	err := httpx.ResponseError(response)
	if err == nil {
		err = merry.Errorf("unexpected status code %d", response.StatusCode).WithHTTPCode(response.StatusCode)
	}
	switch response.StatusCode {
	case http.StatusBadRequest, http.StatusRequestEntityTooLarge:
		return merry.WithCause(err, ErrBadRequest)
	case http.StatusUnauthorized:
		return merry.WithCause(err, ErrUnauthorized)
	case http.StatusNotFound:
		if strings.Contains(response.Request.URL.Path, "/uploads/") {
			return merry.WithCause(err, ErrUploadNotFound)
		}
		return merry.WithCause(err, ErrFileNotFound)
	default:
		return merry.WithCause(err, ErrServerError)
	}
}
