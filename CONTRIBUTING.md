# Contributing to tooltrace

Thank you for your interest in contributing to `tooltrace`! This project is in its early stages, and we welcome your help.

## Trace Format Stability

The core trace format (documented in [TOOLTRACE.md](TOOLTRACE.md)) is intentionally stable. It acts as a public contract. Any changes or additions to the format must be discussed in an issue first, as they require careful consideration to avoid breaking backward compatibility.

## Running Tests Locally

Before submitting a pull request, please ensure that all tests pass and the code is properly formatted and vetted.

1. **Format the code**:
   ```bash
   gofmt -w .
   ```
2. **Run the linter / vet**:
   ```bash
   go vet ./...
   ```
3. **Run the full test suite**:
   ```bash
   go test ./... -v
   ```

## Filing a Bug Report

If you encounter an issue, please file a bug report using our issue templates. A good bug report includes:
- The version of `tooltrace` you are using.
- Your OS and Go version.
- A minimal, reproducible example (if applicable).
- The expected behavior versus the actual behavior.

Please do **NOT** attach trace files that contain sensitive information (API keys, passwords, etc.) to your bug reports, as automatic redaction is not yet supported.

To file a bug or request a feature, head over to the [Issues](../../issues) tab and select the appropriate template.
