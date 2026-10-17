package cli

import "io"

type App struct {
	Command MemoCommand
}

func (app *App) Run(
	stdout io.Writer,
	stderr io.Writer,
	args []string,
) int {
	if len(args) < 1 {
		PrintHelp(stderr)
		return 1
	}

	var exitCode int

	switch args[0] {
	case "add":
		exitCode = app.Command.AddMemo(stdout, stderr, args[1:])
	case "list":
		exitCode = app.Command.ListMemos(stdout, stderr, args[1:])
	case "show":
		exitCode = app.Command.ShowMemo(stdout, stderr, args[1:])
	case "search":
		exitCode = app.Command.SearchMemos(stdout, stderr, args[1:])
	case "delete":
		exitCode = app.Command.DeleteMemo(stdout, stderr, args[1:])
	case "edit":
		exitCode = app.Command.EditMemo(stdout, stderr, args[1:])
	case "serve":
		exitCode = app.Command.Serve(stdout, stderr, args[1:])
	case "help", "-h", "--help":
		PrintHelp(stdout)
		exitCode = 0
	default:
		PrintHelp(stderr)
		return 1
	}

	return exitCode
}
