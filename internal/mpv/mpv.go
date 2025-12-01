package mpv

import (
	"context"
	"os/exec"
	"sync"
)

type Player struct {
	mu  sync.Mutex
	cmd *exec.Cmd
	bin string
}

func New(bin string) *Player {
	if bin == "" {
		bin = "mpv"
	}
	return &Player{bin: bin}
}

func (p *Player) Play(ctx context.Context, url string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
		p.cmd = nil
	}

	cmd := exec.CommandContext(ctx, p.bin,
		"--no-video",
		"--no-terminal",
		url,
	)

	cmd.Stdout = nil
	cmd.Stdin = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return err
	}

	return nil
}

func (p *Player) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	return nil
}
