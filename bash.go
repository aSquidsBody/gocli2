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

type BashResult struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Err    error  `json:"err"`
}

// Run a bash command and return the stdout & stderr in a
// BashResult struct
func Bash(cmd string) (res BashResult) {
	res.Stdout, res.Stderr, res.Err = runBash(cmd, false, false, "")
	return
}

// Run a bash command, stream the stdout and/or stderr, and
// return the stdout & stderr in a BashResult stuct
func BashStream(cmd string, stdout bool, stderr bool) (res BashResult) {
	res.Stdout, res.Stderr, res.Err = runBash(cmd, stdout, stderr, "")
	return
}

// Run a bash command, stream the stdout and/or stderr with a custom label,
// and return the stdout & stderr in a BashResult struct
func BashStreamLabel(cmd string, stdout bool, stderr bool, label string) (res BashResult) {
	res.Stdout, res.Stderr, res.Err = runBash(cmd, stdout, stderr, label)
	return
}

// Run a bash command with special options.
//
// "cmd" is the bash command, "sOut" indicates whether to stream the stdout,
// "sErr" indicates whether to stream the stderr, and "l" is the label of
// any stream
func runBash(cmd string, sOut bool, sErr bool, l string) (stdout string, stderr string, err error) {
	c := exec.Command(fmt.Sprintf(`bash`), "-c", "-e", cmd)

	outPipe, err := c.StdoutPipe()
	if err != nil {
		return
	}

	errPipe, err := c.StderrPipe()
	if err != nil {
		return
	}

	c.Start()

	errs := make(chan error)

	go func() {
		var err error
		stdout, err = readShell(bufio.NewReader(outPipe), sOut, l)
		if err != nil {
			errs <- err
		}
	}()
	go func() {
		var err error
		stderr, err = readShell(bufio.NewReader(errPipe), sErr, l)
		if err != nil {
			errs <- err
		}
	}()

	// wait for the above goroutines to complete
	err = c.Wait()

	// pick up any errors
	select {
	case err = <-errs:
	default:
	}

	return
}

// go routine to parse the output of a shell
func readShell(r *bufio.Reader, stream bool, label string) (output string, err error) {
	strs := make(chan string)
	errs := make(chan error)
	done := make(chan int)

	go func() {
		defer func() { done <- 0 }()
		for {
			line, _, err := r.ReadLine()

			if err != nil {
				if err != io.EOF {
					errs <- err
				}
				break
			}
			strs <- string(line)
		}
	}()

	v := true
	for v {
		select {

		case err = <-errs:
			v = false
		case <-done:
			v = false
		case s := <-strs:
			if stream {
				fmt.Printf("%s%s%s", label, s, Sep())
			}
			output = fmt.Sprintf("%s%s\n", output, s)
		default:
		}
	}
	return
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
