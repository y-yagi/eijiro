package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/chzyer/readline"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/profile"
	"github.com/y-yagi/configure"
	"github.com/y-yagi/debuglog"
	"github.com/y-yagi/eijiro"
)

const cmd = "eijiro"

type config struct {
	DataBase  string `toml:"database"`
	SelectCmd string `toml:"selectcmd"`
}

var cfg config
var dlogger *debuglog.Logger

func init() {
	err := configure.Load(cmd, &cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(cfg.DataBase) == 0 {
		cfg.DataBase = filepath.Join(configure.ConfigDir(cmd), cmd+".db")
		if err := configure.Save(cmd, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save config: %v\n", err)
		}
	}
	dlogger = debuglog.New(os.Stderr, debuglog.Flag(log.LstdFlags))
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
	var profileFlg bool
	var dbconsoleFlg bool
	var err error

	exitCode = 0

	flags := flag.NewFlagSet(cmd, flag.ExitOnError)
	flags.SetOutput(errStream)
	flags.StringVar(&importFile, "import", "", "Import `file`.")
	flags.BoolVar(&config, "c", false, "Edit config.")
	flags.BoolVar(&interactive, "i", false, "Use interactive mode.")
	flags.BoolVar(&interactive, "interactive", false, "Use interactive mode.")
	flags.BoolVar(&profileFlg, "profile", false, "Enable profile.")
	flags.BoolVar(&dbconsoleFlg, "db", false, "Run db console.")
	flags.Parse(args[1:])

	if profileFlg {
		defer profile.Start().Stop()
	}

	if config {
		if err = runConfigure(); err != nil {
			fmt.Fprintf(errStream, "Error: %v\n", err)
			exitCode = 1
		}
		return
	}

	if len(importFile) != 0 {
		if err = runImport(importFile); err != nil {
			fmt.Fprintf(errStream, "Error: %v\n", err)
			exitCode = 1
		}
		return
	}

	if dbconsoleFlg {
		if err = runDBConsole(); err != nil {
			fmt.Fprintf(errStream, "Error: %v\n", err)
			exitCode = 1
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
		searchText = strings.TrimSpace(flags.Args()[0])
		if len(searchText) == 0 {
			exitCode = 2
			usage(flags)
			return
		}
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

	ej := eijiro.NewEijiro(cfg.DataBase)
	err = ej.Init()
	if err != nil {
		fmt.Fprintf(errStream, "Error: %v\n", err)
		exitCode = 1
		return
	}
	defer ej.Terminate()

	for {
		if interactive {
			searchText, err = l.Readline()
			if err != nil {
				return
			}
			searchText = strings.TrimSpace(searchText)
			if len(searchText) == 0 {
				continue
			}
		}

		dlogger.Print("Start Select")
		texts, err := ej.Select(searchText)
		if err != nil {
			fmt.Fprintf(errStream, "Error: %v\n", err)
			exitCode = 1
			return
		}
		dlogger.Print("End Select")

		var sb strings.Builder
		for _, text := range texts {
			sb.WriteString(text)
			sb.WriteString("\n")
		}
		buf := sb.String()

		if len(cfg.SelectCmd) == 0 {
			fmt.Fprintf(outStream, "%v\n", buf)
		} else {
			if err := runSelectCmd(strings.NewReader(buf), outStream, errStream); err != nil {
				fmt.Fprintf(errStream, "Error: %v\n", err)
				if !interactive {
					exitCode = 1
				}
			}
		}

		if !interactive {
			return
		}
	}
}

func runConfigure() error {
	editor := os.Getenv("EDITOR")
	if len(editor) == 0 {
		editor = "vim"
	}

	return configure.Edit(cmd, editor)
}

func runImport(file string) error {
	ej := eijiro.NewEijiro(cfg.DataBase)
	err := ej.Import(file)
	ej.Terminate()
	return err
}

func runSelectCmd(r io.Reader, out, err io.Writer) error {
	cmd := exec.Command("sh", "-c", cfg.SelectCmd)

	cmd.Stderr = err
	cmd.Stdout = out
	cmd.Stdin = r

	return cmd.Run()
}

func runDBConsole() error {
	cmd := exec.Command("sqlite3", cfg.DataBase)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
