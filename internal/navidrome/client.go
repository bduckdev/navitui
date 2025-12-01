package navidrome

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
)

type Config struct {
	BaseURL  string
	Username string
	Password string
}

type Client struct {
	baseURL    *url.URL
	username   string
	password   string
	apiVersion string
	clientName string

	httpClient *http.Client
}

type Artist struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Albums []Album `json:"album"`
}

type Album struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ArtistID string `json:"artistId"`
	Artist   string `json:"artist"`
	Genre    string `json:"genre"`
	Songs    []Song `json:"song"`
}

type Song struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	ArtistID string `json:"artistId"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	Genre    string `json:"genre"`
}

func NewClient(cfg Config) (*Client, error) {
	if cfg.BaseURL == "" {
		return nil, errors.New("navidrome: BaseURL is required")
	}
	if cfg.Username == "" {
		return nil, errors.New("navidrome: Username is required")
	}
	if cfg.Password == "" {
		return nil, errors.New("navidrome: Password is required")
	}

	u, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, errors.New("navidrome: Invalid BaseURL")
	}

	return &Client{
		baseURL:    u,
		username:   cfg.Username,
		password:   cfg.Password,
		apiVersion: "1.16.1",
		clientName: "navitui",
		httpClient: http.DefaultClient,
	}, nil
}

// private helper for encoding urls
func (c *Client) encodeURL(endpoint string, extraQuery func(q url.Values)) url.URL {
	u := *c.baseURL
	u.Path = path.Join(u.Path, "rest", endpoint+".view")

	q := u.Query()
	q.Set("u", c.username)
	q.Set("p", c.password)
	q.Set("v", c.apiVersion)
	q.Set("c", c.clientName)
	q.Set("f", "json")
	if extraQuery != nil {
		extraQuery(q)
	}

	u.RawQuery = q.Encode()

	return u
}

// Generic helper function for interacting wit the navidrome API
func (c *Client) do(
	ctx context.Context,
	endpoint string,
	extraQuery func(q url.Values),
	dest any,
) error {
	u := c.encodeURL(endpoint, extraQuery)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("navidrome: build request: %w", err)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("navidrome: do request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("navidrome: unexpected response status: %x", res.StatusCode)
	}
	if err := json.NewDecoder(res.Body).Decode(dest); err != nil {
		return fmt.Errorf("navidrome: failed to decode response body: %w", err)
	}

	return nil
}

type pingResponse struct {
	Response struct {
		Status string `json:"status"`
		Error  struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	} `json:"subsonic-response"`
}

func (c *Client) Ping(ctx context.Context) error {
	var pr pingResponse
	if err := c.do(ctx, "ping", nil, &pr); err != nil {
		return err
	}
	if pr.Response.Status != "ok" {
		if pr.Response.Error.Message != "" {
			return fmt.Errorf("navidrome: ping: %s (code %d)",
				pr.Response.Error.Message, pr.Response.Error.Code)
		}
		return fmt.Errorf("navidrome: ping: status=%s", pr.Response.Status)
	}

	return nil
}

type getAlbumListResponse struct {
	Response struct {
		Status    string `json:"status"`
		AlbumList struct {
			Albums []Album `json:"album"`
		} `json:"albumList2"`
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	} `json:"subsonic-response"`
}

// Get 10 albums from list based on offset
func (c *Client) GetAlbumList(ctx context.Context, offset int) error {
	var ar getAlbumListResponse
	if err := c.do(ctx, "getAlbumList2", func(q url.Values) {
		q.Set("size", "10")
		q.Set("type", "alphabeticalByArtist")
		q.Set("offset", strconv.Itoa(offset))
	}, &ar); err != nil {
		return err
	}

	return nil
}

type getArtistsResponse struct {
	Response struct {
		Status  string `json:"status"`
		Artists struct {
			Index []struct {
				Letter  string   `json:"name"`
				Artists []Artist `json:"artist"`
			} `json:"index"`
		} `json:"artists"`
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	} `json:"subsonic-response"`
}

// Get all artists
func (c *Client) GetArtists(ctx context.Context) ([]Artist, error) {
	var ar getArtistsResponse
	if err := c.do(ctx, "getArtists", func(q url.Values) {
	}, &ar); err != nil {
		return nil, err
	}

	var out []Artist

	for _, a := range ar.Response.Artists.Index {
		out = append(out, a.Artists...)
	}

	return out, nil
}

type getAlbumResponse struct {
	Response struct {
		Status    string `json:"status"`
		AlbumData Album  `json:"album"`
		Error     struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	} `json:"subsonic-response"`
}

// get songs from album that matches id
func (c *Client) GetSongs(ctx context.Context, id string) ([]Song, error) {
	var ar getAlbumResponse
	if err := c.do(ctx, "getAlbum", func(q url.Values) {
		q.Set("id", id)
	}, &ar); err != nil {
		return ar.Response.AlbumData.Songs, err
	}

	return ar.Response.AlbumData.Songs, nil
}

type getAlbumsByArtistResponse struct {
	Response struct {
		Status string `json:"status"`
		Artist struct {
			Albums []Album `json:"album"`
		} `json:"artist"`
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	} `json:"subsonic-response"`
}

// get albums by artist that matches id
func (c *Client) GetAlbumsByArtist(ctx context.Context, id string) ([]Album, error) {
	var ar getAlbumsByArtistResponse
	if err := c.do(ctx, "getArtist", func(q url.Values) {
		q.Set("id", id)
	}, &ar); err != nil {
		return []Album{}, err
	}

	return ar.Response.Artist.Albums, nil
}

// bulid stream URL for a given song based on it's ID
func (c *Client) BuildStreamURL(id string) string {
	u := c.encodeURL("stream", func(q url.Values) {
		q.Set("id", id)
	})

	return u.String()
}
