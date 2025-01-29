package main

import (
	"encoding/json"
	"log"
)

type LogQuestion struct {
	QuestionId int    `json:"question_id"`
	Question   string `json:"question"`
	Answer     string `json:"answer"`
	Status     string `json:"status"`
}

type LogEntry struct {
	UserToken   string        `json:"user_token"`
	Username    string        `json:"username"`
	Timestamp   string        `json:"timestamp"`
	QuizAttempt []LogQuestion `json:"quiz_attempt"`
	Status      string        `json:"status"`
}

func logAttempt(logEntry LogEntry) {
	data, err := json.Marshal(logEntry)
	if err != nil {
		log.Printf("[logAttempt] failed to serialize log entry: %v\n", err)
		return
	}

	logMutex.Lock()
	defer logMutex.Unlock()

	_, err = logFile.Write(append(data, '\n'))
	if err != nil {
		log.Printf("[logAttempt] failed to write log entry: %v\n", err)
	}
}
