package shutils

import (
	"context"
	"io"
	"os"
	"os/exec"
)

func NopReadDir(context.Context, string) ([]os.FileInfo, error) {
	return nil, os.ErrNotExist
}

func NopStat(context.Context, string, bool) (os.FileInfo, error) {
	return nil, os.ErrNotExist
}

func NopExec(context.Context, []string) error {
	return exec.ErrNotFound
}

func NopOpen(context.Context, string, int, os.FileMode) (io.ReadWriteCloser, error) {
	return NopRWC{}, nil
}

type NopRWC struct{}

func (NopRWC) Read([]byte) (int, error) {
	return 0, os.ErrClosed
}

func (NopRWC) Write([]byte) (int, error) {
	return 0, os.ErrClosed
}

func (NopRWC) Close() error {
	return nil
}
