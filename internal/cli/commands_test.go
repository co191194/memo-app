package cli_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/co191194/memo-app/internal/cli"
	"github.com/co191194/memo-app/internal/memo"
	"github.com/co191194/memo-app/internal/storage"
)

type fakeTimeProvider struct {
}

func (ftp *fakeTimeProvider) Now() time.Time {
	return time.Date(2026, 7, 14, 10, 0, 0, 0, time.Local)
}

type fixedTimeProvider struct {
	now time.Time
}

func (ftp *fixedTimeProvider) Now() time.Time {
	return ftp.now
}

type Memo = memo.Memo
type MemoCommandImpl = cli.MemoCommandImpl
type RealTimeProvider = cli.RealTimeProvider

func TestAddMemo(t *testing.T) {
	t.Run("引数チェック", func(t *testing.T) {

		testCases := []struct {
			testName string
			args     []string
			expected string
			exitCode int
		}{
			{
				testName: "パラメーターが0個の場合",
				args:     []string{},
				expected: "" +
					"Usage:\n" +
					"  memo add <title> [--body <body>]\n" +
					"  -body string\n" +
					"    \tメモの本文\n",
				exitCode: 1,
			},
			{
				testName: "引数が2個の場合",
				args:     []string{"aaa", "bbb"},
				expected: "" +
					"引数が多すぎます\n" +
					"Usage:\n" +
					"  memo add <title> [--body <body>]\n" +
					"  -body string\n" +
					"    \tメモの本文\n",
				exitCode: 1,
			},
			{
				testName: "引数が1個で空文字の場合",
				args:     []string{""},
				expected: "タイトルを入力してください\n",
				exitCode: 1,
			},
			{
				testName: "引数が1個で空白のみの場合",
				args:     []string{"  "},
				expected: "タイトルを入力してください\n",
				exitCode: 1,
			},
			{

				testName: "引数が4個の場合",
				args:     []string{"title", "--body", "body", "xxxxx"},
				expected: "" +
					"引数が多すぎます\n" +
					"Usage:\n" +
					"  memo add <title> [--body <body>]\n" +
					"  -body string\n" +
					"    \tメモの本文\n",
				exitCode: 1,
			},
			{

				testName: "不正なオプションを渡した場合",
				args:     []string{"--unknown"},
				expected: "" +
					"flag provided but not defined: -unknown\n" +
					"Usage:\n" +
					"  memo add <title> [--body <body>]\n" +
					"  -body string\n" +
					"    \tメモの本文\n",
				exitCode: 1,
			},
			{
				testName: "help",
				args:     []string{"--help"},
				expected: "" +
					"Usage:\n" +
					"  memo add <title> [--body <body>]\n" +
					"  -body string\n" +
					"    \tメモの本文\n",
				exitCode: 0,
			},
			{
				testName: "--のみ",
				args:     []string{"--"},
				expected: "" +
					"Usage:\n" +
					"  memo add <title> [--body <body>]\n" +
					"  -body string\n" +
					"    \tメモの本文\n",
				exitCode: 1,
			},
		}

		cmd := MemoCommandImpl{}

		for _, tc := range testCases {
			t.Run(tc.testName, func(t *testing.T) {
				var stdout bytes.Buffer
				var stderr bytes.Buffer
				assertEqualsExitCode(t, cmd.AddMemo(&stdout, &stderr, tc.args), tc.exitCode)
				assertEqualsMessage(t, stderr.String(), tc.expected)
			})
		}
	})

	t.Run("メモの保存に失敗する場合", func(t *testing.T) {

		cmd := MemoCommandImpl{
			MemoPath:        t.TempDir(),
			TimeProvider:    &RealTimeProvider{},
			StorageOperator: &fakeFailSaveMemoOperator{},
		}

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		assertEqualsExitCode(t, cmd.AddMemo(&stdout, &stderr, []string{"Failed Save Memo"}), 1)
		assertEqualsMessage(t, stderr.String(), "メモの保存に失敗しました fake error\n")
	})

	t.Run("AddMemoを実行", func(t *testing.T) {
		testCases := []struct {
			testName      string
			args          []string
			exitCode      int
			expectedTitle string
			expectedBody  string
		}{
			{
				testName:      "タイトルを指定できる",
				args:          []string{"title"},
				exitCode:      0,
				expectedTitle: "title",
				expectedBody:  "",
			},
			{
				testName:      "タイトルの後ろにbodyを指定できる",
				args:          []string{"title", "--body", "body"},
				exitCode:      0,
				expectedTitle: "title",
				expectedBody:  "body",
			},
			{
				testName: "先頭の未定義オプション",
				args:     []string{"--unknown"},
				exitCode: 1,
			},
			{
				testName: "タイトルの後ろの未定義オプション",
				args:     []string{"title", "--unknown"},
				exitCode: 1,
			},
			{
				testName: "オプションが位置引数より前",
				args:     []string{"--body", "body", "title"},
				exitCode: 1,
			},
			{
				testName:      "--の後ろをタイトルとして扱う",
				args:          []string{"--", "--help"},
				exitCode:      0,
				expectedTitle: "--help",
				expectedBody:  "",
			},
			{
				testName:      "--の後ろをタイトルとして扱う2",
				args:          []string{"--body", "body", "--", "-draft"},
				exitCode:      0,
				expectedTitle: "-draft",
				expectedBody:  "body",
			},
			{
				testName:      "bodyで改行ありのメッセージも指定できる",
				args:          []string{"title", "--body", "line 1\nline 2"},
				exitCode:      0,
				expectedTitle: "title",
				expectedBody:  "line 1\nline 2",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.testName, func(t *testing.T) {
				filePath := filepath.Join(t.TempDir(), "memos.json")

				cmd := MemoCommandImpl{
					MemoPath:        filePath,
					TimeProvider:    &RealTimeProvider{},
					StorageOperator: &storage.StorageOperatorImpl{},
				}

				var stdout bytes.Buffer
				var stderr bytes.Buffer
				assertEqualsExitCode(t, cmd.AddMemo(&stdout, &stderr, tc.args), tc.exitCode)

				// 異常系はここまで
				if tc.exitCode == 1 {
					return
				}

				actual, err := realMemoOperator.LoadMemos(filePath)
				if err != nil {
					t.Fatalf("LoadMemos() err = %v", err)
				}

				if len(actual) != 1 {
					t.Fatalf("actual.len = %d, expected = 1", len(actual))
				}

				if actual[0].Title != tc.expectedTitle {
					t.Errorf(
						"Title = %q, expected = %q",
						actual[0].Title,
						tc.expectedTitle,
					)
				}

				if actual[0].Body != tc.expectedBody {
					t.Errorf(
						"Body = %q, expected = %q",
						actual[0].Body,
						tc.expectedBody,
					)
				}
			})
		}

	})
}

