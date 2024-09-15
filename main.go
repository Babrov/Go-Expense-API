package main

import (
    "encoding/json"
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
)

// Weather represents the weather data structure
type Weather struct {
    City        string  `json:"city"`
    Temperature float64 `json:"temperature"`
    Condition   string  `json:"condition"`
}

// GetWeather handles the /weather route and responds with weather data
func GetWeather(w http.ResponseWriter, r *http.Request) {
    // Static weather data
    weatherData := Weather{
        City:        "Kyiv",
        Temperature: 22.5,
        Condition:   "Sunny",
    }

    // Set JSON content type and encode the weather data as JSON
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(weatherData)
}

func main() {
    // Initialize the Chi router
    r := chi.NewRouter()

    // Use built-in middleware such as logging, recovering from panics, etc.
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)

    // Define the /weather route
    r.Get("/weather", GetWeather)

    // Start the HTTP server on port 8080
    http.ListenAndServe(":8080", r)
}
