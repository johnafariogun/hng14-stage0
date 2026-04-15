---

# Profile Aggregator API (Stage 1)

A robust Go-based RESTful API that aggregates demographic data from multiple external sources, applies business logic for classification, and manages a persistent SQLite database.

## Features
* **Full CRUD Support**: Create, Read (single & filtered list), and Delete profiles.
* **Multi-API Data Mashup**: Real-time integration with **Genderize**, **Agify**, and **Nationalize**.
* **Intelligent Classification**:
    * **Age Grouping**: Categorizes age into `child`, `teenager`, `adult`, or `senior`.
    * **Nationality**: Automatically selects the country with the highest probability.
* **Idempotency**: Smart `POST` handling—if a name exists, the existing record is returned without creating a duplicate.
* **Case-Insensitive Filtering**: Filter the list of profiles by `gender`, `country_id`, or `age_group` regardless of casing.
* **UUID v7**: Uses time-ordered UUIDs for better database indexing performance.

## Tech Stack
* **Language**: Go 1.21+
* **Database**: SQLite (Pure Go driver: `modernc.org/sqlite`)
* **ID Generation**: `github.com/google/uuid` (UUID v7)

---

## Getting Started

### Prerequisites
* Go installed on your system.
* *Note: This version uses a pure Go SQLite driver, so no C compiler (GCC) is required.*

### Installation & Running
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

---

## API Documentation

### 1. Create Profile
`POST /api/profiles`
* **Request Body**: `{ "name": "ella" }`
* **Success (201)**: New record created.
* **Success (200)**: Name already exists, returns existing data with `"message": "Profile already exists"`.

### 2. Get All Profiles
`GET /api/profiles`
* **Query Parameters (Optional)**: `gender`, `country_id`, `age_group`.
* **Example**: `/api/profiles?gender=female&country_id=NG`
* **Filtering**: Logic is case-insensitive.

### 3. Get Single Profile
`GET /api/profiles/{id}`
* **Success (200)**: Returns full profile details.
* **Error (404)**: Profile not found.

### 4. Delete Profile
`DELETE /api/profiles/{id}`
* **Success (204)**: No Content.
* **Error (404)**: Profile not found.

---

## ⚠️ Error Handling
All error responses follow the structure: `{ "status": "error", "message": "<error message>" }`.

* **400 Bad Request**: Missing or empty name.
* **422 Unprocessable Entity**: Name is not a string.
* **502 Bad Gateway**: External API (Genderize/Agify/Nationalize) returned an invalid response.

---

## Database Schema

| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | TEXT | PRIMARY KEY (UUID v7) |
| `name` | TEXT | UNIQUE Name |
| `gender` | TEXT | male/female |
| `gender_probability` | REAL | Confidence score |
| `sample_size` | INTEGER | Original `count` from API |
| `age` | INTEGER | Predicted age |
| `age_group` | TEXT | child/teenager/adult/senior |
| `country_id` | TEXT | ISO Country Code |
| `country_probability`| REAL | Confidence score |
| `created_at` | TEXT | ISO 8601 UTC timestamp |

---

## Project Links
* **Live Deployment**: `https://hng14-stage0-production-d5bd.up.railway.app`
* **GitHub**: `https://github.com/johnafariogun/hng14-stage0`

