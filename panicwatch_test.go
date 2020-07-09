package panicwatch_test

import (
	"bytes"
	"encoding/json"
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
main.main\(\)
\s+%scmd/test/test\.go:\d+ \+0x[a-z0-9]+
`

func TestPanicwatch(t *testing.T) {
	builder := strings.Builder{}

	for i := 0; i < 1500; i++ {
		builder.WriteString("some garbage here...")
		builder.WriteString("\n")
	}

	garbageString := builder.String()

	tests := []struct {
		command          string
		expectedExitCode int
		expectedStdout   string
		expectedStderr   string
		expectedPanic    string
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
			expectedPanic:    "i'm split in two lol",
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
			require := require.New(t)

			cmd, stdout, stderr, resultFile := helperProcess(test.command)
			defer os.Remove(resultFile)

			err := cmd.Run()

			if test.expectedExitCode == 0 {
				require.NoError(err, "unexpeced exit code, stderr: "+stderr.String())
			} else {
				require.Error(err)
				require.Equal(
					test.expectedExitCode,
					err.(*exec.ExitError).ExitCode(),
					"unexpected exit code, stderr: "+stderr.String(),
				)
			}

			result := readResult(resultFile)

			require.Equal(test.expectedPanic, result.Message)

			if test.expectedPanic != "" {
				panicRegex := getPanicRegex()

				require.Regexp(panicRegex, result.Stack)

				if test.expectedStderr != "" {
					require.True(strings.HasPrefix(stderr.String(), test.expectedStderr))
				}

				require.Regexp(
					fmt.Sprintf("panic: %s\n\n%s", test.expectedPanic, panicRegex),
					stderr.String(),
				)
			} else {
				require.Equal(test.expectedStderr, stderr.String())
			}

			require.Equal(test.expectedStdout, stdout.String())
		})
	}
}

// each test uses this test method to run a separate process in order to test the functionality
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

	return fmt.Sprintf(panicRegexTemplate, dir+"/")
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
