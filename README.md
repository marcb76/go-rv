# go-rv!

> A high-performance in-memory URL shortener built in Go — Developed as a technical PoC for RVPR.

This microservice was designed and built from scratch in a single day to demonstrate execution speed, clean concurrent architecture with thread-safe in-memory maps, and production-grade observability.

---

## 📋 Executive Summary

**`go-rv!`** is a high-performance, concurrent, in-memory URL shortener and routing microservice built entirely from scratch using **Go**. As a technical Proof of Concept (PoC), it showcases enterprise backend engineering practices, thread-safe memory management without heavy external framework overhead, and intelligent metadata enrichment powered by the **Gemini API**.

---

## 🏗️ Core Architecture & Data Model

The application bypasses traditional database latency by leveraging Go's native primitives for maximum throughput and near-zero latency storage.

### Data Structure (`URLRecord`)
Each shortened URL entity is stored in memory with the following structure:
* `url` (string): The original long destination URL.
* `shortUrl` (string): The generated short routing path... and `map-key` of entry in memory's map.
* `aiTags` ([]string): Intelligent tags or categories generated automatically via AI analysis upon creation.
* `aiDescription` (string): A brief AI-driven description summarizing the content of the link.
* `hits` (int): A concurrency-safe click counter.

### Concurrency Control
The in-memory repository relies on a standard Go `map[string]URLRecord` protected by a **`sync.RWMutex`**. This guarantees safe, race-free writes during link creation and click increments while optimizing high-throughput read operations for redirection.

---

## 🗺️ API Contract (Endpoints)

| Endpoint | Method | Description |
| :--- | :--- | :--- |
| **`/health`** | `GET` | **Infrastructure Monitoring:** Returns a JSON payload with service health status, timestamp, and server uptime (tailored for orchestrators like AWS ECS/K8s). |
| **`/api/url`** | `POST` | **Creation & AI Enrichment:** Accepts a JSON body with a long URL (`{"url": "..."}`), triggers an integration with the Gemini API to generate `aiTags` and `aiDescription`, allocates a unique shortUrl, persists the record, and returns the complete `URLRecord` in the response body. |
| **`/api/url`** | `GET` | **Global Query:** Returns the complete in-memory map containing all registered items and their associated metadata in JSON format. |
| **`/api/url/{shortUrl}`** | `GET` | **Individual Query:** Looks up a specific record by its unique `code` key and returns its `URLRecord` in JSON format. |
| **`/{shortUrl}`** | `GET` | **Redirection & Metrics:** Intercepts root requests matching a registered `shortUrl`, executes an HTTP 302 redirect to the original URL, and concurrently increments the `hits` counter using a lightweight Go `goroutine`. |

---

## ⚙️ Engineering & DevOps Stack

* **Backend:** Go (Standard library `net/http`, concurrency primitives, strict type safety).
* **AI Integration:** Gemini API for automated content tagging and description generation.
* **Containerization:** Multi-stage Docker builds optimized for minimal image size and secure production runtimes.
* **Automation & CI/CD:** GitHub Actions pipelines managing continuous building, testing, and cloud deployment workflows.
* **Testing:** Automated Python integration test scripts to validate service endpoints post-deployment.