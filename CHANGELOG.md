# Changelog

## Unreleased

- Updated the preferred Go build toolchain to 1.27.1 and aligned CI and release builds, retaining Go 1.26 source compatibility.
- Fixed `dom type --clear` writing `undefined` instead of clearing inputs and textareas with current browser dependencies.
- Updated browser automation, file watching, Go 1.26 toolchain, and GitHub Actions dependencies.
- Fixed documented `dom` subcommands rejecting the `--mode` flag.
