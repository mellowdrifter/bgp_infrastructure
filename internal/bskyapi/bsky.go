package bskyapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
)

const (
	blueskyAPIURL       = "https://bsky.social/xrpc/com.atproto.repo.createRecord"
	BlueskyUploadAPIURL = "https://bsky.social/xrpc/com.atproto.repo.uploadBlob"
	BlueskyAuthURL      = "https://bsky.social/xrpc/com.atproto.server.createSession"
	blueskyRslvURL      = "https://bsky.social/xrpc/com.atproto.identity.resolveHandle?handle="
)

type Account struct {
	Username string
	Handle   string
	Password string
}

type AuthResponse struct {
	AccessJwt string `json:"accessJwt"`
}

type BlobReference struct {
	Type string `json:"$type"`
	Link string `json:"$link"`
}

type BlobResponse struct {
	Blob struct {
		Type     string        `json:"$type"`
		Ref      BlobReference `json:"ref"`
		MimeType string        `json:"mimeType"`
		Size     int           `json:"size"`
	} `json:"blob"`
}

type UploadedImage struct {
	Cfg *image.Config
	Fmt string
	Ref string
}

type CreateRecordRequest struct {
	Collection string      `json:"collection"`
	Repo       string      `json:"repo"`
	Record     interface{} `json:"record"`
}

type ImagePostContent struct {
	Type      string    `json:"$type"`
	Text      string    `json:"text"`
	CreatedAt string    `json:"createdAt"`
	Embed     PostEmbed `json:"embed"`
}

type TextPostContent struct {
	Type      string `json:"$type"`
	Text      string `json:"text"`
	CreatedAt string `json:"createdAt"`
}

type PostContent interface {
	GetText() string
}

type PostEmbed struct {
	Type   string       `json:"$type"`
	Images []EmbedImage `json:"images"`
}

type EmbedImage struct {
	Alt         string      `json:"alt"`
	Image       Blob        `json:"image"`
	AspectRatio AspectRatio `json:"aspectRatio"`
}

type Blob struct {
	Type     string  `json:"$type"`
	Ref      BlobRef `json:"ref"`
	MimeType string  `json:"mimeType"`
	Size     int     `json:"size"`
}

type BlobRef struct {
	Link string `json:"$link"`
}

type AspectRatio struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type Client struct {
	Account Account
	Token   string
}

func NewClient(account Account) *Client {
	return &Client{
		Account: account,
	}
}

func (p ImagePostContent) GetText() string {
	return p.Text
}

func (p TextPostContent) GetText() string {
	return p.Text
}

func (c *Client) Authenticate() (string, error) {

	authReq := map[string]string{
		"identifier": c.Account.Username,
		"password":   c.Account.Password,
	}
	authReqJSON, err := json.Marshal(authReq)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(BlueskyAuthURL, "application/json", bytes.NewBuffer(authReqJSON))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("authentication failed with status: %s", resp.Status)
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", err
	}

	return authResp.AccessJwt, nil

}

func decodeImageConfigFromBytes(data []byte) (*image.Config, string, error) {
	reader := bytes.NewReader(data)
	img, format, err := image.DecodeConfig(reader)
	if err != nil {
		return nil, "", err
	}
	return &img, format, nil
}

// TODO: Token can just go into the client?
func (c *Client) UploadImage(token string, imageData []byte) (UploadedImage, error) {
	fmt.Println("Uploading Image... of size:", len(imageData))

	ul := UploadedImage{}

	cfg, format, err := decodeImageConfigFromBytes(imageData)
	if err != nil {
		return ul, fmt.Errorf("error decoding image: %v", err)
	}
	fmt.Println("Image Format:", format)
	ul.Cfg = cfg
	ul.Fmt = format

	// Create a new bytes buffer to store the image data
	var imgBytes bytes.Buffer
	_, err = io.Copy(&imgBytes, bytes.NewReader(imageData))
	if err != nil {
		return ul, fmt.Errorf("error copying image data: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, BlueskyUploadAPIURL, &imgBytes)
	if err != nil {
		return ul, fmt.Errorf("error creating upload request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "image/"+format)
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(imageData)))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ul, fmt.Errorf("error sending upload request: %v", err)
	}
	defer resp.Body.Close()

	bodyread, _ := io.ReadAll(resp.Body)
	fmt.Println("Upload Response Body:", string(bodyread))

	if resp.StatusCode != http.StatusOK {
		return ul, fmt.Errorf("image upload failed with status: %s", resp.Status)
	}

	var blobResp BlobResponse
	if err := json.Unmarshal(bodyread, &blobResp); err != nil {
		return ul, fmt.Errorf("error parsing upload response: %v", err)
	}

	blobRef := blobResp.Blob.Ref.Link
	fmt.Println("Parsed Blob Reference:", blobRef)
	ul.Ref = blobRef
	return ul, nil
}

func (c *Client) CreatePost(token, repo string, content PostContent) error {

	payload := map[string]interface{}{
		"repo":       repo,
		"collection": "app.bsky.feed.post",
		"record":     content,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling post content: %v", err)
	}
	fmt.Println("Post Payload:", string(payloadBytes))

	req, err := http.NewRequest("POST", blueskyAPIURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Response Body: %s\n", string(body))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("post creation failed with status: %s, response: %s", resp.Status, string(body))
	}

	return nil
}

func (c *Client) GetDID(token, handle string) (string, error) {
	url := blueskyRslvURL + handle

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to resolve handle: %s, response: %s", resp.Status, string(body))
	}

	var response struct {
		DID string `json:"did"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("error decoding response: %v", err)
	}

	return response.DID, nil
}
