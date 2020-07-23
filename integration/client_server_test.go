// +build integration

package integration

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	"github.com/aquasecurity/trivy/internal"
)

type args struct {
	Format            string
	TemplatePath      string
	Version           string
	IgnoreUnfixed     bool
	Severity          []string
	IgnoreIDs         []string
	Input             string
	ClientToken       string
	ClientTokenHeader string
}

func TestClientServer(t *testing.T) {
	cases := []struct {
		name     string
		testArgs args
		golden   string
		wantErr  string
	}{
		{
			name: "alpine 3.10 integration",
			testArgs: args{
				Version: "dev",
				Input:   "testdata/fixtures/alpine-310.tar.gz",
			},
			golden: "testdata/alpine-310.json.golden",
		},
		{
			name: "alpine 3.10 integration with --ignore-unfixed option",
			testArgs: args{
				Version:       "dev",
				IgnoreUnfixed: true,
				Input:         "testdata/fixtures/alpine-310.tar.gz",
			},
			golden: "testdata/alpine-310-ignore-unfixed.json.golden",
		},
		{
			name: "alpine 3.10 integration with medium and high severity",
			testArgs: args{
				Version:       "dev",
				IgnoreUnfixed: true,
				Severity:      []string{"MEDIUM", "HIGH"},
				Input:         "testdata/fixtures/alpine-310.tar.gz",
			},
			golden: "testdata/alpine-310-medium-high.json.golden",
		},
		{
			name: "alpine 3.10 integration with .trivyignore",
			testArgs: args{
				Version:       "dev",
				IgnoreUnfixed: false,
				IgnoreIDs:     []string{"CVE-2019-1549", "CVE-2019-1563"},
				Input:         "testdata/fixtures/alpine-310.tar.gz",
			},
			golden: "testdata/alpine-310-ignore-cveids.json.golden",
		},
		{
			name: "alpine 3.10 integration with gitlab template",
			testArgs: args{
				Format:       "template",
				TemplatePath: "@../contrib/gitlab.tpl",
				Version:      "dev",
				Input:        "testdata/fixtures/alpine-310.tar.gz",
			},
			golden: "testdata/alpine-310.gitlab.golden",
		},
		{
			name: "alpine 3.10 integration with sarif template",
			testArgs: args{
				Format:       "template",
				TemplatePath: "@../contrib/sarif.tpl",
				Version:      "dev",
				Input:        "testdata/fixtures/alpine-310.tar.gz",
			},
			golden: "testdata/alpine-310.sarif.golden",
		},
		{
			name: "alpine 3.9 integration",
			testArgs: args{
				Version: "dev",
				Input:   "testdata/fixtures/alpine-39.tar.gz",
			},
			golden: "testdata/alpine-39.json.golden",
		},
		{
			name: "debian buster integration",
			testArgs: args{
				Version: "dev",
				Input:   "testdata/fixtures/debian-buster.tar.gz",
			},
			golden: "testdata/debian-buster.json.golden",
		},
		{
			name: "debian buster integration with --ignore-unfixed option",
			testArgs: args{
				Version:       "dev",
				IgnoreUnfixed: true,
				Input:         "testdata/fixtures/debian-buster.tar.gz",
			},
			golden: "testdata/debian-buster-ignore-unfixed.json.golden",
		},
		{
			name: "debian stretch integration",
			testArgs: args{
				Version: "dev",
				Input:   "testdata/fixtures/debian-stretch.tar.gz",
			},
			golden: "testdata/debian-stretch.json.golden",
		},
		{
			name: "ubuntu 18.04 integration",
			testArgs: args{
				Version: "dev",
				Input:   "testdata/fixtures/ubuntu-1804.tar.gz",
			},
			golden: "testdata/ubuntu-1804.json.golden",
		},
		{
			name: "ubuntu 18.04 integration with --ignore-unfixed option",
			testArgs: args{
				Version:       "dev",
				IgnoreUnfixed: true,
				Input:         "testdata/fixtures/ubuntu-1804.tar.gz",
			},
			golden: "testdata/ubuntu-1804-ignore-unfixed.json.golden",
		},
		{
			name: "ubuntu 16.04 integration",
			testArgs: args{
				Version: "dev",
				Input:   "testdata/fixtures/ubuntu-1604.tar.gz",
			},
			golden: "testdata/ubuntu-1604.json.golden",
		},
		{
			name: "centos 7 integration",
			testArgs: args{
				Version: "dev",
				Input:   "testdata/fixtures/centos-7.tar.gz",
			},
			golden: "testdata/centos-7.json.golden",
		},
		{
			name: "centos 7 integration with --ignore-unfixed option",
			testArgs: args{
				Version:       "dev",
				IgnoreUnfixed: true,
				Input:         "testdata/fixtures/centos-7.tar.gz",
			},
			golden: "testdata/centos-7-ignore-unfixed.json.golden",
		},
		{
			name: "centos 7 integration with low and high severity",
			testArgs: args{
				Version:       "dev",
				IgnoreUnfixed: true,
				Severity:      []string{"LOW", "HIGH"},
				Input:         "testdata/fixtures/centos-7.tar.gz",
			},
			golden: "testdata/centos-7-low-high.json.golden",
		},
		{
			name: "centos 6 integration",
			testArgs: args{
				Version: "dev",
				Input:   "testdata/fixtures/centos-6.tar.gz",
			},
			golden: "testdata/centos-6.json.golden",
		},
		{
			name: "ubi 7 integration",
			testArgs: args{
				Version: "dev",
				Input:   "testdata/fixtures/ubi-7.tar.gz",
			},
			golden: "testdata/ubi-7.json.golden",
		},
		{
			name: "distroless base integration",
			testArgs: args{
				Version: "dev",
				Input:   "testdata/fixtures/distroless-base.tar.gz",
			},
			golden: "testdata/distroless-base.json.golden",
		},
		{
			name: "distroless base integration with --ignore-unfixed option",
			testArgs: args{
				Version:       "dev",
				IgnoreUnfixed: true,
				Input:         "testdata/fixtures/distroless-base.tar.gz",
			},
			golden: "testdata/distroless-base-ignore-unfixed.json.golden",
		},
		{
			name: "distroless python27 integration",
			testArgs: args{
				Version: "dev",
				Input:   "testdata/fixtures/distroless-python27.tar.gz",
			},
			golden: "testdata/distroless-python27.json.golden",
		},
		{
			name: "amazon 1 integration",
			testArgs: args{
				Version: "dev",
				Input:   "testdata/fixtures/amazon-1.tar.gz",
			},
			golden: "testdata/amazon-1.json.golden",
		},
		{
			name: "amazon 2 integration",
			testArgs: args{
				Version: "dev",
				Input:   "testdata/fixtures/amazon-2.tar.gz",
			},
			golden: "testdata/amazon-2.json.golden",
		},
		{
			name: "oracle 6 integration",
			testArgs: args{
				Version: "dev",
				Input:   "testdata/fixtures/oraclelinux-6-slim.tar.gz",
			},
			golden: "testdata/oraclelinux-6-slim.json.golden",
		},
		{
			name: "oracle 7 integration",
			testArgs: args{
				Version: "dev",
				Input:   "testdata/fixtures/oraclelinux-7-slim.tar.gz",
			},
			golden: "testdata/oraclelinux-7-slim.json.golden",
		},
		{
			name: "oracle 8 integration",
			testArgs: args{
				Version: "dev",
				Input:   "testdata/fixtures/oraclelinux-8-slim.tar.gz",
			},
			golden: "testdata/oraclelinux-8-slim.json.golden",
		},
		{
			name: "opensuse leap 15.1 integration",
			testArgs: args{
				Version: "dev",
				Input:   "testdata/fixtures/opensuse-leap-151.tar.gz",
			},
			golden: "testdata/opensuse-leap-151.json.golden",
		},
		{
			name: "opensuse leap 42.3 integration",
			testArgs: args{
				Version: "dev",
				Input:   "testdata/fixtures/opensuse-leap-423.tar.gz",
			},
			golden: "testdata/opensuse-leap-423.json.golden",
		},
		{
			name: "photon 1.0 integration",
			testArgs: args{
				Version: "dev",
				Input:   "testdata/fixtures/photon-10.tar.gz",
			},
			golden: "testdata/photon-10.json.golden",
		},
		{
			name: "photon 2.0 integration",
			testArgs: args{
				Version: "dev",
				Input:   "testdata/fixtures/photon-20.tar.gz",
			},
			golden: "testdata/photon-20.json.golden",
		},
		{
			name: "photon 3.0 integration",
			testArgs: args{
				Version: "dev",
				Input:   "testdata/fixtures/photon-30.tar.gz",
			},
			golden: "testdata/photon-30.json.golden",
		},
		{
			name: "buxybox with Cargo.lock integration",
			testArgs: args{
				Version: "dev",
				Input:   "testdata/fixtures/busybox-with-lockfile.tar.gz",
			},
			golden: "testdata/busybox-with-lockfile.json.golden",
		},
	}

	app, addr, cacheDir := setup(t, "", "")

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			osArgs, outputFile, cleanup := setupClient(t, c.testArgs, addr, cacheDir, c.golden)
			defer cleanup()

			// Run Trivy client
			err := app.Run(osArgs)
			require.NoError(t, err)

			compare(t, c.golden, outputFile)
		})
	}
}

