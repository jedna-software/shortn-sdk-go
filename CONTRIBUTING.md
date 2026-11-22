# Contributing

Thanks for helping improve the Shortn Link Go SDK.

## Development

```sh
gofmt -l .
go vet ./...
go test ./...
```

Unit tests use a mocked HTTP transport and run offline. Every SDK method has a case in
`TestEveryMethod` checking its method, path, query, body and decoding; add one for each new
method.

`smoke_test.go` is an opt-in test against a live, non-production instance. It is skipped
unless `SHORTN_API_KEY` is set; see the comment at its top for the other variables. It
creates resources with random names and removes them again.

## Pull Requests

- Keep changes focused.
- Update README, CHANGELOG and GAP_LIST when behavior changes.
- Add tests for new request/response behavior.
- Do not commit secrets or real customer data.
