package mpv

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

type Client struct {
	socketPath string
}

type metadata struct {
	Title  string
	Artist string
	Album  string
}
type Status struct {
	Metadata metadata
	Position time.Duration
	Duration time.Duration
	Paused   bool
	Playing  bool
	metadata map[string]string
}

type command struct {
	Command []any `json:"command"`
}

type propertyResponse[T any] struct {
	Error     string `json:"error"`
	Data      T      `json:"data"`
	RequestID int    `json:"request_id"`
}

func getProperty[T any](ctx context.Context, c *Client, name string) (T, error) {
	var zero T
	cmd := command{
		Command: []any{"get_property", name},
	}

	var res propertyResponse[T]
	if err := c.send(ctx, cmd, &res); err != nil {
		return zero, err
	}

	if res.Error != "" && res.Error != "success" {
		return zero, fmt.Errorf("mpv: get_property %s: %s", name, res.Error)
	}

	return res.Data, nil
}

// Helper to send calls to mpv
func (c *Client) send(ctx context.Context, cmd any, out any) error {
	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "unix", c.socketPath)
	if err != nil {
		return fmt.Errorf("Failed to create unix socket at path: %s: %w", c.socketPath, err)
	}
	defer conn.Close()

	enc := json.NewEncoder(conn)
	if err := enc.Encode(cmd); err != nil {
		return fmt.Errorf("Failed to encode mpv command: %w", err)
	}

	if out == nil {
		return nil
	}

	dec := json.NewDecoder(conn)
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("Failed to decode out: %w", err)
	}

	return nil
}

func NewClient(socketPath string) *Client {
	return &Client{socketPath: socketPath}
}

// Plays current song based on url
func (c *Client) Play(ctx context.Context, url string) error {
	cmd := command{
		Command: []any{"loadfile", url, "replace"},
	}

	return c.send(ctx, cmd, nil)
}

// Gets current status of song from mpv
func (c *Client) GetStatus(ctx context.Context) (*Status, error) {
	var stat Status

	posSec, err := getProperty[float64](ctx, c, "time-pos")
	if err != nil {
		return nil, err
	}
	durSec, err := getProperty[float64](ctx, c, "duration")
	if err != nil {
		return nil, err
	}
	meta, err := c.getMetadata(ctx)
	if err != nil {
		return nil, err
	}
	paused, err := getProperty[bool](ctx, c, "pause")
	if err != nil {
		return nil, err
	}

	stat.Paused = paused
	stat.Position = time.Duration(posSec * float64(time.Second))
	stat.Duration = time.Duration(durSec * float64(time.Second))
	stat.Playing = !paused && durSec > 0
	stat.Metadata = meta

	return &stat, nil
}

func (c *Client) getMetadata(ctx context.Context) (metadata, error) {
	var out metadata
	md, err := getProperty[map[string]string](ctx, c, "metadata")
	if err != nil {
		return out, fmt.Errorf("mpv: error getting metadata from mpv: %w", err)
	}

	if artist, ok := md["artist"]; ok {
		out.Artist = artist
	}
	if album, ok := md["album"]; ok {
		out.Album = album
	}
	if title, ok := md["title"]; ok {
		out.Title = title
	}

	return out, nil
}

// EnsureDaemon
