package main

import (
	"os"
	"strings"

	"github.com/co191194/memo-app/internal/cli"
	"github.com/co191194/memo-app/internal/storage"
)

func main() {
	app := cli.App{
		Command: &cli.MemoCommandImpl{
			MemoPath:        resolveMemoPath(),
			TimeProvider:    &cli.RealTimeProvider{},
			StorageOperator: &storage.StorageOperatorImpl{},
		},
	}
	exitCode := app.Run(
		os.Stdout,
		os.Stderr,
		os.Args[1:],
	)

	os.Exit(exitCode)
}

func resolveMemoPath() string {
	memoPath := os.Getenv("MEMO_PATH")
	if strings.TrimSpace(memoPath) == "" {
		return "~/.memo/memos.json"
	}
	return memoPath
}
