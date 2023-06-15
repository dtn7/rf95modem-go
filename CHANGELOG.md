# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog][keep-a-changelog], and this project adheres to [Semantic Versioning][semantic-versioning].

## [Unreleased]
### Changed
- Breaking API changes: `Modem` now uses a `context.Context` which must be passed during creation.
- Adapted the [Keep a Changelog][keep-a-changelog] format and created a changelog from prior git tags.
- Changed default branch from `master` to `main`.
- Bumped dependencies.

### Fixed
- Restored compatibility with rf95modem upstream code, version 0.7.3 including later commits.

## [0.3.2] - 2023-02-25
> _This release was created before adapting the [Keep a Changelog][keep-a-changelog] format._

Bump dependencies

## [0.3.1] - 2019-12-27
> _This release was created before adapting the [Keep a Changelog][keep-a-changelog] format._

Fix blocking in case of a missing Read

## [0.3.0] - 2019-12-21
> _This release was created before adapting the [Keep a Changelog][keep-a-changelog] format._

Add handler registration for RX messages to API

## [0.2.0] - 2019-12-17
> _This release was created before adapting the [Keep a Changelog][keep-a-changelog] format._

Generalized Modem API to allow different endpoints

## [0.1.4] - 2019-12-10
> _This release was created before adapting the [Keep a Changelog][keep-a-changelog] format._

Adapted code for rf95modem 0.5.1

## [0.1.3] - 2019-11-06
> _This release was created before adapting the [Keep a Changelog][keep-a-changelog] format._

Adapted code to rf95modem 0.4

## [0.1.2] - 2019-10-25
> _This release was created before adapting the [Keep a Changelog][keep-a-changelog] format._

Update library to latest version of rf95modem

## [0.1.1] - 2019-10-23
> _This release was created before adapting the [Keep a Changelog][keep-a-changelog] format._

Fix race condition when sending parallel messages

## [0.1.0] - 2019-10-22
> _This release was created before adapting the [Keep a Changelog][keep-a-changelog] format._

First release

[keep-a-changelog]: https://keepachangelog.com/en/1.1.0/
[semantic-versioning]: https://semver.org/spec/v2.0.0.html

[unreleased]: https://github.com/dtn7/rf95modem-go/compare/v0.3.2...HEAD
[0.3.2]: https://github.com/dtn7/rf95modem-go/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/dtn7/rf95modem-go/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/dtn7/rf95modem-go/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/dtn7/rf95modem-go/compare/v0.1.4...v0.2.0
[0.1.4]: https://github.com/dtn7/rf95modem-go/compare/v0.1.3...v0.1.4
[0.1.3]: https://github.com/dtn7/rf95modem-go/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/dtn7/rf95modem-go/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/dtn7/rf95modem-go/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/dtn7/rf95modem-go/releases/tag/v0.1.0
