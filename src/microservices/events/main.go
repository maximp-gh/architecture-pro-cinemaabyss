package main

import (
	"encoding/json"
	"log"
	//"fmt"
	"net/http"
	"os"
	"strings"
	"github.com/segmentio/kafka-go"
	"time"
	"context"
)

var routemap = map[string]string  {"/api/movies": "movies",
                                   "/api/users": "/monolith",
								   "/api/payments": "/monolith",
								   "/api/subscriptions": "/monolith", }
var writer *kafka.Writer
var reader *kafka.Reader

func main() {
	

	// Set up HTTP routes

	// Check our health, not monolith's health
	http.HandleFunc("/api/events/health", healthHandler)
	
	http.HandleFunc("/api/events/movie",  createMovieEvent)
	http.HandleFunc("/api/events/user", createUserEvent)
	http.HandleFunc("/api/events/payment", createPaymentEvent)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	// Initializing Kafka producer and consumer
	kafka_brokers := strings.Fields(os.Getenv("KAFKA_BROKERS"))
	if len (kafka_brokers) == 0 {
		kafka_brokers = append (kafka_brokers, "kafka:9092")
	}

	writer, reader = initKafkaTopic(kafka_brokers, "topicForEverything")
	defer reader.Close()
	defer writer.Close()

	log.Printf("Starting Event Reader loop")
	go ReadkafkaMessages()

	log.Printf("Starting Events service on port %s", port)
	log.Println(http.ListenAndServe(":"+port, nil))
	log.Printf("Stoppin Events service on port %s", port)
	
	time.Sleep(5 *time.Second)
}


func initKafkaTopic (brokers []string, topic string) (*kafka.Writer, *kafka.Reader) {
	wr := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      brokers,
		Topic:        topic,
		RequiredAcks: -1,               // Подтверждение от всех реплик
		MaxAttempts:  10,               //кол-во попыток доставки(по умолчанию всегда 10)
		BatchSize:    100,              // Ограничение на количество сообщений(по дефолту 100)
		WriteTimeout: 10 * time.Second, //время ожидания для записи(по умолчанию 10сек)
		Balancer:     &kafka.RoundRobin{}, //балансировщик.
	})
	

	rdr := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   topic,
		//GroupID: "my-groupID",
	})
	
	return wr, rdr
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"status": true})
}

func ReadkafkaMessages() {
	for {
		select {
        case <-context.Background().Done():
			break
        default:
			if consumeEvent() != nil {
				time.Sleep(1 * time.Second)
			}
        }
	}
	log.Println ("Event Service: Reading loop has finished")
}


func createMovieEvent(w http.ResponseWriter, r *http.Request) {
	
	what := "Movie Event"
	if r.Method == "POST" {
		err := produceEvent (what)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "Method not allowed", http.StatusBadRequest)
        return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func createUserEvent(w http.ResponseWriter, r *http.Request) {
	what := "User Event"
	if r.Method == "POST" {
		err:= produceEvent (what)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "Method not allowed", http.StatusBadRequest)
        return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func createPaymentEvent(w http.ResponseWriter, r *http.Request) {
	what := "Payment Event"
	if r.Method == "POST" {
		err:= produceEvent (what)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "Method not allowed", http.StatusBadRequest)
        return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}


func produceEvent (text string) error {
	err := writer.WriteMessages(context.Background(), kafka.Message{
		Value: []byte(text),
	})
	if err != nil {
		log.Println("Ошибка при отправке:", err)
	} else {
		log.Println("Сообщение отправлено.")
	}
	return err
}

func consumeEvent () error {
	msg, err := reader.ReadMessage(context.Background())
	if err != nil {
		log.Println("Ошибка при получении:", err)
	} else {
		log.Println("Получено сообщение: ", string(msg.Value))
	}
	return err
}
