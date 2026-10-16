package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/co191194/memo-app/internal/memo"
)

type MemoCommand interface {
	AddMemo(stdout io.Writer, stderr io.Writer, args []string) int
	ListMemos(stdout io.Writer, stderr io.Writer, args []string) int
	ShowMemo(stdout io.Writer, stderr io.Writer, args []string) int
	SearchMemos(stdout io.Writer, stderr io.Writer, args []string) int
	DeleteMemo(stdout io.Writer, stderr io.Writer, args []string) int
	EditMemo(stdout io.Writer, stderr io.Writer, args []string) int
}

type TimeProvider interface {
	Now() time.Time
}

type RealTimeProvider struct {
}

func (rtp *RealTimeProvider) Now() time.Time {
	return time.Now()
}

type Memo = memo.Memo

type StorageOperator interface {
	LoadMemos(path string) ([]Memo, error)
	SaveMemos(path string, memos []Memo) error
}

type MemoCommandImpl struct {
	MemoPath        string
	TimeProvider    TimeProvider
	StorageOperator StorageOperator
}

type addOptions struct {
	Title string
	Body  string
}

func (cmd *MemoCommandImpl) AddMemo(stdout io.Writer, stderr io.Writer, args []string) int {

	options, err := parseAddArgs(stderr, args)
	if err != nil {
		return resolveParseErrorExitCode(err)
	}

	memos, err := cmd.StorageOperator.LoadMemos(cmd.MemoPath)
	if err != nil {
		printOpenFileError(stderr, err)
		return 1
	}

	now := cmd.TimeProvider.Now()

	addedMemo := memo.CreateMemo(memos, options.Title, options.Body, now)

	memos = append(memos, addedMemo)

	if err := cmd.StorageOperator.SaveMemos(cmd.MemoPath, memos); err != nil {
		fmt.Fprintln(stderr, "メモの保存に失敗しました", err)
		return 1
	}

	return 0
}

func parseAddArgs(stderr io.Writer, args []string) (addOptions, error) {
	var options addOptions

	fs := newCommandFlagSet(
		"add",
		"memo add <title> [--body <body>]",
		stderr,
	)

	fs.StringVar(
		&options.Body,
		"body",
		"",
		"メモの本文",
	)

	positionals, err := parseCommandArgs(fs, args, 1)
	if err != nil {
		return addOptions{}, err
	}

	if isEmpty(positionals[0]) {
		printEmptyError(stderr, "タイトル")
		return addOptions{}, fmt.Errorf("title must be non-empty")
	}

	options.Title = positionals[0]

	return options, nil
}

func (cmd *MemoCommandImpl) ListMemos(stdout io.Writer, stderr io.Writer, args []string) int {
	err := parseListArgs(stderr, args)
	if err != nil {
		return resolveParseErrorExitCode(err)
	}

	memos, err := cmd.StorageOperator.LoadMemos(cmd.MemoPath)
	if err != nil {
		printOpenFileError(stderr, err)
		return 1
	}

	if len(memos) == 0 {
		fmt.Fprintln(stdout, "No memos found.")
	} else {
		for _, memo := range memos {
			printMemoForList(stdout, memo)
		}
	}
	return 0
}

func parseListArgs(stderr io.Writer, args []string) error {
	fs := newCommandFlagSet("list", "memo list", stderr)

	_, err := parseCommandArgs(fs, args, 0)

	return err
}

const DATE_TIME_FORMAT = "2006-01-02 15:04"

type showOptions struct {
	ID int
}

func (cmd *MemoCommandImpl) ShowMemo(stdout io.Writer, stderr io.Writer, args []string) int {
	options, err := parseShowArgs(stderr, args)
	if err != nil {
		return resolveParseErrorExitCode(err)
	}

	memos, err := cmd.StorageOperator.LoadMemos(cmd.MemoPath)
	if err != nil {
		printOpenFileError(stderr, err)
		return 1
	}

	memo, err := memo.FindById(memos, options.ID)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	fmt.Fprintln(stdout, "# "+memo.Title)
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "ID: "+strconv.Itoa(memo.ID))
	fmt.Fprintln(stdout, "Created: "+memo.CreatedAt.Format(DATE_TIME_FORMAT))
	fmt.Fprintln(stdout, "Updated: "+memo.UpdatedAt.Format(DATE_TIME_FORMAT))
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, memo.Body)
	return 0
}

func parseShowArgs(stderr io.Writer, args []string) (showOptions, error) {
	var options showOptions

	fs := newCommandFlagSet("show", "memo show <id>", stderr)

	positionals, err := parseCommandArgs(fs, args, 1)
	if err != nil {
		return showOptions{}, err
	}

	id, err := resolveId(positionals[0])
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return showOptions{}, err
	}
	options.ID = id

	return options, nil
}

type searchOptions struct {
	Keyword string
}

func (cmd *MemoCommandImpl) SearchMemos(stdout io.Writer, stderr io.Writer, args []string) int {

	options, err := parseSearchArgs(stderr, args)
	if err != nil {
		return resolveParseErrorExitCode(err)
	}

	memos, err := cmd.StorageOperator.LoadMemos(cmd.MemoPath)
	if err != nil {
		printOpenFileError(stderr, err)
		return 1
	}

	filteredMemos, err := memo.FilterMemos(memos, options.Keyword)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	for _, memo := range filteredMemos {
		printMemoForList(stdout, memo)
	}

	return 0
}

