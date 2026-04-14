package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// --- Data Models ---

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
	Status  string   `json:"status"`
	Message string   `json:"message,omitempty"`
	Data    *Profile `json:"data"`
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

// --- Main Handler ---

func profileHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var body struct {
		Name interface{} `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	// 1. Validation Logic
	if body.Name == nil || body.Name == "" {
		sendError(w, http.StatusBadRequest, "Missing or empty name")
		return
	}
	nameStr, ok := body.Name.(string)
	if !ok {
		sendError(w, http.StatusUnprocessableEntity, "Name must be a string")
		return
	}

	// 2. Idempotency Handling
	var p Profile
	err := db.QueryRow(`SELECT id, name, gender, gender_probability, sample_size, age, age_group, country_id, country_probability, created_at 
		FROM profiles WHERE name = ?`, nameStr).Scan(
		&p.ID, &p.Name, &p.Gender, &p.GenderProbability, &p.SampleSize, &p.Age, &p.AgeGroup, &p.CountryID, &p.CountryProbability, &p.CreatedAt)

	if err == nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(SuccessResponse{
			Status:  "success",
			Message: "Profile already exists",
			Data:    &p,
		})
		return
	}

	// 3. Multi-API Integration with Error Handling
	client := &http.Client{Timeout: 10 * time.Second}

	// Fetch Genderize
	respG, err := client.Get("https://api.genderize.io?name=" + nameStr)
	if err != nil {
		sendError(w, http.StatusBadGateway, "Genderize API unreachable")
		return
	}
	defer respG.Body.Close()
	var gRes GenderizeRes
	json.NewDecoder(respG.Body).Decode(&gRes)
	if gRes.Gender == nil || gRes.Count == 0 {
		sendError(w, http.StatusNotFound, "Genderize returned no data")
		return
	}

	// Fetch Agify
	respA, err := client.Get("https://api.agify.io?name=" + nameStr)
	if err != nil {
		sendError(w, http.StatusBadGateway, "Agify API unreachable")
		return
	}
	defer respA.Body.Close()
	var aRes AgifyRes
	json.NewDecoder(respA.Body).Decode(&aRes)
	if aRes.Age == nil {
		sendError(w, http.StatusNotFound, "Agify returned no data")
		return
	}

	// Fetch Nationalize
	respN, err := client.Get("https://api.nationalize.io?name=" + nameStr)
	if err != nil {
		sendError(w, http.StatusBadGateway, "Nationalize API unreachable")
		return
	}
	defer respN.Body.Close()
	var nRes NationalizeRes
	json.NewDecoder(respN.Body).Decode(&nRes)
	if len(nRes.Country) == 0 {
		sendError(w, http.StatusNotFound, "Nationalize returned no country data")
		return
	}

	// 4. Processing & Filtering Logic
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

	// 5. Data Persistence
	_, err = db.Exec(`INSERT INTO profiles (id, name, gender, gender_probability, sample_size, age, age_group, country_id, country_probability, created_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		profile.ID, profile.Name, profile.Gender, profile.GenderProbability, profile.SampleSize, 
		profile.Age, profile.AgeGroup, profile.CountryID, profile.CountryProbability, profile.CreatedAt)

	if err != nil {
		sendError(w, http.StatusInternalServerError, "Database write error")
		return
	}

	// 6. Final Response
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(SuccessResponse{
		Status: "success",
		Data:   &profile,
	})
}

func main() {
	initDB()
	defer db.Close()

	http.HandleFunc("/api/profiles", profileHandler)

	fmt.Println("Server listening on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
