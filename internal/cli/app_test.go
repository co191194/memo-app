package cli_test

import (
	"bytes"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/co191194/memo-app/internal/cli"
)

type fakeMemoCommand struct {
	MemoPath     string
	TimeProvider cli.TimeProvider
	Command      cli.MemoCommand
	editArgs     []string
	editCalls    int
	editExitCode int
}

type App = cli.App

func (cmd *fakeMemoCommand) AddMemo(stdout io.Writer, stderr io.Writer, args []string) int {
	fmt.Fprint(stdout, "Called AddMemo()")

	if len(args) != 1 || args[0] == "add" {
		fmt.Fprint(stderr, "Failed AddMemo()")
		return 1
	}

	return 0
}
func (cmd *fakeMemoCommand) ListMemos(stdout io.Writer, stderr io.Writer, args []string) int {
	fmt.Fprint(stdout, "Called ListMemos()")

	if len(args) != 0 {
		fmt.Fprint(stderr, "Failed ListMemos()")
		return 1
	}

	return 0
}
func (cmd *fakeMemoCommand) ShowMemo(stdout io.Writer, stderr io.Writer, args []string) int {
	fmt.Fprint(stdout, "Called ShowMemo()")

	if len(args) != 1 || args[0] == "show" {
		fmt.Fprint(stderr, "Failed ShowMemo()")
		return 1
	}

	return 0
}
func (cmd *fakeMemoCommand) SearchMemos(stdout io.Writer, stderr io.Writer, args []string) int {
	fmt.Fprint(stdout, "Called SearchMemos()")

	if len(args) != 1 || args[0] == "search" {
		fmt.Fprint(stderr, "Failed SearchMemos()")
		return 1
	}
	return 0
}
func (cmd *fakeMemoCommand) DeleteMemo(stdout io.Writer, stderr io.Writer, args []string) int {
	fmt.Fprint(stdout, "Called DeleteMemo()")

	if len(args) != 1 || args[0] == "delete" {
		fmt.Fprint(stderr, "Failed DeleteMemo()")
		return 1
	}
	return 0
}

func (cmd *fakeMemoCommand) EditMemo(stdout io.Writer, stderr io.Writer, args []string) int {
	cmd.editCalls++
	cmd.editArgs = append([]string(nil), args...)
	if cmd.editExitCode != 0 {
		fmt.Fprint(stderr, "Failed EditMemo()")
		return cmd.editExitCode
	}
	fmt.Fprint(stdout, "Called EditMemo()")
	return 0
}