func parseSearchArgs(stderr io.Writer, args []string) (searchOptions, error) {
	var options searchOptions
	fs := newCommandFlagSet("search", "memo search <keyword>", stderr)

	positionals, err := parseCommandArgs(fs, args, 1)
	if err != nil {
		return searchOptions{}, err
	}

	if isEmpty(positionals[0]) {
		printEmptyError(stderr, "キーワード")
		return searchOptions{}, fmt.Errorf("keyword must be non-empty")
	}

	options.Keyword = positionals[0]

	return options, nil
}

type deleteOptions struct {
	ID int
}

func (cmd *MemoCommandImpl) DeleteMemo(stdout io.Writer, stderr io.Writer, args []string) int {

	options, err := parseDeleteArgs(stderr, args)
	if err != nil {
		return resolveParseErrorExitCode(err)
	}

	memos, err := cmd.StorageOperator.LoadMemos(cmd.MemoPath)
	if err != nil {
		printOpenFileError(stderr, err)
		return 1
	}

	newMemos, err := memo.BuildDeletedMemos(memos, options.ID)

	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	if err := cmd.StorageOperator.SaveMemos(cmd.MemoPath, newMemos); err != nil {
		fmt.Fprintln(stderr, "メモを削除できませんでした", err)
		return 1
	}
	return 0
}

func parseDeleteArgs(stderr io.Writer, args []string) (deleteOptions, error) {
	var options deleteOptions

	fs := newCommandFlagSet("delete", "memo delete <id>", stderr)

	positionals, err := parseCommandArgs(fs, args, 1)
	if err != nil {
		return deleteOptions{}, err
	}

	id, err := resolveId(positionals[0])
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return deleteOptions{}, err
	}

	options.ID = id

	return options, nil
}

type editOptions struct {
	ID    int
	Title *string
	Body  *string
}

func (cmd *MemoCommandImpl) EditMemo(stdout io.Writer, stderr io.Writer, args []string) int {

	options, err := parseEditArgs(stderr, args)
	if err != nil {
		return resolveParseErrorExitCode(err)
	}

	memos, err := cmd.StorageOperator.LoadMemos(cmd.MemoPath)
	if err != nil {
		printOpenFileError(stderr, err)
		return 1
	}

	input := parseEditInput(options, cmd.TimeProvider.Now())

	editedMemos, err := memo.EditMemo(memos, input)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	if err := cmd.StorageOperator.SaveMemos(cmd.MemoPath, editedMemos); err != nil {
		fmt.Fprintln(stderr, "メモの保存に失敗しました", err)
		return 1
	}

	return 0
}

func parseEditArgs(stderr io.Writer, args []string) (editOptions, error) {
	var options editOptions

	fs := newCommandFlagSet("edit", "memo edit <id> [--title <title>] [--body <body>]", stderr)

	var title, body string
	fs.StringVar(
		&title,
		"title",
		"",
		"メモのタイトル",
	)

	fs.StringVar(
		&body,
		"body",
		"",
		"メモの本文",
	)

	positionals, err := parseCommandArgs(fs, args, 1)
	if err != nil {
		return editOptions{}, err
	}

	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "title":
			options.Title = &title
		case "body":
			options.Body = &body
		}
	})

	if options.Title == nil && options.Body == nil {
		fmt.Fprintln(stderr, "-titleまたは-bodyを指定してください")
		fs.Usage()
		return editOptions{}, fmt.Errorf("not exists title and body options")
	}

	if options.Title != nil && isEmpty(*options.Title) {
		printEmptyError(stderr, "タイトル")
		return editOptions{}, fmt.Errorf("title is empty")
	}

	id, err := resolveId(positionals[0])
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return editOptions{}, err
	}

	options.ID = id

	return options, nil
}

func parseEditInput(options editOptions, now time.Time) memo.EditMemoInput {
	return memo.EditMemoInput{
		ID:    options.ID,
		Title: options.Title,
		Body:  options.Body,
		Now:   now,
	}
}

func printOpenFileError(stderr io.Writer, err error) {
	fmt.Fprintln(stderr, "メモを開くことができませんでした", err)
}

func printMemoForList(stdout io.Writer, memo Memo) {
	fmt.Fprintf(stdout, "%d %s\t%s\n", memo.ID, memo.Title, memo.CreatedAt.Format("2006-01-02"))
}

func printEmptyError(stderr io.Writer, name string) {
	fmt.Fprintln(stderr, name+"を入力してください")
}

func resolveId(id string) (int, error) {
	resolveId, err := strconv.Atoi(id)
	if err != nil {
		return -1, errors.New("idは数値を入力してください: " + id)
	}
	if resolveId <= 0 {
		return -1, errors.New("idは1以上で指定してください")
	}
	return resolveId, nil
}

func isEmpty(str string) bool {
	return strings.TrimSpace(str) == ""
}

func resolveParseErrorExitCode(err error) int {
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	return 1
}
