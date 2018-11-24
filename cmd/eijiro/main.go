package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/chzyer/readline"
	_ "github.com/mattn/go-sqlite3"
	"github.com/y-yagi/configure"
	"github.com/y-yagi/eijiro"
)

const cmd = "eijiro"

type config struct {
	DataBase  string `toml:"database"`
	SelectCmd string `tomo:"selectcmd"`
}

var cfg config

func init() {
	err := configure.Load(cmd, &cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(cfg.DataBase) == 0 {
		cfg.DataBase = filepath.Join(configure.ConfigDir(cmd), cmd+".db")
		configure.Save(cmd, cfg)
	}
}

func main() {
	os.Exit(run(os.Args, os.Stdout, os.Stderr))
}

func usage(flags *flag.FlagSet) {
	fmt.Printf("Usage: %s [OPTIONS] text\n", cmd)
	flags.PrintDefaults()
}

func run(args []string, outStream, errStream io.Writer) (exitCode int) {
	var importFile string
	var config bool
	var interactive bool

	exitCode = 0

	flags := flag.NewFlagSet(cmd, flag.ExitOnError)
	flags.SetOutput(errStream)
	flags.StringVar(&importFile, "import", "", "Import `file`.")
	flags.BoolVar(&config, "c", false, "Edit config.")
	flags.BoolVar(&interactive, "i", false, "Use interactive mode.")
	flags.BoolVar(&interactive, "interactive", false, "Use interactive mode.")
	flags.Parse(args[1:])

	if config {
		editor := os.Getenv("EDITOR")
		if len(editor) == 0 {
			editor = "vim"
		}

		if err := configure.Edit(cmd, editor); err != nil {
			fmt.Fprintf(errStream, "Error: %v\n", err)
			exitCode = 1
			return
		}
		return
	}

	ej := eijiro.NewEijiro(cfg.DataBase)
	err := ej.InitDB()
	if err != nil {
		fmt.Fprintf(errStream, "Error: %v\n", err)
		exitCode = 1
		return
	}

	if len(importFile) != 0 {
		if err = ej.Import(importFile); err != nil {
			fmt.Fprintf(errStream, "Error: %v\n", err)
			exitCode = 1
			return
		}
		return
	}

	var searchText string
	var l *readline.Instance

	if !interactive {
		if len(flags.Args()) != 1 {
			exitCode = 2
			usage(flags)
			return
		}
		searchText = flags.Args()[0]
	} else {
		l, err = readline.NewEx(&readline.Config{
			Prompt:          "Eiji: ",
			InterruptPrompt: "^C",
			Stdout:          outStream,
		})

		if err != nil {
			fmt.Fprintf(errStream, "Error: %v\n", err)
			exitCode = 1
			return
		}
		defer l.Close()
	}

	for {
		if interactive {
			searchText, err = l.Readline()
			if err != nil {
				return
			}
			if len(searchText) == 0 {
				continue
			}
		}

		documents, err := ej.Select(searchText)
		if err != nil {
			fmt.Fprintf(errStream, "Error: %v\n", err)
			exitCode = 1
			return
		}

		var buf string
		for _, document := range documents {
			buf += fmt.Sprintf("%s\n", document.Text)
		}

		if len(cfg.SelectCmd) == 0 {
			fmt.Fprintf(outStream, "%v\n", buf)
		} else {
			runSelectCmd(strings.NewReader(buf), outStream, errStream)
		}

		if !interactive {
			return
		}
	}
	return
}

func runSelectCmd(r io.Reader, out, err io.Writer) error {
	cmd := exec.Command("sh", "-c", cfg.SelectCmd)

	cmd.Stderr = err
	cmd.Stdout = out
	cmd.Stdin = r

	return cmd.Run()
}