func TestRun_EditMemo(t *testing.T) {
	args := []string{"1", "--title", "Updated Title", "--body", "Updated Body"}
	command := &fakeMemoCommand{}
	app := App{Command: command}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := app.Run(&stdout, &stderr, append([]string{"edit"}, args...))

	if exitCode != 0 {
		t.Errorf("exitCode = %d, expected 0", exitCode)
	}
	if command.editCalls != 1 || !slices.Equal(command.editArgs, args) {
		t.Errorf("EditMemo() calls = %d, args = %q, expected 1 call with %q", command.editCalls, command.editArgs, args)
	}
	if stdout.String() != "Called EditMemo()" || stderr.String() != "" {
		t.Errorf("stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}
}

func TestRun_EditMemoFailure(t *testing.T) {
	command := &fakeMemoCommand{editExitCode: 1}
	app := App{Command: command}
	args := []string{"1", "--title", ""}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := app.Run(&stdout, &stderr, append([]string{"edit"}, args...))

	if exitCode != 1 {
		t.Errorf("exitCode = %d, expected 1", exitCode)
	}
	if command.editCalls != 1 || !slices.Equal(command.editArgs, args) {
		t.Errorf("EditMemo() calls = %d, args = %q, expected 1 call with %q", command.editCalls, command.editArgs, args)
	}
	if stdout.String() != "" || stderr.String() != "Failed EditMemo()" {
		t.Errorf("stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}
}

func TestRun(t *testing.T) {
	app := App{
		Command: &fakeMemoCommand{},
	}

	testCases := []struct {
		testName string
		args     []string
		expected string
	}{
		{"AddMemo()を呼ぶ", []string{"add", "test"}, "Called AddMemo()"},
		{"ListMemos()を呼ぶ", []string{"list"}, "Called ListMemos()"},
		{"ShowMemo()を呼ぶ", []string{"show", "1"}, "Called ShowMemo()"},
		{"SearchMemos()を呼ぶ", []string{"search", "Jack"}, "Called SearchMemos()"},
		{"DeleteMemo()を呼ぶ", []string{"delete", "3"}, "Called DeleteMemo()"},
	}

	for _, tc := range testCases {

		t.Run(tc.testName, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			if exitCode := app.Run(&stdout, &stderr, tc.args); exitCode != 0 {
				t.Fatalf("actual exitCode = %d, expected = 0", exitCode)
			}

			if stdout.String() != tc.expected {
				t.Errorf("actual = %q, expected = %q", stdout.String(), tc.expected)
			}
		})
	}

}

func TestRun_PrintHelp(t *testing.T) {
	app := App{
		Command: &fakeMemoCommand{},
	}

	testCases := []struct {
		testName string
		args     []string
	}{
		{"コマンドが未入力の場合", []string{}},
		{"未定義のコマンドの場合", []string{"unknown"}},
	}

	var sb strings.Builder
	sb.WriteString("Usage:\n")
	sb.WriteString("  memo <command> [arguments]\n")
	sb.WriteString("\n")
	sb.WriteString("Commands:\n")
	sb.WriteString("  add     Add a new memo\n")
	sb.WriteString("  edit    Edit a memo\n")
	sb.WriteString("  list    List memos\n")
	sb.WriteString("  show    Show a memo\n")
	sb.WriteString("  search  Search memos\n")
	sb.WriteString("  delete  delete a memo\n")

	expected := sb.String()

	for _, tc := range testCases {

		t.Run(tc.testName, func(t *testing.T) {

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			if exitCode := app.Run(&stdout, &stderr, tc.args); exitCode != 1 {
				t.Fatalf("actual exitCode = %d, expected = 1", exitCode)
			}

			if stderr.String() != expected {
				t.Errorf("actual = %q, expected = %q", stderr.String(), expected)
			}
			if stdout.String() != "" {
				t.Errorf("stdout = %q, expected empty", stdout.String())
			}
		})
	}
}

func TestHelp(t *testing.T) {

	testCases := []struct {
		testName string
		args     []string
	}{
		{
			testName: "help",
			args:     []string{"help"},
		},
		{
			testName: "--help",
			args:     []string{"--help"},
		},
		{
			testName: "-h",
			args:     []string{"-h"},
		},
	}

	app := App{
		Command: &fakeMemoCommand{},
	}

	expected := "" +
		"Usage:\n" +
		"  memo <command> [arguments]\n" +
		"\n" +
		"Commands:\n" +
		"  add     Add a new memo\n" +
		"  edit    Edit a memo\n" +
		"  list    List memos\n" +
		"  show    Show a memo\n" +
		"  search  Search memos\n" +
		"  delete  delete a memo\n"

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {

			var stdout bytes.Buffer
			var stderr bytes.Buffer

			if exitCode := app.Run(&stdout, &stderr, tc.args); exitCode != 0 {
				t.Fatalf("actual exitCode = %d, expected = 0", exitCode)
			}

			if stdout.String() != expected {
				t.Errorf("actual = %q, expected = %q", stdout.String(), expected)
			}
			if stderr.String() != "" {
				t.Errorf("stderr = %q, expected empty", stderr.String())
			}

		})
	}
}

func TestRun_EditMemoHelp(t *testing.T) {
	app := App{Command: &MemoCommandImpl{}}
	usage := "Usage:\n" +
		"  memo edit <id> [--title <title>] [--body <body>]\n" +
		"  -body string\n" +
		"    \tメモの本文\n" +
		"  -title string\n" +
		"    \tメモのタイトル\n"

	for _, option := range []string{"--help", "-h"} {
		t.Run(option, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			exitCode := app.Run(&stdout, &stderr, []string{"edit", option})

			if exitCode != 0 {
				t.Errorf("exitCode = %d, expected 0", exitCode)
			}
			if stdout.String() != "" || stderr.String() != usage {
				t.Errorf("stdout = %q, stderr = %q, expected stderr = %q", stdout.String(), stderr.String(), usage)
			}
		})
	}
}

func TestRun_FailedCommand(t *testing.T) {

	app := App{
		Command: &fakeMemoCommand{},
	}

	testCases := []struct {
		testName string
		args     []string
		expected string
	}{
		{"AddMemo()が失敗", []string{"add", "test", "test"}, "Failed AddMemo()"},
		{"ListMemos()が失敗", []string{"list", "aaaa"}, "Failed ListMemos()"},
		{"ShowMemo()が失敗", []string{"show"}, "Failed ShowMemo()"},
		{"SearchMemos()が失敗", []string{"search", "Jack", "Joe"}, "Failed SearchMemos()"},
		{"DeleteMemo()が失敗", []string{"delete", "3", "2"}, "Failed DeleteMemo()"},
	}

	for _, tc := range testCases {

		t.Run(tc.testName, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			if exitCode := app.Run(&stdout, &stderr, tc.args); exitCode != 1 {
				t.Fatalf("actual exitCode = %d, expected = 1", exitCode)
			}

			if stderr.String() != tc.expected {
				t.Errorf("actual = %q, expected = %q", stderr.String(), tc.expected)
			}
		})
	}
}
