# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Go module configuration (go.mod)
- Changelog documentation

### Fixed
- Module import paths now match repository URL
- Corrected type usage in main.go (LinearMete instantiation)
- Fixed Direction type conversion in Description struct
- Replaced deprecated ioutil.ReadFile with os.ReadFile

## [0.1.0] - 2020-08-09

### Added
- Initial implementation of legal description library
- Support for metes and bounds parsing from AutoCAD reports
- Linear mete (straight line boundaries) support
- Arc mete (curved boundaries) support
- Bearing calculations and conversions
- Direction enumeration and utilities
- Description template formatting
- Command-line interface for processing legal descriptions
- Test suite for core functionality

[Unreleased]: https://github.com/samuel-kreimeyer/Legal/compare/06f0358...HEAD
[0.1.0]: https://github.com/samuel-kreimeyer/Legal/releases/tag/v0.1.0
