package main

import (
	"encoding/json"
	"log"
	"fmt"
	"net/http"
	"os"
	"io"
	"strconv"
)

var routemap = map[string]string  {"/api/movies": "movies",
                                   "/api/users": "/monolith",
								   "/api/payments": "/monolith",
								   "/api/subscriptions": "/monolith", }

func main() {
	

	// Set up HTTP routes

	// Check our health, not monolith's health
	http.HandleFunc("/health", healthHandler)
	// Need to forward movies to the extracted movie service
	http.HandleFunc("/api/movies", handleMovies)

	// These need to be forwarded to old monolith app still
	http.HandleFunc("/api/users", route2Monolith)
	http.HandleFunc("/api/payments", route2Monolith)
	http.HandleFunc("/api/subscriptions", route2Monolith)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	log.Printf("Starting API Gateway on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"status": true})
}


var totNumReq uint64
var numReqMovie uint64
func handleMovies(w http.ResponseWriter, r *http.Request) {
	mov_url := os.Getenv("MOVIES_SERVICE_URL")
	if mov_url == "" {
		mov_url = "http://movies-service:8081"
	}

	//Feature toggle
	grdMigration, _ := strconv.ParseBool (os.Getenv("GRADUAL_MIGRATION") )
	procents, _ := strconv.ParseInt (os.Getenv("MOVIES_MIGRATION_PERCENT"), 10, 64)
	log.Printf ("ENVIRONMENT: %d, %d", grdMigration, procents)
	if grdMigration && procents > 0 {
		totNumReq++
		if (numReqMovie+1) * (100 / uint64(procents)) > totNumReq 	{
			route2Monolith (w, r)
		} else {
			numReqMovie += 1
			forwardRequest (w, r, mov_url)		
		}
	}
}

func route2Monolith(w http.ResponseWriter, r *http.Request) {
	// Configuration
	
	mon_url := os.Getenv("MONOLITH_URL")
	if mon_url == "" {
		mon_url = "http://monolith:8080"
	}
	
	fmt.Println("Proxy: forwarding to monolith: " + mon_url)

	forwardRequest (w, r, mon_url)
}

func forwardRequest(w http.ResponseWriter, r *http.Request, targetHost string) {    
    // Создаем новый запрос

	targetURL := targetHost + r.URL.Path

    req, err := http.NewRequest(r.Method, targetURL, r.Body)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Копируем все заголовки
    for key, values := range r.Header {
        for _, value := range values {
            req.Header.Add(key, value)
        }
    }
    
    // Копируем Query параметры
    req.URL.RawQuery = r.URL.RawQuery

	//req.Header.Set("X-Forwarded-For", r.RemoteAddr)
    //req.Header.Set("Host", target)


    // Отправляем запрос
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadGateway)
        return
    }
    defer resp.Body.Close()
    
    // Копируем заголовки ответа
    for key, values := range resp.Header {
        for _, value := range values {
            w.Header().Add(key, value)
        }
    }
    
    // Копируем статус код
    w.WriteHeader(resp.StatusCode)
    
    // Копируем тело
    io.Copy(w, resp.Body)
}