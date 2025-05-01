# Transformer Test Data

This directory contains test data used by transformer unit tests.

## Directory Structure

- `arithmetic/`: Contains test PHP files for arithmetic-related transformations.
- `array_access/`: Contains test PHP files for array access-related transformations.
- `classes/`: Contains test PHP files with class-related code.
- `complex/`: Contains complex PHP examples that exercise multiple transformations.
- `control_flow/`: Contains test PHP files for control flow obfuscation. These files were previously in `examples/control_flow_test/` at the project root.
- `dead_code/`: Contains test PHP files for dead code insertion transformations.
- `simple/`: Contains simple PHP examples for basic transformation tests.
- `strings/`: Contains test PHP files for string-related transformations.

## Purpose

These test files are intended to be used by unit tests located in the `internal/transformer` package. For integration tests that require PHP execution, see the `tests/integration` directory instead. 