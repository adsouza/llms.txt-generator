# Guide for AI Agents

This codebase uses the Clean Architecture with aspects for cross-cutting concerns (e.g. retries).
We are using Go 1.25 so stick to modern idioms; e.g. use the any keyword instead of interface{}.
Mocks tend to be fragile so avoid them in tests; use integration-style testing with shared fakes instead. 
Use 'bd' for task tracking if it is available on the system.
Remember to run `gofmt`, `go vet ./...` & `staticcheck ./...` after changing code. 
