package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tsingson/bitproto/compiler-go/native"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	args := os.Args[1:]

	if len(args) == 0 {
		return errors.New("usage: bitproto-go [options] lang file [out]\n       bitproto-go native-check file")
	}

	if args[0] == "native-check" {
		if len(args) != 2 {
			return errors.New("usage: bitproto-go native-check file")
		}
		proto, err := native.ParseFile(args[1])
		if err != nil {
			return err
		}
		fmt.Printf("native parse ok: proto=%s aliases=%d enums=%d messages=%d\n", proto.Name, len(proto.Aliases), len(proto.Enums), len(proto.Messages))
		return nil
	}

	if args[0] == "native-c" {
		if len(args) < 2 || len(args) > 3 {
			return errors.New("usage: bitproto-go native-c file [outdir]")
		}
		outDir := ""
		if len(args) == 3 {
			outDir = args[2]
		}
		proto, err := native.ParseFile(args[1])
		if err != nil {
			return err
		}
		h, c, err := native.RenderC(proto, outDir)
		if err != nil {
			return err
		}
		fmt.Printf("native c generated: %s\n", h)
		fmt.Printf("native c generated: %s\n", c)
		return nil
	}

	if args[0] == "native-go" {
		if len(args) < 2 || len(args) > 3 {
			return errors.New("usage: bitproto-go native-go file [outdir]")
		}
		outDir := ""
		if len(args) == 3 {
			outDir = args[2]
		}
		proto, err := native.ParseFile(args[1])
		if err != nil {
			return err
		}
		p, err := native.RenderGo(proto, outDir)
		if err != nil {
			return err
		}
		fmt.Printf("native go generated: %s\n", p)
		return nil
	}

	if args[0] == "native-py" {
		if len(args) < 2 || len(args) > 3 {
			return errors.New("usage: bitproto-go native-py file [outdir]")
		}
		outDir := ""
		if len(args) == 3 {
			outDir = args[2]
		}
		proto, err := native.ParseFile(args[1])
		if err != nil {
			return err
		}
		p, err := native.RenderPy(proto, outDir)
		if err != nil {
			return err
		}
		fmt.Printf("native py generated: %s\n", p)
		return nil
	}

	backend := os.Getenv("BITPROTO_BACKEND_BIN")
	if backend == "" {
		backend = "bitproto"
	}
	backendArgs := strings.Fields(os.Getenv("BITPROTO_BACKEND_ARGS"))

	self, err := os.Executable()
	if err == nil {
		selfAbs, err1 := filepath.Abs(self)
		backendAbs, err2 := filepath.Abs(backend)
		if err1 == nil && err2 == nil && selfAbs == backendAbs {
			return errors.New("BITPROTO_BACKEND_BIN points to current executable; refusing recursive invocation")
		}
	}

	cmdArgs := append([]string{}, backendArgs...)
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.Command(backend, cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to invoke backend compiler %q: %w", backend, err)
	}
	return nil
}
