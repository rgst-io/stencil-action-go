// Copyright (C) 2026 stencil-action-go contributors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public
// License along with this program. If not, see
// <https://www.gnu.org/licenses/>.
//
// SPDX-License-Identifier: LGPL-3.0

// Package main implements stencil-action-go.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"path/filepath"

	"github.com/sethvargo/go-githubactions"
	"go.rgst.io/jaredallard/archives/v2"
	"go.rgst.io/jaredallard/cmdexec/v2"
	"go.rgst.io/jaredallard/vcs/v2"
	"go.rgst.io/jaredallard/vcs/v2/releases"
	"go.rgst.io/jaredallard/vcs/v2/resolver"
	"go.rgst.io/jaredallard/vcs/v2/token"
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

	if err := os.MkdirAll(extractDir, 0o750); err != nil {
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

	//nolint:gosec // Why: By design.
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

	if _, err := exec.LookPath("gh"); err != nil {
		githubactions.Infof("Validating binary attestation ...")
		cmd := cmdexec.Command("gh", "attestation", "verify", extractPath, "--repo", repo)
		cmd.SetEnviron([]string{"GH_TOKEN=" + tok})
		cmd.UseOSStreams(true)
		if err := cmd.Run(); err != nil {
			githubactions.Fatalf("attestation validation failed: %v", err)
		}
	} else {
		githubactions.Warningf("Skipping binary attestation validation (gh CLI not found)")
	}

	githubactions.AddPath(extractDir)
	if err := cmdexec.CommandContext(ctx, "stencil", "--version").Run(); err != nil {
		githubactions.Fatalf("failed to run stencil --version as a test: %v", err)
	}
}
