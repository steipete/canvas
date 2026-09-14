# Changelog

## 0.1.0 - 2026-09-13

**Highlights:** DOM commands now accept `--mode`, and `dom type --clear` correctly clears inputs and textareas.

- Fixed `dom type --clear` writing `undefined` instead of clearing inputs and textareas with current browser dependencies.
- Fixed documented `dom` subcommands rejecting the `--mode` flag.
- Updated the preferred/release Go toolchain to 1.26.8 and CI to 1.27.1, retaining Go 1.26 source compatibility and macOS 12 release support.
- Updated browser automation, file watching, Go 1.26 toolchain, and GitHub Actions dependencies.
