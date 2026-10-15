# Go-Sample

A small Go HTTP API example for registering and authenticating users and managing
their wallet balances. It demonstrates HTTP handlers, middleware, interfaces,
AWS DynamoDB storage, and decimal arithmetic.

## How it works

- `application.go` initializes the AWS SDK client, creates the auth and wallet
  services, and registers the HTTP routes.
- `auth/` handles registration, login, logout, and JWT-cookie authentication.
  Passwords are stored as bcrypt hashes. Wallet endpoints require a valid
  `auth_token` cookie.
- `internal/` contains the wallet API handlers, request/response types, the
  `AccountStore` interface, and account-store implementations. The running
  application uses the DynamoDB implementation; in-memory channel- and
  mutex-based implementations are also included as examples.
- `docs/` contains the generated Swagger documentation, served at `/swagger/`.

After logging in, the username in the authenticated token identifies the wallet
for deposit, withdrawal, and balance requests.

## Requirements

- Go version specified by `go.mod`
- AWS credentials available through the AWS SDK's default credential chain
- Access to DynamoDB in the `eu-north-1` region
- Two DynamoDB tables:
  - `Login`, with a string partition key named `UserName`
  - `UserWallet` (or the table selected with `DYNAMODB_TABLE_NAME`), with a
    string partition key named `UserId`

The application uses `Login` as the authentication table name. The wallet
table defaults to `UserWallet`.

## Run

From the repository root:

```sh
go run .
```

The server listens on port `8080` by default. Set `PORT` to use a different
port, or `DYNAMODB_TABLE_NAME` to select a different wallet table.

## API examples

Register a user:

```sh
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"player1\",\"password\":\"example-password\"}"
```

Log in and save the authentication cookie:

```sh
curl -i -c cookies.txt -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"player1\",\"password\":\"example-password\"}"
```

Use that cookie for wallet requests:

```sh
curl -b cookies.txt -X POST http://localhost:8080/deposit \
  -H "Content-Type: application/json" \
  -d "{\"amount\":10.50}"

curl -b cookies.txt -X POST http://localhost:8080/withdraw \
  -H "Content-Type: application/json" \
  -d "{\"amount\":2.25}"

curl -b cookies.txt http://localhost:8080/balance
```

Log out with `POST /logout`, also using the saved cookie. For the complete API
schema, open `http://localhost:8080/swagger/` after starting the server.

## Test

Run the Go test suite from the repository root:

```sh
go test ./...
```

> **Note:** This is an example project, not a production-ready authentication
> service. Configure a strong, private JWT signing secret and enable secure
> cookies over HTTPS before deploying it.
