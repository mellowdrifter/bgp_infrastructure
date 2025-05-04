package bskyapi

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"gopkg.in/ini.v1"
)

func setup() (*Client, string, string) {
	cf, err := ini.ShadowLoad("config.ini")
	if err != nil {
		panic(err)
	}
	user := cf.Section("bsky").Key("username").String()
	hand := cf.Section("bsky").Key("handle").String()
	pass := cf.Section("bsky").Key("password").String()

	// Create a new account
	c := NewClient(
		Account{
			Username: user,
			Handle:   hand,
			Password: pass,
		},
	)
	// Authenticate
	token, err := c.Authenticate()
	if err != nil {
		panic(err)
	}
	did, err := c.GetDID(token, hand)

	return c, token, did
}

func TestPostBsky(t *testing.T) {

	c, token, did := setup()

	post := TextPostContent{
		Type:      "app.bsky.feed.post",
		Text:      "testing",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Post it
	if err := c.CreatePost(token, did, post); err != nil {
		panic(err)
	}

}

func TestPostBskyPNG(t *testing.T) {
	c, token, did := setup()

	post := ImagePostContent{
		Type:      "app.bsky.feed.post",
		Text:      "test post with png",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	media, err := os.ReadFile("test.png")
	if err != nil {
		panic(err)
	}

	// Upload image
	ul, err := c.UploadImage(token, media)
	if err != nil {
		panic(err)
	}
	post.Embed = PostEmbed{
		Type: "app.bsky.embed.images",
		Images: []EmbedImage{
			{
				Image: Blob{
					Type:     "blob",
					Ref:      BlobRef{Link: ul.Ref},
					MimeType: fmt.Sprintf("image/%s", ul.Fmt),
				},
				AspectRatio: AspectRatio{Width: ul.Cfg.Width, Height: ul.Cfg.Height},
			},
		},
	}

	// Post it
	if err := c.CreatePost(token, did, post); err != nil {
		panic(err)
	}
}

func TestPostBskyJPG(t *testing.T) {
	c, token, did := setup()

	post := ImagePostContent{
		Type:      "app.bsky.feed.post",
		Text:      "testing post with jpg",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	media, err := os.ReadFile("test.jpg")
	if err != nil {
		panic(err)
	}
	var b bytes.Buffer
	io.Copy(&b, bytes.NewReader(media))

	// Upload image
	ul, err := c.UploadImage(token, b.Bytes())
	if err != nil {
		panic(err)
	}
	post.Embed = PostEmbed{
		Type: "app.bsky.embed.images",
		Images: []EmbedImage{
			{
				Image: Blob{
					Type:     "blob",
					Ref:      BlobRef{Link: ul.Ref},
					MimeType: fmt.Sprintf("image/%s", ul.Fmt),
					Size:     len(media),
				},
				AspectRatio: AspectRatio{Width: ul.Cfg.Width, Height: ul.Cfg.Height},
			},
		},
	}

	// Post it
	if err := c.CreatePost(token, did, post); err != nil {
		panic(err)
	}
}
