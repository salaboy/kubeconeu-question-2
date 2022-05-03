package function

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis"
	"net/http"
	"os"
	"time"
)


type Answers struct {
	SessionId string `json:"sessionId"`
	OptionA bool `json:"optionA"`
	OptionB bool `json:"optionB"`
	OptionC bool `json:"optionC"`
	OptionD bool `json:"optionD"`
	RemainingTime int `json:"remainingTime"`
}

type GameScore struct {
	SessionId string
	Time      time.Time
	Level     string
	LevelScore int
}


var redisHost = os.Getenv("REDIS_HOST") // This should include the port which is most of the time 6379
var redisPassword = os.Getenv("REDIS_PASSWORD")

// Handle an HTTP Request.
func Handle(ctx context.Context, res http.ResponseWriter, req *http.Request) {
	client := redis.NewClient(&redis.Options{
		Addr:     redisHost,
		Password: redisPassword,
		DB:       0,
	})

	points := 0
	var answers Answers

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

	res.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(res, string(scoreJson))
}
