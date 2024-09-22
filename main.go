package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func RedisConnection(addr string, ctx context.Context) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}

	log.Printf("Redis is running on: %s\n", addr)

	return rdb
}

// WeatherResponse represents the weather data structure
type WeatherResponse struct {
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
	ResolvedAddress string  `json:"resolvedAddress"`
}

const weatherApiUrl = "https://weather.visualcrossing.com/VisualCrossingWebServices/rest/services/timeline"

// GetWeather handles the /weather route and responds with weather data
func GetWeather(location string) (*WeatherResponse, error) {
	apiKey := os.Getenv("WEATHER_API_KEY")

	apiUrl := fmt.Sprintf("%s/%s?key=%s", weatherApiUrl, location, apiKey)

	log.Printf("Fetching weather for %s\n", location)

	// Build the URL with query parameters
	apiUrlBuilder, err := url.Parse(apiUrl)

	if err != nil {
		return nil, fmt.Errorf("error parsing base URL: %v", err)
	}

	// Add query parameters
	params := url.Values{}
	params.Add("key", apiKey)
	params.Add("contentType", "json")
	params.Add("unitGroup", "metric")
	apiUrlBuilder.RawQuery = params.Encode()

	// Make the GET request
	response, err := http.Get(apiUrlBuilder.String())
	if err != nil {
		return nil, fmt.Errorf("error making GET request: %v", err)
	}
	defer response.Body.Close()

	// Check for a successful response
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	// Decode the JSON response
	var apiResponse WeatherResponse
	if err := json.NewDecoder(response.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("error decoding JSON response: %v", err)
	}

	log.Printf("Successfully fetched weather")

	return &apiResponse, nil

}

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
}

func main() {
	LoadEnv()

	redisAddr := os.Getenv("REDIS_ADDR")
	ctx := context.Background()

	router := chi.NewRouter()
	rdb := RedisConnection(redisAddr, ctx)

	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	// Define the route with a parameter {location}
	router.Get("/weather/{location}", func(w http.ResponseWriter, r *http.Request) {
		location := chi.URLParam(r, "location")
		cacheKey := strings.ToLower(location)

		cacheData, err := rdb.Get(ctx, cacheKey).Result()

		if err == redis.Nil {
			data, fetchErr := GetWeather(location)

			if fetchErr != nil {
				http.Error(w, "Failed to fetch weather data", http.StatusInternalServerError)
				log.Println("Error fetching weather data:", err)
				return
			}

			// Store the weather data in Redis with a 24-hour TTL
			err = rdb.Set(ctx, cacheKey, data, 24*time.Hour).Err()
			if err != nil {
				log.Fatal(err)
				http.Error(w, "Failed to store weather data in Redis", http.StatusInternalServerError)
				return
			}

			// Return the weather data
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(data)

		} else if err != nil {
			// Handle any other Redis errors
			http.Error(w, "Redis error", http.StatusInternalServerError)
			log.Println("Redis error:", err)
		} else {
			// Return the cached weather data from Redis
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(cacheData))
		}
	})

	// Start the HTTP server on port 8080
	http.ListenAndServe(":8080", router)
}
