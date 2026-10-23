# Changelog

## Unreleased

- Stop blocked browser calls promptly and keep replacement sessions and Unix sockets intact during restart. Thanks @SebTardif.
- Move macOS CI to Xcode 27 runners and verify release binaries on both Apple Silicon and Intel under Rosetta.
- Use Go 1.27.3 for CI and releases to fix reachable GO-2026-6809 in certificate verification; retain the Go 1.26 source minimum. New builds require macOS 13 or later. Add safe release dry runs through the updated shared workflow.

## 0.1.0 - 2026-09-13

**Highlights:** DOM commands now accept `--mode`, and `dom type --clear` correctly clears inputs and textareas.

- Fixed `dom type --clear` writing `undefined` instead of clearing inputs and textareas with current browser dependencies.
- Fixed documented `dom` subcommands rejecting the `--mode` flag.
- Updated the preferred/release Go toolchain to 1.26.8 and CI to 1.27.1, retaining Go 1.26 source compatibility and macOS 12 release support.
- Updated browser automation, file watching, Go 1.26 toolchain, and GitHub Actions dependencies.
