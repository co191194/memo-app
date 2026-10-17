package cli

import (
	"fmt"
	"io"
)

func PrintHelp(writer io.Writer) {
	fmt.Fprintln(writer, "Usage:")
	fmt.Fprintln(writer, "  memo <command> [arguments]")
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "Commands:")
	fmt.Fprintln(writer, "  add     Add a new memo")
	fmt.Fprintln(writer, "  edit    Edit a memo")
	fmt.Fprintln(writer, "  list    List memos")
	fmt.Fprintln(writer, "  show    Show a memo")
	fmt.Fprintln(writer, "  search  Search memos")
	fmt.Fprintln(writer, "  delete  delete a memo")
	fmt.Fprintln(writer, "  serve   Start the web server")
}
