# d-im Go SDK

Go SDK for calling the d-im HTTP API from a business service.

## Local development

The SDK and demo are independent Go modules. No root `go.work` is required.

```bash
cd sdk/go
go test ./...

cd demo
go run .
```

The demo reads these environment variables:

- `IM_BASE_URL`, defaulting to `http://localhost:8080`
- `JWT_API_KEY`, defaulting to `im-api-key-change-me`

## Synchronize a user

`UpsertUser` writes a complete snapshot. Increment `Version` for every business-side user change; retrying the same version is idempotent and older versions are rejected.

```go
err := client.UpsertUser(ctx, dimsdk.UserData{
    UserID:   "user-123",
    Nickname: "Alice",
    Avatar:   "https://example.com/alice.png",
    Status:   "active",
    Version:  42,
})
```
