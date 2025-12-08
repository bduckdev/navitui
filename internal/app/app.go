package app

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"navitui/internal/mpv"
	"navitui/internal/navidrome"
	"navitui/internal/tui"

	"golang.org/x/sync/errgroup"
)

var songs []navidrome.Song

func Init() error {
	config := navidrome.Config{
		BaseURL:  os.Getenv("NAVIDROME_URL"),
		Username: os.Getenv("NAVIDROME_USER"),
		Password: os.Getenv("NAVIDROME_PASSWORD"),
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	client, err := navidrome.NewClient(config)
	if err != nil {
		return fmt.Errorf("navidrome not reachable: %v", err)
	}

	artists, err := client.GetArtists(ctx)
	if err != nil {
		return fmt.Errorf("navidrome not reachable: %v", err)
	}

	g, gctx := errgroup.WithContext(ctx)

	g.SetLimit(8)

	var total int64

	for i, artist := range artists {
		g.Go(func() error {
			updated, songCount, err := loadArtist(gctx, client, artist)
			if err != nil {
				return fmt.Errorf("failed to retrieve albums for artist %s: %w", artist.Name, err)
			}

			artists[i] = updated

			atomic.AddInt64(&total, int64(songCount))
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	for _, artist := range artists {
		for _, album := range artist.Albums {
			for _, song := range album.Songs {
				songs = append(songs, song)
				fmt.Printf("ARTIST:%s ALBUM: %s SONG: %s\n", artist.Name, album.Name, song.Title)
			}
		}
	}
	fmt.Printf("Successfully Loaded %d songs\n", total)
	player := mpv.NewClient("/tmp/navitui-mpv.sock")

	playerCtx := context.Background()

	runTUI(songs,
		func(id string) error {
			url := client.BuildStreamURL(id)
			player.Play(playerCtx, url)
			return nil
		},
		func() (*mpv.Status, error) {
			stat, err := player.GetStatus(playerCtx)
			return stat, err
		})

	return nil
}

func loadArtist(ctx context.Context, client *navidrome.Client, artist navidrome.Artist) (navidrome.Artist, int, error) {
	g, gctx := errgroup.WithContext(ctx)

	g.SetLimit(32)

	albums, err := client.GetAlbumsByArtist(ctx, artist.ID)
	if err != nil {
		return navidrome.Artist{}, 0, fmt.Errorf("failed to retrieve albums for artist %s: %w", artist.Name, err)
	}

	var totalSongs int64

	for i := range albums {
		album := &albums[i]
		g.Go(func() error {
			songs, err := client.GetSongs(gctx, album.ID)
			if err != nil {
				return err
			}

			album.Songs = append(album.Songs, songs...)

			atomic.AddInt64(&totalSongs, int64(len(songs)))
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return navidrome.Artist{}, 0, err
	}

	artist.Albums = append(artist.Albums, albums...)

	return artist, int(totalSongs), nil
}

//	func run(args []string) int {
//		if len(args) < 2 {
//			runTUI()
//			return 0
//		}
//		fmt.Printf("nice command bro\n")
//		return 0
//	}
func runTUI(songs []navidrome.Song, onPlay func(id string) error, getNowPlaying func() (*mpv.Status, error)) {
	tui.Run(songs, onPlay, getNowPlaying)
}