type fakeFailSaveMemoOperator struct{}

func (f *fakeFailSaveMemoOperator) LoadMemos(path string) ([]Memo, error) {
	return []Memo{}, nil
}

func (f *fakeFailSaveMemoOperator) SaveMemos(path string, memos []Memo) error {
	return errors.New("fake error")
}

func TestListMemos(t *testing.T) {
	t.Run("引数チェック", func(t *testing.T) {

		testCases := []struct {
			testName string
			args     []string
			expected string
			exitCode int
		}{
			{
				testName: "パラメーターが1個の場合",
				args:     []string{"aaaaa"},
				expected: "" +
					"引数が多すぎます\n" +
					"Usage:\n" +
					"  memo list\n",
				exitCode: 1,
			},
			{
				testName: "不正なオプションを渡した場合",
				args:     []string{"--unknown"},
				expected: "" +
					"flag provided but not defined: -unknown\n" +
					"Usage:\n" +
					"  memo list\n",
				exitCode: 1,
			},
			{
				testName: "help",
				args:     []string{"--help"},
				expected: "" +
					"Usage:\n" +
					"  memo list\n",
				exitCode: 0,
			},
		}

		cmd := MemoCommandImpl{}

		for _, tc := range testCases {
			t.Run(tc.testName, func(t *testing.T) {
				var stdout bytes.Buffer
				var stderr bytes.Buffer
				assertEqualsExitCode(t, cmd.ListMemos(&stdout, &stderr, tc.args), tc.exitCode)
				assertEqualsMessage(t, stderr.String(), tc.expected)
			})
		}
	})
}

var realMemoOperator = storage.StorageOperatorImpl{}

func TestAddMemoAndListMemo(t *testing.T) {

	t.Run("メモが空の場合", func(t *testing.T) {
		filePath := filepath.Join(t.TempDir(), "memos.json")

		var stdout1 bytes.Buffer
		var stderr1 bytes.Buffer

		cmd := MemoCommandImpl{
			MemoPath:        filePath,
			TimeProvider:    &fakeTimeProvider{},
			StorageOperator: &realMemoOperator,
		}

		assertEqualsExitCode(t, cmd.ListMemos(&stdout1, &stderr1, []string{}), 0)
		assertEqualsMessage(t, stdout1.String(), "No memos found.\n")

		var stdout2 bytes.Buffer
		var stderr2 bytes.Buffer
		assertEqualsExitCode(t, cmd.AddMemo(&stdout2, &stderr2, []string{"Add First"}), 0)

		var stdout3 bytes.Buffer
		var stderr3 bytes.Buffer
		assertEqualsExitCode(t, cmd.ListMemos(&stdout3, &stderr3, []string{}), 0)
		assertEqualsMessage(t, stdout3.String(), "1 Add First\t2026-07-14\n")
	})

	t.Run("既存のメモがある場合", func(t *testing.T) {
		filePath := filepath.Join(t.TempDir(), "memos.json")

		cmd := MemoCommandImpl{
			MemoPath:        filePath,
			TimeProvider:    &fakeTimeProvider{},
			StorageOperator: &realMemoOperator,
		}

		existTime := time.Date(2026, 7, 1, 9, 30, 0, 0, time.Local)
		memos := []Memo{
			{ID: 1, Title: "Exist Memo", Body: "", CreatedAt: existTime, UpdatedAt: existTime},
		}

		if err := realMemoOperator.SaveMemos(filePath, memos); err != nil {
			t.Fatalf("SaveMemos() err = %v", err)
		}

		var stdout1 bytes.Buffer
		var stderr1 bytes.Buffer

		assertEqualsExitCode(t, cmd.AddMemo(&stdout1, &stderr1, []string{"Add Second"}), 0)

		var stdout2 bytes.Buffer
		var stderr2 bytes.Buffer

		assertEqualsExitCode(t, cmd.ListMemos(&stdout2, &stderr2, []string{}), 0)

		assertEqualsMessage(
			t,
			stdout2.String(),
			""+
				"1 Exist Memo\t2026-07-01\n"+
				"2 Add Second\t2026-07-14\n",
		)

	})

}

