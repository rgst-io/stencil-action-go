package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"path/filepath"

	"github.com/jaredallard/archives"
	"github.com/jaredallard/cmdexec"
	"github.com/jaredallard/vcs"
	"github.com/jaredallard/vcs/releases"
	"github.com/jaredallard/vcs/resolver"
	"github.com/jaredallard/vcs/token"
	"github.com/sethvargo/go-githubactions"
)

var (
	repo    = "rgst-io/stencil"
	repoURL = "https://github.com/" + repo
)

func main() {
	ctx := context.Background()

	tok := githubactions.GetInput("github-token")
	if tok == "" {
		st, err := token.Fetch(ctx, vcs.ProviderGithub, false)
		if err != nil {
			githubactions.Fatalf("failed to get github token: %v", err)
		}

		tok = st.Value
	}

	version := githubactions.GetInput("version")
	if version == "" {
		version = "latest"
	}

	binaryDir := githubactions.GetInput("binary-dir")
	if binaryDir == "" {
		var err error
		if binaryDir, err = os.UserCacheDir(); err != nil {
			githubactions.Fatalf("failed to get cache dir: %v", err)
		}
	}

	if version == "latest" {
		v, err := resolver.NewResolver().Resolve(ctx, repoURL, &resolver.Criteria{
			Constraint: "*",
		})
		if err != nil {
			githubactions.Fatalf("failed to resolve latest: %v", err)
		}

		githubactions.Infof("Resolved latest -> %s", version)
		version = v.Tag
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		githubactions.Fatalf("failed to get user's home directory: %v", err)
	}

	extractPath := filepath.Join(
		strings.ReplaceAll(binaryDir, "~", homeDir),
		"stencil", version, "stencil",
	)
	extractDir := filepath.Dir(extractPath)

	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		githubactions.Fatalf("failed to create extract path dirs %q: %v", extractDir, err)
	}

	githubactions.Infof("Downloading stencil %s -> %s", version, extractPath)
	af, finf, err := releases.Fetch(ctx, &releases.FetchOptions{
		RepoURL:   repoURL,
		AssetName: fmt.Sprintf("stencil_*_%s_%s.tar.*", runtime.GOOS, runtime.GOARCH),
		Tag:       version,
	})
	if err != nil {
		githubactions.Fatalf("failed to download release: %v", err)
	}

	f, err := os.Create(extractPath)
	if err != nil {
		githubactions.Fatalf("failed to create %q: %v", extractPath, err)
	}
	defer f.Close()

	a, err := archives.Open(af, archives.OpenOptions{Extension: archives.Ext(finf.Name())})
	if err != nil {
		githubactions.Fatalf("failed to open %q as an archive: %v", finf.Name(), err)
	}
	defer a.Close()

	r, err := archives.Pick(a, archives.PickFilterByName("stencil"))
	if err != nil {
		githubactions.Fatalf("failed to find stencil in %q", finf.Name())
	}

	if _, err := io.Copy(f, r); err != nil {
		githubactions.Fatalf("failed to extract stencil: %v", err)
	}

	if err := f.Chmod(0o755); err != nil {
		githubactions.Fatalf("failed to chmod +x stencil: %v", err)
	}

	githubactions.Infof("Validating binary attestation ...")
	cmd := cmdexec.Command("gh", "attestation", "verify", extractPath, "--repo", repo)
	cmd.SetEnviron([]string{"GH_TOKEN=" + tok})
	cmd.UseOSStreams(true)
	if err := cmd.Run(); err != nil {
		githubactions.Fatalf("attestation validation failed: %v", err)
	}

	githubactions.AddPath(extractDir)
	if err := cmdexec.CommandContext(ctx, "stencil", "--version").Run(); err != nil {
		githubactions.Fatalf("failed to run stencil --version as a test: %v", err)
	}
}