func TestClientServerWithToken(t *testing.T) {
	cases := []struct {
		name     string
		testArgs args
		golden   string
		wantErr  string
	}{
		{
			name: "alpine 3.10 integration with token",
			testArgs: args{
				Version:           "dev",
				Input:             "testdata/fixtures/alpine-310.tar.gz",
				ClientToken:       "token",
				ClientTokenHeader: "Trivy-Token",
			},
			golden: "testdata/alpine-310.json.golden",
		},
		{
			name: "invalid token",
			testArgs: args{
				Version:           "dev",
				Input:             "testdata/fixtures/distroless-base.tar.gz",
				ClientToken:       "invalidtoken",
				ClientTokenHeader: "Trivy-Token",
			},
			wantErr: "twirp error unauthenticated: invalid token",
		},
		{
			name: "invalid token header",
			testArgs: args{
				Version:           "dev",
				Input:             "testdata/fixtures/distroless-base.tar.gz",
				ClientToken:       "valid-token",
				ClientTokenHeader: "Trivy-Token",
			},
			wantErr: "twirp error unauthenticated: invalid token",
		},
	}

	serverToken := "token"
	serverTokenHeader := "Trivy-Token"
	app, addr, cacheDir := setup(t, serverToken, serverTokenHeader)
	defer os.RemoveAll(cacheDir)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			osArgs, outputFile, cleanup := setupClient(t, c.testArgs, addr, cacheDir, c.golden)
			defer cleanup()

			// Run Trivy client
			err := app.Run(osArgs)

			if c.wantErr != "" {
				require.NotNil(t, err, c.name)
				assert.Contains(t, err.Error(), c.wantErr, c.name)
				return
			} else {
				assert.NoError(t, err, c.name)
			}

			compare(t, c.golden, outputFile)
		})
	}
}