func TestShowMemo(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "memos.json")
	memos := []Memo{
		{ID: 1, Title: "Other"},
		{
			ID:        2,
			Title:     "Target",
			Body:      "Test Body",
			CreatedAt: time.Date(2026, 7, 15, 10, 15, 30, 45, time.Local),
			UpdatedAt: time.Date(2026, 8, 20, 11, 12, 13, 14, time.Local),
		},
		{ID: 3, Title: "Other"},
	}

	if err := realMemoOperator.SaveMemos(filePath, memos); err != nil {
		t.Fatalf("SaveMemos() err = %v", err)
	}

	cmd := MemoCommandImpl{
		MemoPath:        filePath,
		TimeProvider:    &fakeTimeProvider{},
		StorageOperator: &realMemoOperator,
	}

	t.Run("指定IDのメモが存在する場合", func(t *testing.T) {

		var stdout bytes.Buffer
		var stderr bytes.Buffer

		exitCode := cmd.ShowMemo(&stdout, &stderr, []string{"2"})
		assertEqualsExitCode(t, exitCode, 0)

		expected := "" +
			"# Target\n" +
			"\n" +
			"ID: 2\n" +
			"Created: 2026-07-15 10:15\n" +
			"Updated: 2026-08-20 11:12\n" +
			"\n" +
			"Test Body\n"

		assertEqualsMessage(t, stdout.String(), expected)
	})

	t.Run("指定IDのメモが存在しない場合", func(t *testing.T) {

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		exitCode := cmd.ShowMemo(&stdout, &stderr, []string{"4"})

		assertEqualsExitCode(t, exitCode, 1)
		assertEqualsMessage(t, stderr.String(), "memo not found: 4\n")
	})

	t.Run("引数チェック", func(t *testing.T) {

		testCases := []struct {
			testName string
			args     []string
			expected string
			exitCode int
		}{
			{
				testName: "パラメーターが0個の場合",
				args:     []string{},
				expected: "" +
					"Usage:\n" +
					"  memo show <id>\n",
				exitCode: 1,
			},
			{
				testName: "パラメーターが2個の場合",
				args:     []string{"aaa", "bbb"},
				expected: "" +
					"引数が多すぎます\n" +
					"Usage:\n" +
					"  memo show <id>\n",
				exitCode: 1,
			},
			{
				testName: "パラメーターが数値でない場合",
				args:     []string{"a"},
				expected: "idは数値を入力してください: a\n",
				exitCode: 1,
			},
			{
				testName: "不正なオプションを渡した場合",
				args:     []string{"--unknown"},
				expected: "" +
					"flag provided but not defined: -unknown\n" +
					"Usage:\n" +
					"  memo show <id>\n",
				exitCode: 1,
			},
			{
				testName: "help",
				args:     []string{"--help"},
				expected: "" +
					"Usage:\n" +
					"  memo show <id>\n",
				exitCode: 0,
			},
			{
				testName: "idが0の場合",
				args:     []string{"0"},
				expected: "idは1以上で指定してください\n",
				exitCode: 1,
			},
			{
				testName: "idが-1の場合",
				args:     []string{"--", "-1"},
				expected: "idは1以上で指定してください\n",
				exitCode: 1,
			},
		}

		cmd := MemoCommandImpl{}

		for _, tc := range testCases {
			t.Run(tc.testName, func(t *testing.T) {
				var stdout bytes.Buffer
				var stderr bytes.Buffer
				assertEqualsExitCode(t, cmd.ShowMemo(&stdout, &stderr, tc.args), tc.exitCode)
				assertEqualsMessage(t, stderr.String(), tc.expected)
			})
		}
	})

}

func TestSearch(t *testing.T) {

	cmd := MemoCommandImpl{
		MemoPath:        filepath.Join(t.TempDir(), "memos.json"),
		TimeProvider:    &fakeTimeProvider{},
		StorageOperator: &realMemoOperator,
	}

	memos := []Memo{
		{
			ID:        1,
			Title:     "Jack",
			Body:      "Blue",
			CreatedAt: time.Date(2026, 1, 1, 9, 0, 0, 0, time.Local),
			UpdatedAt: time.Date(2026, 1, 2, 10, 15, 30, 45, time.Local),
		},
		{
			ID:        2,
			Title:     "Nick",
			Body:      "Red",
			CreatedAt: time.Date(2026, 2, 1, 9, 0, 0, 0, time.Local),
			UpdatedAt: time.Date(2026, 2, 2, 10, 15, 30, 45, time.Local),
		},
		{
			ID:        3,
			Title:     "Red",
			Body:      "Green",
			CreatedAt: time.Date(2026, 3, 1, 9, 0, 0, 0, time.Local),
			UpdatedAt: time.Date(2026, 3, 2, 10, 15, 30, 45, time.Local),
		},
	}

	if err := realMemoOperator.SaveMemos(cmd.MemoPath, memos); err != nil {
		t.Fatalf("SaveMemos() err = %v", err)
	}

	t.Run("引数チェック", func(t *testing.T) {
		testCases := []struct {
			testName string
			args     []string
			expected string
			exitCode int
		}{
			{
				testName: "パラメーターが0個の場合",
				args:     []string{},
				expected: "" +
					"Usage:\n" +
					"  memo search <keyword>\n",
				exitCode: 1,
			},
			{
				testName: "パラメーターが2個の場合",
				args:     []string{"aaa", "bbb"},
				expected: "" +
					"引数が多すぎます\n" +
					"Usage:\n" +
					"  memo search <keyword>\n",
				exitCode: 1,
			},
			{
				testName: "キーワードが空文字の場合",
				args:     []string{""},
				expected: "キーワードを入力してください\n",
				exitCode: 1,
			},
			{
				testName: "不正なオプションを渡した場合",
				args:     []string{"--unknown"},
				expected: "" +
					"flag provided but not defined: -unknown\n" +
					"Usage:\n" +
					"  memo search <keyword>\n",
				exitCode: 1,
			},
			{
				testName: "help",
				args:     []string{"--help"},
				expected: "" +
					"Usage:\n" +
					"  memo search <keyword>\n",
				exitCode: 0,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.testName, func(t *testing.T) {
				var stdout bytes.Buffer
				var stderr bytes.Buffer
				assertEqualsExitCode(t, cmd.SearchMemos(&stdout, &stderr, tc.args), tc.exitCode)
				assertEqualsMessage(t, stderr.String(), tc.expected)
			})
		}
	})

	t.Run("キーワードに該当するメモが存在する場合", func(t *testing.T) {

		testCases := []struct {
			testName string
			args     []string
			expected string
		}{
			{
				testName: "keyword = Red",
				args:     []string{"Red"},
				expected: "" +
					"2 Nick\t2026-02-01\n" +
					"3 Red\t2026-03-01\n",
			},
			{
				testName: "keyword = Jack",
				args:     []string{"Jack"},
				expected: "" +
					"1 Jack\t2026-01-01\n",
			},
			{
				testName: "keyword = e",
				args:     []string{"e"},
				expected: "" +
					"1 Jack\t2026-01-01\n" +
					"2 Nick\t2026-02-01\n" +
					"3 Red\t2026-03-01\n",
			},
		}
		for _, tc := range testCases {
			t.Run(tc.testName, func(t *testing.T) {
				var stdout bytes.Buffer
				var stderr bytes.Buffer
				assertEqualsExitCode(t, cmd.SearchMemos(&stdout, &stderr, tc.args), 0)
				assertEqualsMessage(t, stdout.String(), tc.expected)
			})
		}
	})

	t.Run("キーワードに該当するメモが存在しない場合", func(t *testing.T) {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		assertEqualsExitCode(t, cmd.SearchMemos(&stdout, &stderr, []string{"Yellow"}), 1)
		assertEqualsMessage(t, stderr.String(), "No matching memos found.\n")
	})
}

