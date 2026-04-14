# Profile Aggregator API

A robust Go-based RESTful API that aggregates demographic data from multiple external sources, processes it with custom business logic, and persists the results in a SQLite database.

## Features

  * **Multi-API Integration**: Combines data from Genderize.io, Agify.io, and Nationalize.io.
  * **Data Transformation**: Automatically classifies age groups and identifies primary country origins.
  * **Idempotency**: Prevents duplicate records; returning existing data for repeated names.
  * **Persistence**: Uses SQLite for lightweight, reliable data storage.
  * **Strict Validation**: Handles missing data and invalid types with clear error responses.
  * **UUID v7**: Implements time-ordered UUIDs for primary keys.

## Tech Stack

  * **Language**: Go 1.21+
  * **Database**: SQLite3
  * **Driver**: `github.com/mattn/go-sqlite3`
  * **ID Generation**: `github.com/google/uuid`

-----

## Getting Started

### Prerequisites

  * Go installed on your machine.
  * A C compiler (GCC) installed (required for the SQLite driver).

### Installation

1.  **Clone the repository**:

    ```bash
    git clone https://github.com/yourusername/your-repo-name.git
    cd your-repo-name
    ```

2.  **Install dependencies**:

    ```bash
    go mod tidy
    ```

3.  **Run the server**:

    ```bash
    go run main.go
    ```

    The server will start on `http://localhost:8080`.

-----

## API Documentation

### Create/Retrieve Profile

**Endpoint**: `POST /api/profiles`

**Request Body**:

```json
{
  "name": "ella"
}
```

**Success Response (201 Created or 200 OK)**:

```json
{
  "status": "success",
  "data": {
    "id": "018e6b8c-...",
    "name": "ella",
    "gender": "female",
    "gender_probability": 0.99,
    "sample_size": 1234,
    "age": 46,
    "age_group": "adult",
    "country_id": "DRC",
    "country_probability": 0.85,
    "created_at": "2026-04-14T13:28:00Z"
  }
}
```

**Error Response (400/422/404)**:

```json
{
  "status": "error",
  "message": "Missing or empty name"
}
```

-----

## Testing with cURL

You can test the endpoint using the following command in your terminal:

```bash
curl -X POST http://localhost:8080/api/profiles \
     -H "Content-Type: application/json" \
     -d '{"name": "john"}'
```

-----

## Database Schema

The SQLite database `profiles.db` consists of a single table:

| Column | Type | Constraints |
| :--- | :--- | :--- |
| `id` | TEXT | PRIMARY KEY (UUID v7) |
| `name` | TEXT | UNIQUE |
| `gender` | TEXT | |
| `age` | INTEGER | |
| `age_group` | TEXT | Derived Logic |
| `country_id` | TEXT | ISO Code |
| `created_at` | TEXT | ISO 8601 UTC |

-----

## 🔗 Public API URL

[Insert your Deployment URL here, e.g., [https://your-app.render.com](https://www.google.com/search?q=https://your-app.render.com)]
