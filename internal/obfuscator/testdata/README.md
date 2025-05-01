# Obfuscator Test Data

This directory contains test data used by the obfuscator unit tests.

## Directory Structure

- `golden/context/`: Contains golden files used for testing context saving/loading functionality. These files were previously in `dist_test/context/` at the project root.
- `inputs/`: Contains input PHP files specifically for obfuscator tests.

## Purpose

These test files are intended to be used by unit tests located in the `internal/obfuscator` package. For integration tests that require PHP execution, see the `tests/integration` directory instead. 