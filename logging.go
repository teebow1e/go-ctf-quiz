package main

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

type LogQuestion struct {
	QuestionId int    `json:"question_id"`
	Question   string `json:"question"`
	Answer     string `json:"answer"`
	Status     string `json:"status"`
}

type LogEntry struct {
	UserToken         string        `json:"user_token"`
	Username          string        `json:"username"`
	Timestamp         string        `json:"timestamp"`
	QuizAttempt       []LogQuestion `json:"quiz_attempt"`
	Status            string        `json:"status"`
	QuestionsAnswered int           `json:"questions_answered"`
	RetryCount        int           `json:"retry_count"`
}

var (
	logChannel   chan LogEntry
	retryTracker = make(map[string]int) // tracks retry count per user token
	retryMutex   sync.RWMutex
)

// getAndIncrementRetryCount returns the current retry count and increments it
func getAndIncrementRetryCount(userToken string) int {
	retryMutex.Lock()
	defer retryMutex.Unlock()

	count := retryTracker[userToken]
	retryTracker[userToken] = count + 1
	return count + 1 // return 1-indexed count (1st attempt = 1, 2nd attempt = 2, etc.)
}

// initLogging starts the async logging goroutine
func initLogging(logFile *os.File) {
	logChannel = make(chan LogEntry, 1000) // Buffer up to 1000 log entries

	go func() {
		for entry := range logChannel {
			data, err := json.Marshal(entry)
			if err != nil {
				log.Printf("[logAttempt] failed to serialize log entry: %v\n", err)
				continue
			}

			_, err = logFile.Write(append(data, '\n'))
			if err != nil {
				log.Printf("[logAttempt] failed to write log entry: %v\n", err)
			}
		}
	}()
}

// logAttempt sends a log entry to the async logger
func logAttempt(logEntry LogEntry) {
	select {
	case logChannel <- logEntry:
		// Successfully queued
	default:
		// Channel full, log synchronously as fallback
		log.Printf("[logAttempt] log channel full, logging synchronously\n")
		data, err := json.Marshal(logEntry)
		if err != nil {
			log.Printf("[logAttempt] failed to serialize log entry: %v\n", err)
			return
		}

		logMutex.Lock()
		_, err = logFile.Write(append(data, '\n'))
		logMutex.Unlock()

		if err != nil {
			log.Printf("[logAttempt] failed to write log entry: %v\n", err)
		}
	}
}
