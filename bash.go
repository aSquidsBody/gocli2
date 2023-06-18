package gocli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"golang.org/x/crypto/ssh/terminal"
)

type BashProcess struct {
	command       *exec.Cmd
	stdinReader   *stdinReader
	stdoutHandler StdHandler
	stderrHandler StdHandler
	running       bool
}

// StdHandler is a user defined function to handle the contents of cmd.Stdout and
// cmd.Stderr.
type StdHandler func(line []byte) error

// customStdWriter implements io.Writer. It is a custom resource that will take the
// place of os.Stdout/os.Stderr to allow for custom processing of the cmd output.
type customStdWriter struct {
	handler StdHandler
}

func (c *customStdWriter) Write(p []byte) (int, error) {
	if err := c.handler(p); err != nil {
		return 0, err
	}

	return len(p), nil
}

// StdinReader
type stdinReader struct {
	lines chan []byte
}

func (s *stdinReader) Read(p []byte) (int, error) {
	line, ok := <-s.lines
	if !ok {
		return 0, io.EOF
	}
	return copy(p, line), nil

}

func (b *BashProcess) HandleStdout(handler StdHandler) {
	if !b.running {
		b.stdoutHandler = handler
	}
}

func (b *BashProcess) HandleStderr(handler StdHandler) {
	if !b.running {
		b.stderrHandler = handler
	}
}

func (b *BashProcess) CustomStdinWithBufferSize(preload []string, bufferLines int) {
	if b.running {
		return
	}
	if preload == nil {
		preload = []string{}
	}

	b.stdinReader = &stdinReader{}
	b.stdinReader.lines = make(chan []byte, bufferLines)
	for _, line := range preload {
		b.stdinReader.lines <- []byte(line)
	}
}

func (b *BashProcess) CustomStdin(preload []string) {
	b.CustomStdinWithBufferSize(preload, len(preload)+16)
}

func (b *BashProcess) Stdin(line string) {
	if b.running && b.stdinReader != nil && b.stdinReader.lines != nil {
		b.stdinReader.lines <- []byte(line)
	}
}

// ProcessState returns the underlying *os.ProcessState of the cmd object. Will return
// nil if Exec has not been called
func (b *BashProcess) ProcessState() *os.ProcessState {
	if b.command != nil {
		return b.command.ProcessState
	}
	return nil
}

func (b *BashProcess) Process() *os.Process {
	if b.command != nil {
		return b.command.Process
	}
	return nil
}

func (b *BashProcess) Exec(cmd string) error {
	if b.running {
		return fmt.Errorf("Cannot exec command. Already running")
	}

	b.command = exec.Command("bash", "-c", "-e", cmd)

	// if no stdout handler is defined, then default to printing to stdout
	if b.stdoutHandler == nil {
		b.command.Stdout = os.Stdout
	} else {
		b.command.Stdout = &customStdWriter{
			handler: b.stdoutHandler,
		}
	}

	// if no stderr handler is defined, then default to printing to stderr
	if b.stderrHandler == nil {
		b.command.Stderr = os.Stderr
	} else {
		b.command.Stderr = &customStdWriter{
			handler: b.stderrHandler,
		}
	}

	// if no inputs overwrite stdin, then default to reading from stdin
	if b.stdinReader == nil {
		b.command.Stdin = os.Stdin
	} else {
		b.command.Stdin = b.stdinReader

	}
	b.running = true

	// check if process has exited
	go func() {
		for b.command.ProcessState.ExitCode() < 0 {
		}
		b.running = false
		if b.stdinReader.lines != nil {
			close(b.stdinReader.lines)
		}
	}()
	return b.command.Run()
}

func Bash() *BashProcess {
	return &BashProcess{
		stdinReader:   nil,
		stdoutHandler: nil,
		stderrHandler: nil,
		running:       false,
	}
}

func Sep() string {
	if runtime.GOOS == "windows" {
		return "\r\n"
	} else {
		return "\n"
	}
}

func ReadHidden(prompt string) (string, error) {
	fmt.Print(prompt)
	p, err := terminal.ReadPassword(0)
	fmt.Println()
	return string(p), err
}

func ReadInput(prompt string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	line = strings.TrimSuffix(line, "\n")
	return line, err
}