func TestAddShowSearch(t *testing.T) {
	t.Run("AddMemo で登録したメモを ShowMemo や SearchMemos で表示される", func(t *testing.T) {
		filePath := filepath.Join(t.TempDir(), "memos.json")

		cmd := MemoCommandImpl{
			MemoPath:        filePath,
			TimeProvider:    &fakeTimeProvider{},
			StorageOperator: &storage.StorageOperatorImpl{},
		}

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		assertEqualsExitCode(t, cmd.AddMemo(&stdout, &stderr, []string{"title", "--body", "body"}), 0)

		stdout.Reset()
		stderr.Reset()
		assertEqualsExitCode(t, cmd.ShowMemo(&stdout, &stderr, []string{"1"}), 0)
		assertEqualsMessage(
			t,
			stdout.String(),
			""+
				"# title\n"+
				"\n"+
				"ID: 1\n"+
				"Created: 2026-07-14 10:00\n"+
				"Updated: 2026-07-14 10:00\n"+
				"\n"+
				"body\n",
		)

		stdout.Reset()
		stderr.Reset()
		assertEqualsExitCode(t, cmd.SearchMemos(&stdout, &stderr, []string{"title"}), 0)
		assertEqualsMessage(
			t,
			stdout.String(),
			"1 title\t2026-07-14\n",
		)

		stdout.Reset()
		stderr.Reset()
		assertEqualsExitCode(t, cmd.SearchMemos(&stdout, &stderr, []string{"body"}), 0)
		assertEqualsMessage(
			t,
			stdout.String(),
			"1 title\t2026-07-14\n",
		)
	})
}

