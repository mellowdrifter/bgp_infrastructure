package xapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/dghubble/oauth1"
)

type Client struct {
	httpClient  *http.Client
	apiKey      string
	apiSecret   string
	token       string
	tokenSecret string
}

// NewClient creates a new Twitter API client
func NewClient(consumerKey, consumerSecret, accessToken, accessSecret string) *Client {
	config := oauth1.NewConfig(consumerKey, consumerSecret)
	token := oauth1.NewToken(accessToken, accessSecret)
	httpClient := config.Client(oauth1.NoContext, token)
	return &Client{
		httpClient:  httpClient,
		apiKey:      consumerKey,
		apiSecret:   consumerSecret,
		token:       accessToken,
		tokenSecret: accessSecret,
	}
}

// UploadImage uploads an image and returns media_id
func (c *Client) UploadImage(imagePath string) (string, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	part, err := writer.CreateFormFile("media", filepath.Base(imagePath))
	if err != nil {
		return "", err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return "", err
	}
	writer.Close()

	req, err := http.NewRequest("POST", "https://upload.twitter.com/1.1/media/upload.json", &b)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res struct {
		MediaID int64 `json:"media_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	if res.MediaID == 0 {
		return "", errors.New("failed to upload media")
	}
	return fmt.Sprintf("%d", res.MediaID), nil
}

// PostTweet posts a tweet with optional media IDs
func (c *Client) PostTweet(text string, mediaIDs ...string) (string, error) {
	body := map[string]interface{}{
		"text": text,
	}
	if len(mediaIDs) > 0 {
		body["media"] = map[string]interface{}{
			"media_ids": mediaIDs,
		}
	}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", "https://api.twitter.com/2/tweets", bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.bearerHeader())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to post tweet: %s", string(bodyBytes))
	}

	var result struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Data.ID, nil
}

// bearerHeader constructs a dummy bearer token using the OAuth 1.0a token
func (c *Client) bearerHeader() string {
	// This is just a dummy to satisfy Twitter API 2.0 endpoint; actual auth is done via oauth1 client
	return fmt.Sprintf("Bearer dummy-%s", c.token)
}
