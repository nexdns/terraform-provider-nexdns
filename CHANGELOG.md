# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.1] - 2026-11-22

### Security

- Built with Go 1.26.11 and `google.golang.org/grpc` 1.85.0. Release 1.0.0 shipped
  gRPC 1.67.1 and an older Go, both of which have since received security fixes;
  `go.mod` now names the toolchain, so release builds use the patched one.

### Changed

- Built with `terraform-plugin-framework` 1.19.0 and `terraform-plugin-go` 0.31.0
  (release 1.0.0 used 1.13.0 and 0.25.0).

## [1.0.0] - 2026-07-31

Initial public release.