func TestDeleteMemo(t *testing.T) {

	t.Run("指定IDのメモが存在する場合", func(t *testing.T) {
		filePath := filepath.Join(t.TempDir(), "memos.json")

		memos := []Memo{
			{ID: 1, Title: "Other"},
			{ID: 2, Title: "Target"},
			{ID: 3, Title: "Other"},
		}

		if err := realMemoOperator.SaveMemos(filePath, memos); err != nil {
			t.Fatalf("SaveMemos() err = %v", err)
		}

		var stdout bytes.Buffer
		var stderr bytes.Buffer

		cmd := MemoCommandImpl{
			MemoPath:        filePath,
			TimeProvider:    &fakeTimeProvider{},
			StorageOperator: &realMemoOperator,
		}

		assertEqualsExitCode(t, cmd.DeleteMemo(&stdout, &stderr, []string{"2"}), 0)

		actual, err := realMemoOperator.LoadMemos(filePath)
		if err != nil {
			t.Fatalf("LoadMemos() err = %v", err)
		}

		if len(actual) != 2 {
			t.Fatalf("actual.len = %d, expected = 2", len(actual))
		}

		if actual[0].ID != 1 {
			t.Errorf("actual[0].ID = %d, expected = 1", actual[0].ID)
		}

		if actual[1].ID != 3 {
			t.Errorf("actual[1].ID = %d, expected = 3", actual[1].ID)
		}
	})

	t.Run("指定IDのメモが存在しない場合", func(t *testing.T) {

		filePath := filepath.Join(t.TempDir(), "memos.json")

		memos := []Memo{
			{ID: 1, Title: "Other"},
			{ID: 3, Title: "Other"},
		}

		if err := realMemoOperator.SaveMemos(filePath, memos); err != nil {
			t.Fatalf("SaveMemos() err = %v", err)
		}

		var stdout bytes.Buffer
		var stderr bytes.Buffer

		cmd := MemoCommandImpl{
			MemoPath:        filePath,
			TimeProvider:    &fakeTimeProvider{},
			StorageOperator: &realMemoOperator,
		}

		exitCode := cmd.DeleteMemo(&stdout, &stderr, []string{"2"})
		assertEqualsExitCode(t, exitCode, 1)

		expected := "memo not found: 2\n"
		assertEqualsMessage(t, stderr.String(), expected)

		actual, err := realMemoOperator.LoadMemos(filePath)
		if err != nil {
			t.Fatalf("LoadMemos() err = %v", err)
		}

		if len(actual) != 2 {
			t.Fatalf("actual.len = %d, expected = 2", len(actual))
		}

	})

	t.Run("引数チェック", func(t *testing.T) {

		testCases := []struct {
			testName string
			args     []string
			expected string
			exitCode int
		}{
			{
				testName: "パラメーターが0個の場合",
				args:     []string{},
				expected: "" +
					"Usage:\n" +
					"  memo delete <id>\n",
				exitCode: 1,
			},
			{
				testName: "パラメーターが2個の場合",
				args:     []string{"1", "aaa"},
				expected: "" +
					"引数が多すぎます\n" +
					"Usage:\n" +
					"  memo delete <id>\n",
				exitCode: 1,
			},
			{
				testName: "パラメーターが数値でない場合",
				args:     []string{"a"},
				expected: "idは数値を入力してください: a\n",
				exitCode: 1,
			},
			{
				testName: "不正なオプションを渡した場合",
				args:     []string{"--unknown"},
				expected: "" +
					"flag provided but not defined: -unknown\n" +
					"Usage:\n" +
					"  memo delete <id>\n",
				exitCode: 1,
			},
			{
				testName: "help",
				args:     []string{"--help"},
				expected: "" +
					"Usage:\n" +
					"  memo delete <id>\n",
				exitCode: 0,
			},
			{
				testName: "idが0の場合",
				args:     []string{"0"},
				expected: "idは1以上で指定してください\n",
				exitCode: 1,
			},
			{
				testName: "idが-1の場合",
				args:     []string{"--", "-1"},
				expected: "idは1以上で指定してください\n",
				exitCode: 1,
			},
		}

		cmd := MemoCommandImpl{
			MemoPath:        filepath.Join(t.TempDir(), "/memos.json"),
			TimeProvider:    &fakeTimeProvider{},
			StorageOperator: &realMemoOperator,
		}

		memos := []Memo{
			{ID: 1, Title: "title1"},
			{ID: 2, Title: "title2"},
			{ID: 3, Title: "title3"},
		}

		for _, tc := range testCases {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			if err := realMemoOperator.SaveMemos(cmd.MemoPath, memos); err != nil {
				t.Fatalf("Error SaveMemos(): %v ", err)
			}

			before, err := os.ReadFile(cmd.MemoPath)
			if err != nil {
				t.Fatalf("Error ReadFile(): %v", err)
			}

			exitCode := cmd.DeleteMemo(&stdout, &stderr, tc.args)

			assertEqualsExitCode(t, exitCode, tc.exitCode)
			assertEqualsMessage(t, stderr.String(), tc.expected)

			after, err := os.ReadFile(cmd.MemoPath)
			if err != nil {
				t.Fatalf("Error ReadFile(): %v", err)
			}

			if !bytes.Equal(before, after) {
				t.Errorf("after = %v, before = %v", after, before)
			}

			afterMemos, err := realMemoOperator.LoadMemos(cmd.MemoPath)
			if err != nil {
				t.Fatalf("Error LoadMemos(): %v", err)
			}

			if len(afterMemos) != len(memos) {
				t.Fatalf(
					"afterMemos.len = %d, expected = %d",
					len(afterMemos),
					len(memos),
				)
			}

			for idx, memo := range memos {
				afterMemo := afterMemos[idx]
				if afterMemo != memo {
					t.Errorf(
						"afterMemos[%d] = %v, memos[%d] = %v",
						idx,
						afterMemo,
						idx,
						memo,
					)
				}

			}
		}
	})

}

type fakeStorageOperator struct {
	countCalledSave int
	countCalledLoad int
	isFailedSave    bool
	isFailedLoad    bool
}

func (fs *fakeStorageOperator) SaveMemos(path string, memos []Memo) error {
	fs.countCalledSave += 1
	if fs.isFailedSave {
		return fmt.Errorf("Failed SaveMemos()")
	}
	return realMemoOperator.SaveMemos(path, memos)
}

func (fs *fakeStorageOperator) LoadMemos(path string) ([]Memo, error) {
	fs.countCalledLoad += 1
	if fs.isFailedLoad {
		return []Memo{}, fmt.Errorf("Failed LoadMemos()")
	}
	return realMemoOperator.LoadMemos(path)
}

