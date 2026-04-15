package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// --- Models ---

type Profile struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	Gender             string  `json:"gender"`
	GenderProbability  float64 `json:"gender_probability"`
	SampleSize         int     `json:"sample_size"`
	Age                int     `json:"age"`
	AgeGroup           string  `json:"age_group"`
	CountryID          string  `json:"country_id"`
	CountryProbability float64 `json:"country_probability"`
	CreatedAt          string  `json:"created_at"`
}

type SuccessResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Count   int         `json:"count,omitempty"`
	Data    interface{} `json:"data"`
}

type ErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// --- API Schema Helpers ---

type GenderizeRes struct {
	Gender      *string `json:"gender"`
	Probability float64 `json:"probability"`
	Count       int     `json:"count"`
}

type AgifyRes struct {
	Age *int `json:"age"`
}

type NationalizeRes struct {
	Country []struct {
		CountryID   string  `json:"country_id"`
		Probability float64 `json:"probability"`
	} `json:"country"`
}

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite", "./profiles.db")
	if err != nil {
		log.Fatal("DB Connection Error:", err)
	}

	query := `
	CREATE TABLE IF NOT EXISTS profiles (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE,
		gender TEXT,
		gender_probability REAL,
		sample_size INTEGER,
		age INTEGER,
		age_group TEXT,
		country_id TEXT,
		country_probability REAL,
		created_at TEXT
	);`
	if _, err := db.Exec(query); err != nil {
		log.Fatal("Table Creation Error:", err)
	}
}

// --- Utility Functions ---

func sendError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{Status: "error", Message: message})
}

func getAgeGroup(age int) string {
	if age <= 12 { return "child" }
	if age <= 19 { return "teenager" }
	if age <= 59 { return "adult" }
	return "senior"
}

// --- Handlers ---

func router(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Simple path routing
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	if len(parts) == 2 && parts[0] == "api" && parts[1] == "profiles" {
		if r.Method == http.MethodPost {
			createProfile(w, r)
			return
		}
		if r.Method == http.MethodGet {
			getAllProfiles(w, r)
			return
		}
	}

	if len(parts) == 3 && parts[0] == "api" && parts[1] == "profiles" {
		id := parts[2]
		if r.Method == http.MethodGet {
			getSingleProfile(w, r, id)
			return
		}
		if r.Method == http.MethodDelete {
			deleteProfile(w, r, id)
			return
		}
	}

	sendError(w, http.StatusNotFound, "Endpoint not found")
}

