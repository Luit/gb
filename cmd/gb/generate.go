package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
)

func init() {
	registerCommand(GenerateCmd)
}

var (
	generateN bool
	generateX bool
)

func addGenerateFlags(fs *flag.FlagSet) {
	fs.BoolVar(&generateN, "n", false, "print commands that would be executed")
	fs.BoolVar(&generateX, "x", false, "print commands as they are executed")
}

var GenerateCmd = &cmd.Command{
	Name:      "generate",
	UsageLine: "generate",
	Short:     "generate Go files by processing source",
	Long: `Generate runs commands described by directives within existing files.
Those commands can run any process but the intent is to create or update Go
source files, for instance by running yacc.

See 'go help generate'`,
	Run: func(ctx *gb.Context, args []string) error {
		bin, err := exec.LookPath("go")
		if err != nil {
			return err
		}
		env := cmd.MergeEnv(os.Environ(), map[string]string{
			"GOPATH": fmt.Sprintf("%s:%s", ctx.Projectdir(), filepath.Join(ctx.Projectdir(), "vendor")),
		})
		if generateN {
			args = append([]string{"-n"}, args...)
		}
		if generateX {
			args = append([]string{"-x"}, args...)
		}
		if gb.Verbose {
			args = append([]string{"-v"}, args...)
		}
		args = append([]string{bin, "generate"}, args...)
		cmd := exec.Cmd{
			Path: args[0],
			Args: args,
			Env:  env,

			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}

		return cmd.Run()
	},
	AddFlags: addGenerateFlags,
}