func TestEditMemo(t *testing.T) {
	beforeMemos := []Memo{
		{
			ID:        1,
			Title:     "Before Title 1",
			Body:      "Before Body 1",
			CreatedAt: date(2026, 1, 1, 9, 15),
			UpdatedAt: date(2026, 1, 1, 9, 15),
		},
		{
			ID:        2,
			Title:     "Before Title 2",
			Body:      "Before Body 2",
			CreatedAt: date(2026, 1, 2, 9, 15),
			UpdatedAt: date(2026, 1, 2, 9, 15),
		},
	}
	usage := "Usage:\n" +
		"  memo edit <id> [--title <title>] [--body <body>]\n" +
		"  -body string\n" +
		"    \tメモの本文\n" +
		"  -title string\n" +
		"    \tメモのタイトル\n"

	t.Run("バリデーションチェック", func(t *testing.T) {
		testCases := []struct {
			testName        string
			args            []string
			isFailedSave    bool
			isFailedLoad    bool
			expected        string
			exitCode        int
			countCalledSave int
			countCalledLoad int
		}{
			{
				testName:        "helpオプションがある",
				args:            []string{"--help"},
				isFailedSave:    false,
				isFailedLoad:    false,
				expected:        usage,
				exitCode:        0,
				countCalledSave: 0,
				countCalledLoad: 0,
			},
			{
				testName:        "helpオプションがある (短縮形)",
				args:            []string{"-h"},
				isFailedSave:    false,
				isFailedLoad:    false,
				expected:        usage,
				exitCode:        0,
				countCalledSave: 0,
				countCalledLoad: 0,
			},
			{
				testName:        "引数の指定がない",
				args:            []string{},
				isFailedSave:    false,
				isFailedLoad:    false,
				expected:        usage,
				exitCode:        1,
				countCalledSave: 0,
				countCalledLoad: 0,
			},
			{
				testName: "IDなしでオプションを指定",
				args:     []string{"--body", "x", "--"},
				expected: usage,
				exitCode: 1,
			},
			{
				testName: "余分な位置引数",
				args:     []string{"1", "--body", "x", "extra"},
				expected: "引数が多すぎます\n" + usage,
				exitCode: 1,
			},
			{
				testName: "未知のオプション",
				args:     []string{"1", "--unknown"},
				expected: "flag provided but not defined: -unknown\n" + usage,
				exitCode: 1,
			},
			{
				testName: "オプション値がない",
				args:     []string{"1", "--body"},
				expected: "flag needs an argument: -body\n" + usage,
				exitCode: 1,
			},
			{
				testName: "オプションの後に区切りなしでIDを指定",
				args:     []string{"--body", "x", "1"},
				expected: "位置引数はオプションより前に指定してください\n" + usage,
				exitCode: 1,
			},
			{
				testName:        "titleとbodyの両方のオプションがない",
				args:            []string{"1"},
				isFailedSave:    false,
				isFailedLoad:    false,
				expected:        "-titleまたは-bodyを指定してください\n" + usage,
				exitCode:        1,
				countCalledSave: 0,
				countCalledLoad: 0,
			},
			{
				testName:     "titleが空文字",
				args:         []string{"1", "--title", ""},
				isFailedSave: false,
				isFailedLoad: false,
				expected: "" +
					"タイトルを入力してください\n" +
					"",
				exitCode:        1,
				countCalledSave: 0,
				countCalledLoad: 0,
			},
			{
				testName:     "titleが空白のみ",
				args:         []string{"1", "--title", "  "},
				isFailedSave: false,
				isFailedLoad: false,
				expected: "" +
					"タイトルを入力してください\n" +
					"",
				exitCode:        1,
				countCalledSave: 0,
				countCalledLoad: 0,
			},
			{
				testName:     "titleが空白のみで有効なbodyがある",
				args:         []string{"1", "--title", "  ", "--body", "available body"},
				isFailedSave: false,
				isFailedLoad: false,
				expected: "" +
					"タイトルを入力してください\n" +
					"",
				exitCode:        1,
				countCalledSave: 0,
				countCalledLoad: 0,
			},
			{
				testName:     "idが数字以外",
				args:         []string{"a", "--title", "aaaa"},
				isFailedSave: false,
				isFailedLoad: false,
				expected: "" +
					"idは数値を入力してください: a\n" +
					"",
				exitCode:        1,
				countCalledSave: 0,
				countCalledLoad: 0,
			},
			{
				testName:     "idが0",
				args:         []string{"0", "--title", "aaaa"},
				isFailedSave: false,
				isFailedLoad: false,
				expected: "" +
					"idは1以上で指定してください\n" +
					"",
				exitCode:        1,
				countCalledSave: 0,
				countCalledLoad: 0,
			},
			{
				testName:     "idがマイナス",
				args:         []string{"--title", "aaaa", "--", "-1"},
				isFailedSave: false,
				isFailedLoad: false,
				expected: "" +
					"idは1以上で指定してください\n" +
					"",
				exitCode:        1,
				countCalledSave: 0,
				countCalledLoad: 0,
			},
			{
				testName:     "存在しないidを指定",
				args:         []string{"99999", "--title", "aaaa"},
				isFailedSave: false,
				isFailedLoad: false,
				expected: "" +
					"memo not found: 99999\n" +
					"",
				exitCode:        1,
				countCalledSave: 0,
				countCalledLoad: 1,
			},
			{
				testName:     "SaveMemosが失敗する",
				args:         []string{"1", "--title", "aaaa"},
				isFailedSave: true,
				isFailedLoad: false,
				expected: "" +
					"メモの保存に失敗しました Failed SaveMemos()\n" +
					"",
				exitCode:        1,
				countCalledSave: 1,
				countCalledLoad: 1,
			},
			{
				testName:     "LoadMemosが失敗する",
				args:         []string{"1", "--title", "aaaa"},
				isFailedSave: false,
				isFailedLoad: true,
				expected: "" +
					"メモを開くことができませんでした Failed LoadMemos()\n" +
					"",
				exitCode:        1,
				countCalledSave: 0,
				countCalledLoad: 1,
			},
		}

		for _, tc := range testCases {
			fakeMemoOperator := fakeStorageOperator{
				countCalledSave: 0,
				countCalledLoad: 0,
				isFailedSave:    tc.isFailedSave,
				isFailedLoad:    tc.isFailedLoad,
			}

			cmd := MemoCommandImpl{
				MemoPath:        filepath.Join(t.TempDir(), "memos.json"),
				TimeProvider:    &fakeTimeProvider{},
				StorageOperator: &fakeMemoOperator,
			}
			t.Run(tc.testName, func(t *testing.T) {
				if err := realMemoOperator.SaveMemos(cmd.MemoPath, beforeMemos); err != nil {
					t.Fatalf("Failed SaveMemos(): %v", err)
				}
				before, err := os.ReadFile(cmd.MemoPath)
				if err != nil {
					t.Fatalf("Error ReadFile(): %v", err)
				}

				var stdout bytes.Buffer
				var stderr bytes.Buffer
				assertEqualsExitCode(t, cmd.EditMemo(&stdout, &stderr, tc.args), tc.exitCode)
				assertEqualsMessage(t, stderr.String(), tc.expected)
				assertEqualsMessage(t, stdout.String(), "")

				if fakeMemoOperator.countCalledSave != tc.countCalledSave {
					t.Errorf("actual called SaveMemos() = %d, expected = %d", fakeMemoOperator.countCalledSave, tc.countCalledSave)
				}

				if fakeMemoOperator.countCalledLoad != tc.countCalledLoad {
					t.Errorf("actual called LoadMemos() = %d, expected = %d", fakeMemoOperator.countCalledLoad, tc.countCalledLoad)
				}

				after, err := os.ReadFile(cmd.MemoPath)
				if err != nil {
					t.Fatalf("Error ReadFile(): %v", err)
				}

				if !bytes.Equal(before, after) {
					t.Errorf("after = %v, before = %v", after, before)
				}

				afterMemos, err := realMemoOperator.LoadMemos(cmd.MemoPath)
				if err != nil {
					t.Fatalf("Error LoadMemos(): %v", err)
				}

				if len(afterMemos) != len(beforeMemos) {
					t.Fatalf(
						"afterMemos.len = %d, expected = %d",
						len(afterMemos),
						len(beforeMemos),
					)
				}

				for idx, beforeMemo := range beforeMemos {
					afterMemo := afterMemos[idx]
					assertEqualsMemo(t, idx, afterMemo, beforeMemo)
				}
			})
		}
	})

	t.Run("空の一覧で対象IDがない", func(t *testing.T) {
		filePath := filepath.Join(t.TempDir(), "memos.json")
		operator := &fakeStorageOperator{}
		cmd := MemoCommandImpl{
			MemoPath:        filePath,
			TimeProvider:    &fakeTimeProvider{},
			StorageOperator: operator,
		}

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		assertEqualsExitCode(t, cmd.EditMemo(&stdout, &stderr, []string{"1", "--body", "x"}), 1)
		assertEqualsMessage(t, stdout.String(), "")
		assertEqualsMessage(t, stderr.String(), "memo not found: 1\n")
		if operator.countCalledLoad != 1 || operator.countCalledSave != 0 {
			t.Errorf("LoadMemos() = %d, SaveMemos() = %d, expected 1 and 0", operator.countCalledLoad, operator.countCalledSave)
		}
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			t.Errorf("memo file should not be created, os.Stat() error = %v", err)
		}
	})

	t.Run("メモを編集する", func(t *testing.T) {

		testCases := []struct {
			testName      string
			args          []string
			expectedTitle string
			expectedBody  string
		}{
			{
				testName:      "titleを更新",
				args:          []string{"1", "--title", "Updated Title"},
				expectedTitle: "Updated Title",
				expectedBody:  "Before Body 1",
			},
			{
				testName:      "bodyを更新",
				args:          []string{"1", "--body", "Updated Body"},
				expectedTitle: "Before Title 1",
				expectedBody:  "Updated Body",
			},
			{
				testName:      "bodyを空文字で更新",
				args:          []string{"1", "--body", ""},
				expectedTitle: "Before Title 1",
				expectedBody:  "",
			},
			{
				testName:      "titleとbodyを更新",
				args:          []string{"1", "--title", "Updated Title", "--body", "Updated Body"},
				expectedTitle: "Updated Title",
				expectedBody:  "Updated Body",
			},
			{
				testName:      "同じ値でも更新日時を変更",
				args:          []string{"1", "--title", "Before Title 1", "--body", "Before Body 1"},
				expectedTitle: "Before Title 1",
				expectedBody:  "Before Body 1",
			},
			{
				testName:      "区切りの後にIDを指定",
				args:          []string{"--body", "x", "--", "1"},
				expectedTitle: "Before Title 1",
				expectedBody:  "x",
			},
			{
				testName:      "等号形式で本文を--に変更",
				args:          []string{"1", "--body=--"},
				expectedTitle: "Before Title 1",
				expectedBody:  "--",
			},
			{
				testName:      "空白だけの本文を保存",
				args:          []string{"1", "--body", "  "},
				expectedTitle: "Before Title 1",
				expectedBody:  "  ",
			},
			{
				testName:      "改行を含む本文を保存",
				args:          []string{"1", "--body", "line 1\nline 2"},
				expectedTitle: "Before Title 1",
				expectedBody:  "line 1\nline 2",
			},
			{
				testName:      "タイトルの前後の空白を維持",
				args:          []string{"1", "--title", "  Updated Title  "},
				expectedTitle: "  Updated Title  ",
				expectedBody:  "Before Body 1",
			},
		}

		cmd := MemoCommandImpl{
			MemoPath:        filepath.Join(t.TempDir(), "memos.json"),
			TimeProvider:    &fakeTimeProvider{},
			StorageOperator: &realMemoOperator,
		}

		for _, tc := range testCases {
			t.Run(tc.testName, func(t *testing.T) {
				if err := realMemoOperator.SaveMemos(cmd.MemoPath, beforeMemos); err != nil {
					t.Fatalf("Error SaveMemos(): %v", err)
				}

				var stdout bytes.Buffer
				var stderr bytes.Buffer

				exitCode := cmd.EditMemo(&stdout, &stderr, tc.args)

				assertEqualsExitCode(t, exitCode, 0)
				assertEqualsMessage(t, stderr.String(), "")
				assertEqualsMessage(t, stdout.String(), "")

				memos, err := cmd.StorageOperator.LoadMemos(cmd.MemoPath)
				if err != nil {
					t.Fatalf("Failed LoadMemos(): %v", err)
				}

				memo := memos[0]
				if memo.ID != 1 {
					t.Errorf("actual ID = %d, expected ID = %d", memo.ID, 1)
				}

				if memo.Title != tc.expectedTitle {
					t.Errorf("actual Title = %q, expected Title = %q", memo.Title, tc.expectedTitle)
				}

				if memo.Body != tc.expectedBody {
					t.Errorf("actual Body = %q, expected Body = %q", memo.Body, tc.expectedBody)
				}

				if !memo.CreatedAt.Equal(date(2026, 1, 1, 9, 15)) {
					t.Errorf(
						"actual CreatedAt = %q, expected CratedAt = %q",
						memo.CreatedAt.String(),
						date(2026, 1, 1, 9, 15).String(),
					)
				}

				if !memo.UpdatedAt.Equal(time.Date(2026, 7, 14, 10, 0, 0, 0, time.Local)) {

					t.Errorf(
						"actual UpdatedAt = %q, expected UpdatedAt = %q",
						memo.UpdatedAt.String(),
						time.Date(2026, 7, 14, 10, 0, 0, 0, time.Local).String(),
					)
				}

				assertEqualsMemo(t, 1, memos[1], beforeMemos[1])

			})
		}
	})
}

