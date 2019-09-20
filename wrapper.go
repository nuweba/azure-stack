package azurestack

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

func ExecCmd(funcDir string, bin string, cmdToExec ...string) (string, string, error) {
	var stdoutBuf, stderrBuf bytes.Buffer
	var errStdout, errStderr error
	var wg sync.WaitGroup
	wg.Add(2)

	cwd := funcDir

	cmd := exec.Command(bin, cmdToExec...)
	cmd.Dir = cwd

	stdoutIn, _ := cmd.StdoutPipe()
	stderrIn, _ := cmd.StderrPipe()

	stdout := io.MultiWriter(os.Stdout, &stdoutBuf)
	stderr := io.MultiWriter(os.Stderr, &stderrBuf)
	err := cmd.Start()

	go func() {
		_, errStdout = io.Copy(stdout, stdoutIn)
		wg.Done()
	}()

	go func() {
		_, errStderr = io.Copy(stderr, stderrIn)
		wg.Done()
	}()

	wg.Wait()
	err = cmd.Wait()
	if errStdout != nil || errStderr != nil {
		return "", "", errors.New("failed to capture stdout or stderr")
	}
	return strings.TrimSpace(stdoutBuf.String()), strings.TrimSpace(stderrBuf.String()), err
}
