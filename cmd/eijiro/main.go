package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/y-yagi/configure"
	"github.com/y-yagi/eijiro"
)

const cmd = "eijiro"

type config struct {
	DataBase string `toml:"database"`
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

func run(args []string, outStream, errStream io.Writer) (exitCode int) {
	var importFile string
	var config bool

	exitCode = 0

	flags := flag.NewFlagSet(cmd, flag.ExitOnError)
	flags.SetOutput(errStream)
	flags.StringVar(&importFile, "i", "", "Import `file`.")
	flags.BoolVar(&config, "c", false, "Edit config.")
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

	if len(flags.Args()) != 1 {
		exitCode = 2
		usage(errStream)
		return
	}

	documents, err := ej.Select(flags.Args()[0])
	if err != nil {
		fmt.Fprintf(errStream, "Error: %v\n", err)
		exitCode = 1
		return
	}

	for _, document := range documents {
		fmt.Fprintf(outStream, "%s\n", document.Text)
	}

	return
}

func usage(errStream io.Writer) {
	fmt.Fprintf(errStream, "Usage: %s [OPTIONS] text\n", cmd)
}
