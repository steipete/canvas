# Releasing canvas

Canvas uses the GoReleaser-based shared release pipeline also used by
`steipete/eightctl`. Releases contain `canvas_<version>_<os>_<arch>.tar.gz`
archives for macOS and Linux on arm64 and amd64, with `README.md` and `LICENSE`,
plus `checksums.txt` and verification manifests. The macOS binaries are signed
with Peter Steinberger’s Developer ID and notarized. Release builds use the
Go 1.26.8 toolchain in `go.mod` to retain macOS 12 support.

## Repository setup

- Protect `main` and require the `Test`, `Lint`, and `Linux` CI checks.
- Allow GitHub Actions to create pull requests for the next `Unreleased` section.
- Configure per-repository secrets: `MACOS_SIGNING_P12`,
  `MACOS_SIGNING_P12_PASSWORD`, `ASC_KEY_ID`, `ASC_ISSUER_ID`, and
  `ASC_PRIVATE_KEY_P8`. Use the personal Developer ID certificate.
- Keep authorized SSH tag-signing public keys in `.github/release-allowed-signers`.
- Homebrew updates use a maintainer login with access to
  `steipete/homebrew-tap`. An `openclaw/homebrew-tap` token cannot update this tap.

## Prepare and publish

1. Start from clean, synchronized `main`. Check local and remote tags and
   releases; stop if the target version already exists.
2. On a task branch, finalize every `Unreleased` entry under a dated version
   heading. Put highlights first. Review the release change and land it via PR.
3. Run the complete local gate before landing:

   ```sh
   test -z "$(gofmt -l .)"
   go mod tidy && git diff --exit-code -- go.mod go.sum
   go build ./...
   go test ./...
   go test -race ./...
   go test -race -tags=integration ./internal/browser
   golangci-lint run --timeout=5m
   actionlint
   goreleaser check
   GOTOOLCHAIN=go1.26.8 go test ./...
   GOTOOLCHAIN=go1.26.8 goreleaser build --snapshot --clean
   ```

4. Wait for all CI checks on the exact merged `main` commit to pass. Recheck
   tags and releases immediately before creating the SSH-signed annotated tag:

   ```sh
   git tag -s v0.1.0 -m "canvas 0.1.0"
   git -c gpg.ssh.allowedSignersFile=.github/release-allowed-signers verify-tag v0.1.0
   git push origin v0.1.0
   gh workflow run release.yml --ref main -f version=0.1.0
   ```

   Substitute the requested version for subsequent releases. Tag pushes do not
   publish by themselves. The manual workflow requires the signed tag, verifies
   its frozen commit and CI, builds all four targets, signs and notarizes macOS
   binaries, and independently verifies both macOS architectures before publishing.
   Release notes come directly from the tagged, finalized changelog section.

5. Watch the release run to completion. Never move a tag. If a run fails, inspect
   the failed stage before retrying the same version; an already-published release
   should be verified instead of republished.
6. Download release assets and verify `checksums.txt`, the binary version, macOS
   signing/notarization, and the Mach-O minimum OS (12.0). Verify the Go module:

   ```sh
   GOPROXY=https://proxy.golang.org go list -m github.com/steipete/canvas@v0.1.0
   ```

## Homebrew and closeout

After the GitHub release is published, run `./scripts/update-homebrew v0.1.0`.
This follows `steipete/gifgrep`’s dispatch handoff to the personal tap’s existing
`update-formula.yml` workflow. The tap downloads all four archives, computes
checksums, and creates or updates `Formula/canvas.rb`. Watch that exact tap run,
compare its URLs and hashes with the release, then install or upgrade
`steipete/tap/canvas` and verify `canvas --version`.

The shared release workflow opens a PR adding the next `Unreleased` section.
Review and merge that PR after verifying publication and Homebrew. If automatic
PR creation failed, add the section in a maintainer PR. Leave `main` synchronized
and clean, and remove task branches and local `dist/` build outputs.
