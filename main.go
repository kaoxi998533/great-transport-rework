package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type config struct {
	channelID    string
	videoID      string
	platform     string
	outputDir    string
	limit        int
	sleepSeconds int
}

type uploader interface {
	Upload(path string) error
}

type dummyUploader struct {
	platform string
}

func (u dummyUploader) Upload(path string) error {
	log.Printf("stub upload to %s: %s", u.platform, path)
	return nil
}

func main() {
	log.SetFlags(0)

	cfg, err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}

	if _, err := exec.LookPath("yt-dlp"); err != nil {
		log.Fatal("yt-dlp not found in PATH; install it first (see README for Docker setup)")
	}

	if err := os.MkdirAll(cfg.outputDir, 0o755); err != nil {
		log.Fatal(err)
	}

	var targetURL string
	isChannel := cfg.channelID != ""
	if isChannel {
		targetURL = channelURL(cfg.channelID)
	} else {
		targetURL = videoURL(cfg.videoID)
	}

	ctx := context.Background()
	downloaded, err := download(ctx, targetURL, cfg.outputDir, cfg.limit, time.Duration(cfg.sleepSeconds)*time.Second, isChannel)
	if err != nil {
		log.Fatal(err)
	}
	if len(downloaded) == 0 {
		log.Fatal("no files downloaded; check the ID and try again")
	}

	up := dummyUploader{platform: cfg.platform}
	for _, path := range downloaded {
		if err := up.Upload(path); err != nil {
			log.Printf("upload failed for %s: %v", path, err)
		}
	}
}

func parseFlags() (config, error) {
	var cfg config
	flag.StringVar(&cfg.channelID, "channel-id", "", "YouTube channel ID or URL")
	flag.StringVar(&cfg.videoID, "video-id", "", "YouTube video ID or URL")
	flag.StringVar(&cfg.platform, "platform", "bilibili", "target platform (bilibili or tiktok)")
	flag.StringVar(&cfg.outputDir, "output", "downloads", "output directory")
	flag.IntVar(&cfg.limit, "limit", 5, "max videos to download for channel")
	flag.IntVar(&cfg.sleepSeconds, "sleep-seconds", 5, "sleep seconds between downloads")
	flag.Parse()

	if cfg.channelID == "" && cfg.videoID == "" {
		return cfg, errors.New("provide either --channel-id or --video-id")
	}
	if cfg.channelID != "" && cfg.videoID != "" {
		return cfg, errors.New("provide only one of --channel-id or --video-id")
	}
	if cfg.channelID != "" && cfg.limit <= 0 {
		return cfg, errors.New("--limit must be > 0 for channel downloads")
	}
	if cfg.sleepSeconds < 0 {
		return cfg, errors.New("--sleep-seconds must be >= 0")
	}

	cfg.platform = strings.ToLower(strings.TrimSpace(cfg.platform))
	switch cfg.platform {
	case "bilibili", "tiktok":
	default:
		return cfg, errors.New("--platform must be bilibili or tiktok")
	}

	return cfg, nil
}

func channelURL(input string) string {
	if looksLikeURL(input) {
		return input
	}
	return fmt.Sprintf("https://www.youtube.com/channel/%s/videos", input)
}

func videoURL(input string) string {
	if looksLikeURL(input) {
		return input
	}
	return fmt.Sprintf("https://www.youtube.com/watch?v=%s", input)
}

func looksLikeURL(input string) bool {
	return strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://")
}

func download(ctx context.Context, targetURL, outputDir string, limit int, sleep time.Duration, isChannel bool) ([]string, error) {
	outputTemplate := filepath.Join(outputDir, "%(title)s.%(ext)s")
	args := []string{
		"--quiet",
		"--no-warnings",
		"--print", "%(filepath)s",
		"-o", outputTemplate,
	}
	if sleep > 0 {
		args = append(args,
			fmt.Sprintf("--sleep-interval=%d", int(sleep.Seconds())),
			fmt.Sprintf("--max-sleep-interval=%d", int(sleep.Seconds())+1),
		)
	}
	if isChannel {
		args = append(args, "--playlist-items", fmt.Sprintf("1:%d", limit))
	}
	args = append(args, targetURL)

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var files []string
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			files = append(files, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return files, err
	}
	if err := cmd.Wait(); err != nil {
		return files, fmt.Errorf("yt-dlp failed: %w", err)
	}

	return files, nil
}
