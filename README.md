# Gigpurse Backend

A Go backend built with **Clean Architecture** principles.

## Project Structure

```
Gigpurse/
├── cmd/
│   └── gigpurse/
│       └── main.go       # Application entry point and dependency injection
├── internal/
│   ├── domain/           # Core enterprise business rules (Entities & Interfaces)
│   ├── usecase/          # Application business rules (orchestrates domain entities)
│   ├── repository/       # Data storage implementations (memory, databases)
│   └── delivery/         # Transport layer (HTTP handlers, routes)
├── go.mod                # Module definitions
└── README.md             # This file
```

## Layers Explained

1. **Domain (`internal/domain`)**: Core business objects and contract interfaces. This layer is completely independent of external libraries, databases, and frameworks.
2. **Usecase (`internal/usecase`)**: Contains application-specific business logic. It orchestrates the flow of data to and from the domain entities.
3. **Repository (`internal/repository`)**: Data access layer implementations. Currently contains an in-memory store, which can easily be replaced by SQL/NoSQL databases without changing the business logic.
4. **Delivery (`internal/delivery`)**: Exposes the application logic to the outer world. Currently implemented with HTTP endpoints using the standard library.

## Running the Server

Start the server by running:

```bash
go run cmd/gigpurse/main.go
```

The server runs on port `:8080`.

### Endpoints

* **Create Wallet**
  * `POST /wallet`
  * Body: `{"user_id": "user_id_here"}`
* **Get Balance**
  * `GET /wallet?user_id=user_id_here`
* **Deposit**
  * `POST /wallet/deposit`
  * Body: `{"user_id": "user_id_here", "amount": 100.50}`
