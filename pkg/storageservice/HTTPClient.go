package storageservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kduong/trading-backend/internal/contextx"
)

type HTTPClient struct {
	baseURL    url.URL
	httpClient *http.Client
}

type NewHTTPClientInput struct {
	Timeout time.Duration
	BaseURL url.URL
}

func NewHTTPClient(input NewHTTPClientInput) *HTTPClient {
	return &HTTPClient{
		baseURL: input.BaseURL,
		httpClient: &http.Client{
			Timeout: input.Timeout,
		},
	}
}

type initialiseUploadRequestBody struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
}

func (client *HTTPClient) InitialiseUpload(ctx context.Context, filename string, contentType string) (output *Upload, err error) {
	target := url.URL{
		Scheme: client.baseURL.Scheme,
		Host:   client.baseURL.Host,
		Path:   "/storage/v1/uploads",
	}
	requestBody := initialiseUploadRequestBody{
		Filename:    filename,
		ContentType: contentType,
	}
	encodedBody, err := json.Marshal(requestBody)
	if err != nil {
		panic(err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), bytes.NewReader(encodedBody))
	if err != nil {
		panic(err)
	}
	request.Header.Set("Content-Type", "application/json")
	accessToken := contextx.GetAccessToken(ctx)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response, err := client.httpClient.Do(request)
	if err != nil {
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		err = client.mapResponseError(response)
		return
	}
	err = json.NewDecoder(response.Body).Decode(&output)
	return
}

func (client *HTTPClient) UploadPart(ctx context.Context, uploadID string, partNumber int, body io.Reader) (output *UploadPartResponse, err error) {
	target := url.URL{
		Scheme: client.baseURL.Scheme,
		Host:   client.baseURL.Host,
		Path:   fmt.Sprintf("/storage/v1/uploads/%s/parts/%d", uploadID, partNumber),
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, target.String(), body)
	if err != nil {
		panic(err)
	}
	request.Header.Set("Content-Type", "application/octet-stream")
	accessToken := contextx.GetAccessToken(ctx)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response, err := client.httpClient.Do(request)
	if err != nil {
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		err = client.mapResponseError(response)
		return
	}
	err = json.NewDecoder(response.Body).Decode(&output)
	return
}

func (client *HTTPClient) CompleteUpload(ctx context.Context, uploadID string) (output *File, err error) {
	target := url.URL{
		Scheme: client.baseURL.Scheme,
		Host:   client.baseURL.Host,
		Path:   fmt.Sprintf("/storage/v1/uploads/%s/complete", uploadID),
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), nil)
	if err != nil {
		panic(err)
	}
	accessToken := contextx.GetAccessToken(ctx)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response, err := client.httpClient.Do(request)
	if err != nil {
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		err = client.mapResponseError(response)
		return
	}
	err = json.NewDecoder(response.Body).Decode(&output)
	return
}

func (client *HTTPClient) DownloadFile(ctx context.Context, fileID string) (output *DownloadFileResponse, err error) {
	target := url.URL{
		Scheme: client.baseURL.Scheme,
		Host:   client.baseURL.Host,
		Path:   fmt.Sprintf("/storage/v1/files/%s", fileID),
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		panic(err)
	}
	accessToken := contextx.GetAccessToken(ctx)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response, err := client.httpClient.Do(request)
	if err != nil {
		return
	}
	if response.StatusCode != http.StatusOK {
		defer response.Body.Close()
		err = client.mapResponseError(response)
		return
	}
	output = &DownloadFileResponse{
		ContentType:        response.Header.Get("Content-Type"),
		ContentDisposition: response.Header.Get("Content-Disposition"),
		Body:               response.Body,
	}
	return
}

func (client *HTTPClient) mapResponseError(response *http.Response) (err error) {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return
	}
	message := strings.TrimSpace(string(body))
	switch response.StatusCode {
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s", ErrFileNotFound, message)
	case http.StatusForbidden:
		return fmt.Errorf("%w: %s", ErrFileForbidden, message)
	case http.StatusConflict:
		return fmt.Errorf("%w: %s", ErrUploadNotActive, message)
	default:
		return fmt.Errorf("%w: %s", ErrServerError, message)
	}
}
