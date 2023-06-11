package panicwatch_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"testing"

	goerrors "github.com/go-errors/errors"
	"github.com/grongor/panicwatch"
	"github.com/stretchr/testify/require"
)

const panicRegexTemplate = `goroutine 1 \[running\]:\r?\n` +
	`main\.executeCommand\(.*?\)\r?\n` +
	`\t%[1]s/cmd/test/test\.go:\d+ \+0x[a-z0-9]+\r?\n` +
	`main.main\(\)\r?\n` +
	`\t%[1]s/cmd/test/test\.go:\d+ \+0x[a-z0-9]+\r?\n`

func TestPanicwatch(t *testing.T) {
	builder := strings.Builder{}

	for i := 0; i < 1500; i++ {
		builder.WriteString("some garbage here...")
		builder.WriteString("\n")
	}

	garbageString := builder.String()

	panicRegex := getPanicRegex()

	tests := []struct {
		command          string
		expectedStdout   string
		expectedStderr   string
		expectedPanic    string
		expectedExitCode int
	}{
		{
			command:        "no-panic",
			expectedStdout: "some stdout output\n",
			expectedStderr: "some stderr output\n",
		},
		{
			command:          "no-panic-error",
			expectedExitCode: 1,
			expectedStderr:   "blah blah something happened\n",
		},
		{
			command:          "panic",
			expectedExitCode: 2,
			expectedStdout:   "some output...\neverything looks good...\n",
			expectedPanic:    "wtf, unexpected panic!",
		},
		{
			command:          "panic-and-error",
			expectedExitCode: 2,
			expectedStdout:   "some output...\neverything looks good...\n",
			expectedStderr:   "well something goes bad ...\n",
			expectedPanic:    "... and panic!",
		},
		{
			command:          "panic-sync-split",
			expectedExitCode: 2,
			expectedPanic:    "i'm split in three lol",
		},
		{
			command:          "panic-with-garbage",
			expectedExitCode: 2,
			expectedStdout:   garbageString,
			expectedStderr:   "panic: blah blah\n\n" + garbageString,
			expectedPanic:    "and BAM!",
		},
		{
			command:          "only-last-panic-string-is-detected",
			expectedExitCode: 2,
			expectedStderr:   "panic: this is fake\n\n",
			expectedPanic:    "and this is not",
		},
	}
	for _, test := range tests {
		t.Run(test.command, func(t *testing.T) {
			assert := require.New(t)

			cmd, stdout, stderr, resultFile := helperProcess(test.command)
			defer os.Remove(resultFile)

			err := cmd.Run()

			if test.expectedExitCode == 0 {
				assert.NoError(err, "unexpected exit code, stderr: "+stderr.String())
			} else {
				assert.Error(err)

				var exitErr *exec.ExitError

				assert.True(errors.As(err, &exitErr))
				assert.Equal(
					test.expectedExitCode,
					exitErr.ExitCode(),
					"unexpected exit code, stderr: "+stderr.String(),
				)
			}

			assert.Equal(test.expectedStdout, stdout.String())

			result := readResult(resultFile)

			assert.Equal(test.expectedPanic, result.Message)

			if test.expectedPanic == "" {
				assert.Equal(test.expectedStderr, stderr.String())

				return
			}

			assert.Regexp(panicRegex, result.Stack)

			if test.expectedStderr != "" {
				assert.True(strings.HasPrefix(stderr.String(), test.expectedStderr))
			}

			assert.Regexp(
				fmt.Sprintf("panic: %s\n\n%s", test.expectedPanic, panicRegex),
				stderr.String(),
			)

			var resultAsErr *goerrors.Error

			assert.True(errors.As(result.AsError(), &resultAsErr))

			assert.Equal(test.expectedPanic, resultAsErr.Error())

			builder := strings.Builder{}

			builder.WriteString("goroutine 1 [running]:\n")

			for _, frame := range resultAsErr.StackFrames() {
				if frame.Name == "main" {
					builder.WriteString(frame.Package + `.` + frame.Name + `()` + "\n")
				} else {
					builder.WriteString(frame.Package + `.` + frame.Name + `(0x0, 0x0)` + "\n")
				}

				builder.WriteString("\t" + frame.File + ":" + strconv.Itoa(frame.LineNumber) + ` +0x0` + "\n")
			}

			assert.Regexp(panicRegex, builder.String())
		})
	}
}

// Each test uses this test method to run a separate process in order to test the functionality.
func helperProcess(command string) (*exec.Cmd, *bytes.Buffer, *bytes.Buffer, string) {
	f, err := ioutil.TempFile("", "result")
	if err != nil {
		panic(err)
	}

	err = f.Close()
	if err != nil {
		panic(err)
	}

	cmd := exec.Command("./test", command, f.Name()) //nolint:gosec // we control the inputs
	cmd.Stderr = new(bytes.Buffer)
	cmd.Stdout = new(bytes.Buffer)

	return cmd, cmd.Stdout.(*bytes.Buffer), cmd.Stderr.(*bytes.Buffer), f.Name()
}

func getPanicRegex() string {
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Dir(filename)

	return fmt.Sprintf(panicRegexTemplate, dir)
}

func readResult(resultFile string) panicwatch.Panic {
	resultBytes, err := ioutil.ReadFile(resultFile)
	if err != nil {
		panic(err)
	}

	if len(resultBytes) == 0 {
		return panicwatch.Panic{}
	}

	result := panicwatch.Panic{}

	err = json.Unmarshal(resultBytes, &result)
	if err != nil {
		panic(err)
	}

	return result
}
