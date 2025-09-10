# go-aiprovider

This repo handles any connections, requests, and responses to an AI provider

## Testing

For comprehensive testing instructions, including unit tests, integration tests, and environment setup, see [TESTING.md](TESTING.md).

Quick start:

```powershell
# Run unit tests (fast, no external dependencies)
go test -short ./...

# Run integration tests (requires .env file with API keys)
go test -run Integration ./...
```