func setup(t *testing.T, token, tokenHeader string) (*cli.App, string, string) {
	t.Helper()
	version := "dev"

	// Copy DB file
	cacheDir, err := gunzipDB()
	assert.NoError(t, err)

	port, err := getFreePort()
	assert.NoError(t, err)
	addr := fmt.Sprintf("localhost:%d", port)

	go func() {
		// Setup CLI App
		app := internal.NewApp(version)
		app.Writer = ioutil.Discard
		osArgs := setupServer(addr, token, tokenHeader, cacheDir)

		// Run Trivy server
		app.Run(osArgs)
	}()

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	err = waitPort(ctx, addr)
	assert.NoError(t, err)

	// Setup CLI App
	app := internal.NewApp(version)
	app.Writer = ioutil.Discard

	return app, addr, cacheDir
}

func setupServer(addr, token, tokenHeader, cacheDir string) []string {
	osArgs := []string{"trivy", "server", "--skip-update", "--cache-dir", cacheDir, "--listen", addr}
	if token != "" {
		osArgs = append(osArgs, []string{"--token", token, "--token-header", tokenHeader}...)
	}
	return osArgs
}

func setupClient(t *testing.T, c args, addr string, cacheDir string, golden string) ([]string, string, func()) {
	t.Helper()
	osArgs := []string{"trivy", "client", "--cache-dir", cacheDir, "--remote", "http://" + addr}

	if c.Format != "" {
		osArgs = append(osArgs, "--format", c.Format)
		if c.TemplatePath != "" {
			osArgs = append(osArgs, "--template", c.TemplatePath)
		}
	} else {
		osArgs = append(osArgs, "--format", "json")
	}

	if c.IgnoreUnfixed {
		osArgs = append(osArgs, "--ignore-unfixed")
	}
	if len(c.Severity) != 0 {
		osArgs = append(osArgs,
			[]string{"--severity", strings.Join(c.Severity, ",")}...,
		)
	}

	var err error
	var ignoreTmpDir string
	if len(c.IgnoreIDs) != 0 {
		ignoreTmpDir, err = ioutil.TempDir("", "ignore")
		require.NoError(t, err, "failed to create a temp dir")
		trivyIgnore := filepath.Join(ignoreTmpDir, ".trivyignore")
		err = ioutil.WriteFile(trivyIgnore, []byte(strings.Join(c.IgnoreIDs, "\n")), 0444)
		require.NoError(t, err, "failed to write .trivyignore")
		osArgs = append(osArgs, []string{"--ignorefile", trivyIgnore}...)
	}
	if c.ClientToken != "" {
		osArgs = append(osArgs, []string{"--token", c.ClientToken, "--token-header", c.ClientTokenHeader}...)
	}
	if c.Input != "" {
		osArgs = append(osArgs, []string{"--input", c.Input}...)
	}

	// Setup the output file
	var outputFile string
	if *update {
		outputFile = golden
	} else {
		output, _ := ioutil.TempFile("", "integration")
		assert.Nil(t, output.Close())
		outputFile = output.Name()
	}

	cleanup := func() {
		_ = os.Remove(ignoreTmpDir)
		if !*update {
			_ = os.Remove(outputFile)
		}
	}

	osArgs = append(osArgs, []string{"--output", outputFile}...)
	return osArgs, outputFile, cleanup
}

func compare(t *testing.T, wantFile, gotFile string) {
	t.Helper()
	// Compare want and got
	want, err := ioutil.ReadFile(wantFile)
	assert.NoError(t, err)
	got, err := ioutil.ReadFile(gotFile)
	assert.NoError(t, err)

	assert.JSONEq(t, string(want), string(got))
}
