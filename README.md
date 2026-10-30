# go-rv!

> A lightweight URL shortener built with Go, MongoDB Atlas, and Gemini AI.

`go-rv!` creates short links, enriches them with AI-generated descriptions and tags, persists them in MongoDB Atlas, and tracks redirects with atomic hit counters. It is a technical demonstration of a small REST service, safe concurrent storage, containerization, and CI/CD deployment to AWS EC2.

---

## Author & Copyright

**Author:** Marc Bonet Bretto<br>
**Copyright:** © 2026 Marc Bonet Bretto. All rights reserved.

---

## 📝 Motivation & Objective

The project demonstrates a simple URL-shortening service with a clear division of responsibilities:

* **HTTP API:** Go's standard `net/http` package exposes endpoints for creating, listing, looking up, and redirecting short links.
* **Business logic:** A controller normalizes destination URLs, reuses existing records, generates unique codes, and requests metadata from Gemini.
* **Persistence:** A storage interface separates the controller from its storage implementation. MongoDB Atlas provides persistent storage in the deployed application; a concurrency-safe in-memory implementation is also available.
* **Operational safeguards:** Short-code uniqueness, atomic hit increments, a global creation rate limit, startup checks, and graceful HTTP shutdown.

---

## ⚡ Technology Stack

* **Language and HTTP server:** Go 1.27, using the standard library.
* **Database:** MongoDB Atlas with the official MongoDB Go driver v2.
* **AI metadata:** Gemini API. The service sends the API key in the `x-goog-api-key` request header, not in the URL.
* **Containerization:** Multi-stage Docker build with a small Alpine runtime image.
* **CI/CD:** GitHub Actions builds and pushes the image to Docker Hub, then deploys it to AWS EC2 over SSH.

---

## 🧩 System Architecture

```text
                         HTTPS
 [ Browser / API Client ] ───────────────┐
                                        ▼
                              [ AWS TLS Termination ]
                                        │ HTTP :8080
                                        ▼
                           ┌────────────────────────┐
                           │ go-rv! container (Go)  │
                           │                        │
                           │ Router → Controller    │──────► [ Gemini API ]
                           │             │          │          URL metadata
                           └─────────────┼──────────┘
                                         │
                            Store interface (production)
                                         │
                                         ▼
                           [ MongoDB Atlas: short_urls ]
```

TLS terminates at the AWS layer. The application listens for plain HTTP on port `8080` inside the container; TLS is not configured in the container.

---

## ⚙️ Configuration & Secrets

Non-secret service settings are defined in `config.go`. MongoDB credentials and the Gemini API key are supplied at runtime through environment variables.

| Setting | Source | Purpose |
| :--- | :--- | :--- |
| `GEMINI_API_KEY` | Environment variable | Authenticates requests to the Gemini API. |
| `MONGODB_PASSWORD` | Environment variable | Password for the MongoDB database user. |
| MongoDB server | `MongoDBServer` in `config.go` | Atlas cluster hostname. |
| MongoDB username | `MongoDBUsername` in `config.go` | Atlas database user. |
| MongoDB database | `MongoDBName` in `config.go` | `go-rv`. |
| MongoDB collection | `MongoDBCollection` in `config.go` | `short_urls`. |

### Local execution

For local development, create these credential files on your machine:

```text
assets/gemini/gemini_api.key
assets/mongo/db_password
```

Both files are excluded from Git and the Docker build context. `run.sh` reads them, exports `GEMINI_API_KEY` and `MONGODB_PASSWORD`, builds the application, and starts it:

```bash
bash ./run.sh
```

To build without starting the service:

```bash
bash ./run.sh --noRun
```

The local process requires a reachable Atlas cluster and a database user with permission to create indexes and read/write records in the configured database.

### Production secrets

Configure these GitHub repository secrets for the deployment workflow:

* `DOCKERHUB_USERNAME`
* `DOCKERHUB_PASSWORD`
* `AWS_EC2_HOST`
* `AWS_EC2_USER`
* `AWS_SSH_PRIVATE_KEY`
* `GEMINI_API_KEY`
* `MONGODB_PASSWORD`

GitHub Actions forwards the two application secrets to the remote deployment command, and Docker injects them into the container as environment variables. Do not put secret values in `config.go`, the workflow, or committed files.

---