func TestAppEditPersistsAcrossInstances(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "memos.json")
	createdAt := date(2026, 9, 1, 10, 15)
	editedAt := date(2026, 9, 2, 12, 30)

	run := func(app *App, args []string, wantCode int, wantStdout, wantStderr string) {
		t.Helper()
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		if code := app.Run(&stdout, &stderr, args); code != wantCode {
			t.Errorf("App.Run(%q) exitCode = %d, expected %d", args, code, wantCode)
		}
		if stdout.String() != wantStdout || stderr.String() != wantStderr {
			t.Errorf("App.Run(%q) stdout = %q, stderr = %q; expected %q, %q",
				args, stdout.String(), stderr.String(), wantStdout, wantStderr)
		}
	}

	addApp := App{Command: &MemoCommandImpl{
		MemoPath:        filePath,
		TimeProvider:    &fixedTimeProvider{now: createdAt},
		StorageOperator: &storage.StorageOperatorImpl{},
	}}
	run(&addApp, []string{"add", "Original Title", "--body", "legacy-unique"}, 0, "", "")
	run(&addApp, []string{"add", "Other Title", "--body", "unrelated"}, 0, "", "")

	editApp := App{Command: &MemoCommandImpl{
		MemoPath:        filePath,
		TimeProvider:    &fixedTimeProvider{now: editedAt},
		StorageOperator: &storage.StorageOperatorImpl{},
	}}
	run(&editApp, []string{"edit", "1", "--title", "Revised Title", "--body", "updated-unique"}, 0, "", "")

	readApp := App{Command: &MemoCommandImpl{
		MemoPath:        filePath,
		StorageOperator: &storage.StorageOperatorImpl{},
	}}
	run(&readApp, []string{"show", "1"}, 0,
		"# Revised Title\n\nID: 1\nCreated: 2026-09-01 10:15\nUpdated: 2026-09-02 12:30\n\nupdated-unique\n", "")
	run(&readApp, []string{"show", "2"}, 0,
		"# Other Title\n\nID: 2\nCreated: 2026-09-01 10:15\nUpdated: 2026-09-01 10:15\n\nunrelated\n", "")
	run(&readApp, []string{"search", "updated-unique"}, 0, "1 Revised Title\t2026-09-01\n", "")
	run(&readApp, []string{"search", "Revised Title"}, 0, "1 Revised Title\t2026-09-01\n", "")
	run(&readApp, []string{"search", "legacy-unique"}, 1, "", "No matching memos found.\n")

	stored, err := (&storage.StorageOperatorImpl{}).LoadMemos(filePath)
	if err != nil {
		t.Fatalf("LoadMemos() err = %v", err)
	}
	want := []Memo{
		{ID: 1, Title: "Revised Title", Body: "updated-unique", CreatedAt: createdAt, UpdatedAt: editedAt},
		{ID: 2, Title: "Other Title", Body: "unrelated", CreatedAt: createdAt, UpdatedAt: createdAt},
	}
	if len(stored) != len(want) {
		t.Fatalf("stored memo count = %d, expected %d", len(stored), len(want))
	}
	for i, memo := range stored {
		expected := want[i]
		if memo.ID != expected.ID || memo.Title != expected.Title || memo.Body != expected.Body ||
			!memo.CreatedAt.Equal(expected.CreatedAt) || !memo.UpdatedAt.Equal(expected.UpdatedAt) {
			t.Errorf("stored[%d] = %+v, expected %+v", i, memo, expected)
		}
	}
}

