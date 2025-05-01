# Integration Tests

This directory contains integration tests and their supporting files.

## Directory Structure

- `integration/`: Contains integration test Go files that test the full application or require PHP execution.
- `internal/`: Contains helper packages specifically for integration tests.
- `testdata/integration/`: Contains test data files specifically for integration tests.

## Purpose

Unlike unit tests (co-located with the code they test in `internal/` and `pkg/`), integration tests:

- Require external dependencies (like running PHP).
- Test the interaction between multiple packages.
- Test the command-line interface or the public API end-to-end.

## Running Tests

To run all integration tests:

```bash
go test ./tests/...
```

Note that these tests require PHP to be installed and available in your PATH. 