## 🗃️ Storage & Data Model

The controller depends on the `storage.Store` interface. `MongoStore` is initialized by `main.go` and used by the running application. `MemoryStore` provides an in-memory implementation of the same operations.

Each MongoDB document in `go-rv.short_urls` represents a shortened destination. Its `_id` is the normalized destination key; the short code has a unique index.

```json
{
  "_id": "example.com/page",
  "url": "https://example.com/page/",
  "short_url": "https://go-rv.mbonet.xyz:443/0123456789ab",
  "code": "0123456789ab",
  "ai_tags": ["example", "documentation", "web"],
  "ai_description": "A site that provides example documentation.",
  "hits": 0,
  "created_at": "2026-10-30T12:00:00Z"
}
```

* **Duplicate destinations:** The controller generates a normalized key and returns the existing record when that key is already stored. HTTP/HTTPS schemes, conventional ports, trailing slashes, and letter case are normalized for duplicate comparison.
* **Unique short codes:** Codes are generated from cryptographically random bytes. MongoDB enforces uniqueness with an index; the controller retries if a generated code collides.
* **Hit counting:** MongoDB increments `hits` atomically. The in-memory store protects the increment with a write lock. Both return a copy of the updated record.
* **Metadata:** Gemini receives the submitted URL as prompt text and returns a description and tags. The service does not fetch or crawl the destination website itself.

---

## 🔄 Request Flow

1. A client submits a destination URL to `POST /api/url`.
2. The router validates the JSON request and enforces the global limit of 10 creation requests per rolling minute.
3. The controller checks MongoDB for an existing normalized destination. If one exists, it returns that record without making another Gemini request.
4. For a new destination, Gemini generates a description and tags. The controller creates a unique short code and stores the record.
5. A request to `/{code}` atomically increments the hit count and redirects the client to the destination with HTTP `302 Found`.

---

## 🔌 REST API

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `GET` | `/` | Serves the project overview and interactive URL creation form. |
| `GET` | `/health` | Returns service status, timestamp, and uptime. |
| `POST` | `/api/url` | Creates a short link or returns an existing one. Accepts `{"url":"https://example.com/page"}`. Returns `201 Created` for a new record and `200 OK` when reusing one. |
| `GET` | `/api/url` | Lists stored records keyed by short code. |
| `GET` | `/api/url/{code}` | Retrieves a record by its short code. |
| `GET` | `/{code}` | Increments the hit count and redirects to the original URL. |

Creation requests are limited globally to 10 per rolling minute. Excess requests receive `429 Too Many Requests` and a `Retry-After` header.

---

## 🚀 Local Build & Run

Requirements: Go 1.27.1, Bash, the credential files listed above, and network access to MongoDB Atlas and the Gemini API.

```bash
bash ./run.sh
```

The service listens on `0.0.0.0:8080`. Open `http://localhost:8080/` to use the landing page. Stop the process with `Ctrl+C`; the server then attempts a graceful shutdown.

---

## 🐳 Docker & Deployment

The Dockerfile compiles a static Linux binary in a Go builder stage and copies it into an Alpine runtime image. The container exposes port `8080` and expects `GEMINI_API_KEY` and `MONGODB_PASSWORD` at runtime.

The workflow in `.github/workflows/deploy.yml` runs on pushes to `main` or by manual dispatch. It runs `go test -v ./...`, builds and pushes the `latest` image to Docker Hub, then connects to AWS EC2 over SSH to replace and start the `go-rv-app` container. The container port is mapped to host port `8080`; HTTPS is handled by the AWS layer in front of it.

---

## 🧱 Repository Structure

```text
go-rv!/
├── .github/workflows/deploy.yml  # Build and deploy pipeline
├── config.go                     # Service and MongoDB settings
├── Dockerfile                    # Multi-stage container build
├── go.mod                        # Go module and MongoDB driver dependency
├── go.sum                        # Dependency checksums
├── main.go                       # MongoDB initialization and HTTP server lifecycle
├── run.sh                        # Local credential loading, build, and run
└── internal/
    ├── controller/               # URL normalization, code generation, business logic
    ├── router/                   # REST endpoints and POST rate limiter
    ├── service/                  # Gemini API client
    ├── storage/                  # Store interface, memory store, and MongoDB store
    └── validator/                # Request and short-code validation
```