// POST /api/profiles
func createProfile(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name interface{} `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	if body.Name == nil || body.Name == "" {
		sendError(w, http.StatusBadRequest, "Missing or empty name")
		return
	}
	nameStr, ok := body.Name.(string)
	if !ok {
		sendError(w, http.StatusUnprocessableEntity, "Name must be a string")
		return
	}

	// Idempotency check
	var p Profile
	err := db.QueryRow(`SELECT id, name, gender, gender_probability, sample_size, age, age_group, country_id, country_probability, created_at 
		FROM profiles WHERE name = ?`, nameStr).Scan(
		&p.ID, &p.Name, &p.Gender, &p.GenderProbability, &p.SampleSize, &p.Age, &p.AgeGroup, &p.CountryID, &p.CountryProbability, &p.CreatedAt)

	if err == nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(SuccessResponse{Status: "success", Message: "Profile already exists", Data: &p})
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}

	// Fetch External APIs with specific 502 error mapping
	respG, err := client.Get("https://api.genderize.io?name=" + nameStr)
	if err != nil { return } 
	defer respG.Body.Close()
	var gRes GenderizeRes
	json.NewDecoder(respG.Body).Decode(&gRes)
	if gRes.Gender == nil || gRes.Count == 0 {
		sendError(w, http.StatusBadGateway, "Genderize returned an invalid response")
		return
	}

	respA, err := client.Get("https://api.agify.io?name=" + nameStr)
	if err != nil { return }
	defer respA.Body.Close()
	var aRes AgifyRes
	json.NewDecoder(respA.Body).Decode(&aRes)
	if aRes.Age == nil {
		sendError(w, http.StatusBadGateway, "Agify returned an invalid response")
		return
	}

	respN, err := client.Get("https://api.nationalize.io?name=" + nameStr)
	if err != nil { return }
	defer respN.Body.Close()
	var nRes NationalizeRes
	json.NewDecoder(respN.Body).Decode(&nRes)
	if len(nRes.Country) == 0 {
		sendError(w, http.StatusBadGateway, "Nationalize returned an invalid response")
		return
	}

	newID, _ := uuid.NewV7()
	profile := Profile{
		ID:                 newID.String(),
		Name:               nameStr,
		Gender:             *gRes.Gender,
		GenderProbability:  gRes.Probability,
		SampleSize:         gRes.Count,
		Age:                *aRes.Age,
		AgeGroup:           getAgeGroup(*aRes.Age),
		CountryID:          nRes.Country[0].CountryID,
		CountryProbability: nRes.Country[0].Probability,
		CreatedAt:          time.Now().UTC().Format(time.RFC3339),
	}

	_, err = db.Exec(`INSERT INTO profiles VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		profile.ID, profile.Name, profile.Gender, profile.GenderProbability, profile.SampleSize,
		profile.Age, profile.AgeGroup, profile.CountryID, profile.CountryProbability, profile.CreatedAt)

	if err != nil {
		sendError(w, http.StatusInternalServerError, "Database write error")
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(SuccessResponse{Status: "success", Data: &profile})
}

// GET /api/profiles
func getAllProfiles(w http.ResponseWriter, r *http.Request) {
	gender := r.URL.Query().Get("gender")
	country := r.URL.Query().Get("country_id")
	ageGroup := r.URL.Query().Get("age_group")

	query := "SELECT id, name, gender, age, age_group, country_id FROM profiles WHERE 1=1"
	var args []interface{}

	if gender != "" {
		query += " AND LOWER(gender) = LOWER(?)"
		args = append(args, gender)
	}
	if country != "" {
		query += " AND LOWER(country_id) = LOWER(?)"
		args = append(args, country)
	}
	if ageGroup != "" {
		query += " AND LOWER(age_group) = LOWER(?)"
		args = append(args, ageGroup)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	profiles := []map[string]interface{}{}
	for rows.Next() {
		var id, name, gen, ag, cid string
		var age int
		rows.Scan(&id, &name, &gen, &age, &ag, &cid)
		profiles = append(profiles, map[string]interface{}{
			"id": id, "name": name, "gender": gen, "age": age, "age_group": ag, "country_id": cid,
		})
	}

	json.NewEncoder(w).Encode(SuccessResponse{Status: "success", Count: len(profiles), Data: profiles})
}

// GET /api/profiles/{id}
func getSingleProfile(w http.ResponseWriter, r *http.Request, id string) {
	var p Profile
	err := db.QueryRow(`SELECT id, name, gender, gender_probability, sample_size, age, age_group, country_id, country_probability, created_at 
		FROM profiles WHERE id = ?`, id).Scan(
		&p.ID, &p.Name, &p.Gender, &p.GenderProbability, &p.SampleSize, &p.Age, &p.AgeGroup, &p.CountryID, &p.CountryProbability, &p.CreatedAt)

	if err == sql.ErrNoRows {
		sendError(w, http.StatusNotFound, "Profile not found")
		return
	}

	json.NewEncoder(w).Encode(SuccessResponse{Status: "success", Data: &p})
}

// DELETE /api/profiles/{id}
func deleteProfile(w http.ResponseWriter, r *http.Request, id string) {
	res, _ := db.Exec("DELETE FROM profiles WHERE id = ?", id)
	count, _ := res.RowsAffected()
	if count == 0 {
		sendError(w, http.StatusNotFound, "Profile not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func main() {
	initDB()
	defer db.Close()
	http.HandleFunc("/", router)
	fmt.Println("Server listening on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
