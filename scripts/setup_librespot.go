//go:build ignore

package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	librespotRepo = "https://github.com/devgianlu/go-librespot.git"
	librespotTag  = "v0.7.1"
	destRel       = "third_party/librespot"
	overlayRel    = "patches/go-librespot"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "setup_librespot: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	root, err := repoRoot()
	if err != nil {
		return err
	}
	dest := filepath.Join(root, filepath.FromSlash(destRel))
	overlay := filepath.Join(root, filepath.FromSlash(overlayRel))
	if _, err := os.Stat(filepath.Join(overlay, "vorbis", "decoder.go")); err != nil {
		return fmt.Errorf("overlays missing under %s: %w", overlayRel, err)
	}
	if _, err := os.Stat(filepath.Join(dest, "player", "player.go")); err != nil {
		if err := clone(dest); err != nil {
			return err
		}
	}
	n, err := copyOverlay(overlay, dest)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "setup_librespot: overlaid %d files onto %s@%s\n", n, destRel, librespotTag)
	return tidy(dest)
}

func repoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for d := wd; ; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d, nil
		}
		if filepath.Dir(d) == d {
			return "", fmt.Errorf("go.mod not found from %s", wd)
		}
	}
}

func clone(dest string) error {
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("%s exists but is not a go-librespot tree (missing player/player.go); remove it and re-run", destRel)
	}
	fmt.Fprintf(os.Stderr, "setup_librespot: cloning %s %s\n", librespotRepo, librespotTag)
	cmd := exec.Command("git", "clone", "--depth", "1", "--branch", librespotTag, "--single-branch", librespotRepo, dest)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone: %w", err)
	}
	return nil
}

func copyOverlay(overlay, dest string) (int, error) {
	n := 0
	err := filepath.Walk(overlay, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(overlay, path)
		if err != nil {
			return err
		}
		if strings.EqualFold(info.Name(), "README.md") {
			return nil
		}
		to := filepath.Join(dest, rel)
		if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
			return err
		}
		if err := copyFile(path, to); err != nil {
			return err
		}
		n++
		return nil
	})
	return n, err
}

func copyFile(from, to string) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(to, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func tidy(dest string) error {
	get := exec.Command("go", "get", "github.com/jfreymuth/oggvorbis@v1.0.5", "github.com/mewkiz/flac@v1.0.12")
	get.Dir = dest
	get.Stdout = os.Stderr
	get.Stderr = os.Stderr
	if err := get.Run(); err != nil {
		return fmt.Errorf("go get (librespot): %w", err)
	}
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = dest
	tidy.Stdout = os.Stderr
	tidy.Stderr = os.Stderr
	if err := tidy.Run(); err != nil {
		return fmt.Errorf("go mod tidy (librespot): %w", err)
	}
	return nil
}