func assertEqualsExitCode(t *testing.T, actual int, expected int) {
	t.Helper()
	if actual != expected {
		t.Fatalf("actual ExitCode = %d, expected = %d", actual, expected)
	}
}

func assertEqualsMessage(t *testing.T, actual string, expected string) {
	t.Helper()
	if actual != expected {
		t.Errorf("actual = %q, expected = %q", actual, expected)
	}
}

func date(year int, month time.Month, day int, hour int, min int) time.Time {
	return time.Date(year, month, day, hour, min, 0, 0, time.Local)
}

func assertEqualsMemo(t *testing.T, index int, actual Memo, expected Memo) {
	t.Helper()

	if actual.ID != expected.ID {
		t.Errorf(
			"actual[%d].ID = %d, expected[%d].ID = %d",
			index, actual.ID,
			index, expected.ID,
		)
	}

	if actual.Title != expected.Title {
		t.Errorf(
			"actual[%d].Title = %q, expected[%d].Title = %q",
			index, actual.Title,
			index, expected.Title,
		)
	}

	if actual.Body != expected.Body {
		t.Errorf(
			"actual[%d].Body = %q, expected[%d].Body = %q",
			index, actual.Body,
			index, expected.Body,
		)
	}

	if !actual.CreatedAt.Equal(expected.CreatedAt) {
		t.Errorf(
			"actual[%d].CreatedAt = %q, expected[%d].CreatedAt = %q",
			index, actual.CreatedAt.String(),
			index, expected.CreatedAt.String(),
		)
	}

	if !actual.UpdatedAt.Equal(expected.UpdatedAt) {
		t.Errorf(
			"actual[%d].UpdatedAt = %q, expected[%d].UpdatedAt = %q",
			index, actual.UpdatedAt.String(),
			index, expected.UpdatedAt.String(),
		)
	}
}
