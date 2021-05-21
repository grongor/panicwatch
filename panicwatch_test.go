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
	"strings"
	"testing"

	"github.com/grongor/panicwatch"
	"github.com/stretchr/testify/require"
)

const panicRegexTemplate = `goroutine 1 \[running\]:
main\.executeCommand\(0x[a-z0-9]+, 0x[a-z0-9]+\)
\s+%[1]s/cmd/test/test\.go:\d+ \+0x[a-z0-9]+
main.main\(\)
\s+%[1]s/cmd/test/test\.go:\d+ \+0x[a-z0-9]+
`

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

			result := readResult(resultFile)

			assert.Equal(test.expectedPanic, result.Message)

			if test.expectedPanic != "" {
				assert.Regexp(panicRegex, result.Stack)

				if test.expectedStderr != "" {
					assert.True(strings.HasPrefix(stderr.String(), test.expectedStderr))
				}

				assert.Regexp(
					fmt.Sprintf("panic: %s\n\n%s", test.expectedPanic, panicRegex),
					stderr.String(),
				)
			} else {
				assert.Equal(test.expectedStderr, stderr.String())
			}

			assert.Equal(test.expectedStdout, stdout.String())
		})
	}
}

// Each test uses this test method to run a separate process in order to test the functionality.
func helperProcess(command string) (*exec.Cmd, *bytes.Buffer, *bytes.Buffer, string) {
	f, err := ioutil.TempFile("", "result")
	if err != nil {
		panic(err)
	}

	f.Close()

	cmd := exec.Command("./test", command, f.Name())
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
