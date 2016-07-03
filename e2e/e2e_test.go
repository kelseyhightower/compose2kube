package e2e

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
)

func TestSimple(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "test-compose2docker-simple-")
	defer os.RemoveAll(tempdir)
	assert.NoError(t, err, "Temporary directory creation failed")

	createSpecs(t, tempdir)
	installedRcs := installSpecs(t, tempdir)
	defer removeRcs(t, installedRcs)

	// verify rc created
	assert.Equal(t, []string{"cache", "database", "web"}, installedRcs)
}

func createSpecs(t *testing.T, dir string) {
	cmd := exec.Command("compose2kube", "-compose-file", "simple/docker-compose.yml", "-output-dir", dir)
	err := cmd.Start()
	assert.NoError(t, err, "Process start failed")
	err = cmd.Wait()
	assert.NoError(t, err, "Wait failed")
	assert.True(t, cmd.ProcessState.Success(), "Process failed", cmd.ProcessState.String())
}

func installSpecs(t *testing.T, dir string) []string {
	wg := &sync.WaitGroup{}
	cmd := exec.Command("kubectl", "create", "-f", dir)
	stdout, err := cmd.StdoutPipe()
	stderr, err := cmd.StderrPipe()
	assert.NoError(t, err, "Could not instantiate stdout pipe")
	var capturedStdout string

	wg.Add(2)
	go func() {
		buf := &bytes.Buffer{}
		_, err := io.Copy(buf, stdout)
		assert.NoError(t, err, "Stdout copy failed")
		capturedStdout = buf.String()
		wg.Done()
	}()

	go func() {
		buf := &bytes.Buffer{}
		_, err = io.Copy(buf, stderr)
		assert.NoError(t, err, "Stdout copy failed")
		assert.Empty(t, buf.String())
		wg.Done()
	}()

	err = cmd.Start()
	assert.NoError(t, err, "Process start failed")
	err = cmd.Wait()
	assert.NoError(t, err, "Wait failed")
	assert.True(t, cmd.ProcessState.Success(), "Process failed", cmd.ProcessState.String())
	wg.Wait()

	splitStdout := strings.Split(capturedStdout, "\n")
	rgx, _ := regexp.Compile("replicationcontroller \"([a-zA-Z0-9]+)\".*")
	results := make([]string, 0)
	for _, v := range splitStdout {
		if rgx.MatchString(v) {
			groups := rgx.FindStringSubmatch(v)
			results = append(results, groups[1])
		}
	}
	sort.Strings(results)
	return results
}

func removeRcs(t *testing.T, replicationControllers []string) {
	params := []string{"delete", "rc"}
	params = append(params, replicationControllers...)
	cmd := exec.Command("kubectl", params...)

	err := cmd.Start()
	assert.NoError(t, err, "Process start failed")

	err = cmd.Wait()
	assert.NoError(t, err, "Wait failed")
	assert.True(t, cmd.ProcessState.Success(), "Process failed", cmd.ProcessState.String())
}
