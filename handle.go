package function

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/go-redis/redis"
	"github.com/google/uuid"
	"log"
	"net/http"
	"os"
	"time"
)


type Answers struct {
	Player string `json:"player"`
	SessionId string `json:"sessionId"`
	OptionA bool `json:"optionA"`
	OptionB bool `json:"optionB"`
	OptionC bool `json:"optionC"`
	OptionD bool `json:"optionD"`
	RemainingTime int `json:"remainingTime"`
}

type GameScore struct {
	Player string
	SessionId string
	Time      time.Time
	Level     string
	LevelScore int
}


var redisHost = os.Getenv("REDIS_HOST") // This should include the port which is most of the time 6379
var redisPassword = os.Getenv("REDIS_PASSWORD")
var redisTLSEnabled = os.Getenv("REDIS_TLS")
var gameEventingEnabled = os.Getenv("GAME_EVENTING_ENABLED")
var sink = os.Getenv("GAME_EVENTING_BROKER_URI")
var cloudEventsEnabled bool = false
var redisTLSEnabledFlag = false

// Handle an HTTP Request.
func Handle(ctx context.Context, res http.ResponseWriter, req *http.Request) {
	if redisTLSEnabled != "" && redisTLSEnabled != "false" {
		redisTLSEnabledFlag = true
	}
	var client *redis.Client

	if !redisTLSEnabledFlag {
		client = redis.NewClient(&redis.Options{
			Addr:     redisHost,
			Password: redisPassword,
			DB:       0,
		})
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:     redisHost,
			Password: redisPassword,
			DB:       0,
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		})
	}

	points := 0
	var answers Answers

	if gameEventingEnabled != "" && gameEventingEnabled != "false"{
		cloudEventsEnabled = true
	}

	// Try to decode the request body into the struct. If there is an error,
	// respond to the client with the error message and a 400 status code.
	err := json.NewDecoder(req.Body).Decode(&answers)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	if answers.OptionA == true {
		points =  0
	}
	if answers.OptionB == true {
		points= 3 // KubeCon/KnativeCon special bonus!
	}
	if answers.OptionC == true {
		points = 5
	}
	if answers.OptionD == true {
		points = 0
	}

	points += answers.RemainingTime

	score := GameScore {
		Player:  answers.Player,
		SessionId: answers.SessionId,
		Level: "kubeconeu-question-2",
		LevelScore: points,
		Time: time.Now(),
	}
	scoreJson, err := json.Marshal(score)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	err = client.RPush("score-" + answers.SessionId, string(scoreJson)).Err()
	// if there has been an error setting the value
	// handle the error
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	if cloudEventsEnabled {
		emitCloudEvent(scoreJson)
	}

	res.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(res, string(scoreJson))
}

func emitCloudEvent(gs []byte) error {
	c, err := cloudevents.NewClientHTTP()
	if err != nil {
		log.Fatalf("failed to create client, %v", err)
	}

	// Create an Event.
	event := cloudevents.NewEvent()
	newUUID, _ := uuid.NewUUID()
	event.SetID(newUUID.String())
	event.SetTime(time.Now())
	event.SetSource("kubeconeu-question-2")
	event.SetType("GameScoreEvent")
	event.SetData(cloudevents.ApplicationJSON, gs)

	log.Printf("Emitting an Event: %s to SINK: %s", event, sink)

	// Set a target.
	ctx := cloudevents.ContextWithTarget(context.Background(), sink)

	// Send that Event.
	result := c.Send(ctx, event)
	if result != nil {
		log.Printf("Resutl: %s", result)
		if cloudevents.IsUndelivered(result) {
			log.Printf("failed to send, %v", result)
			return result
		}
	}
	return nil
}