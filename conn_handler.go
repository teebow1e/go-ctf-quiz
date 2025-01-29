package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

func handleRequest(conn net.Conn, quizObj *Quiz) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	conn.Write([]byte("Please enter your token: "))
	message, err := reader.ReadString('\n')
	if err != nil {
		log.Println("[token stage] failed to read user input: ", err)
		return
	}

	token := strings.TrimSpace(string(message))

	log.Printf("Received token: %s\n", token)
	identity, err := verifyIdentity(token)
	if err != nil {
		conn.Write([]byte("Failed to verify. Please double-check your token.\n"))
		return
	}
	conn.Write([]byte(fmt.Sprintf("Verification finished. Welcome %s.\n", identity)))
	conn.Write([]byte(fmt.Sprintf("You are playing challenge %s.", quizObj.Title)))
	conn.Write([]byte("In order to prove that you are worthy to receive the flag, please answer the following questions.\n"))
	conn.Write([]byte("The timeout amount is 60 seconds.\n\n"))
	conn.Write([]byte("Press any key to continue..."))
	_, err = reader.ReadString('\n')
	if err != nil {
		log.Println("[press-any-key] failed to capture user input: ", err)
		return
	}

	logEntry := &LogEntry{
		UserToken:   token,
		Username:    identity,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		QuizAttempt: []LogQuestion{},
	}

	err = askQuestions(conn, reader, quizObj.Questions, logEntry, quizObj.TimeoutAmount)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "out of time") {
			logEntry.Status = "time_out"
		} else if strings.Contains(errMsg, "wrong answer") {
			logEntry.Status = "wrong_answer"
		} else {
			logEntry.Status = "server_runtime_error"
		}
	} else {
		logEntry.Status = "completed"
	}

	if logEntry.Status == "completed" {
		showFlag(conn, quizObj.Flag)
	}
	logAttempt(*logEntry)
}

func askQuestions(conn net.Conn, reader *bufio.Reader, questionPack []Question, logEntry *LogEntry, timeout int) error {
	for _, question := range questionPack {
		timer := time.NewTimer(time.Duration(timeout) * time.Second)
		conn.Write([]byte(question.Question + "\n"))
		answerCh := make(chan string, 1)

		go func() {
			userAnswer, err := reader.ReadString('\n')
			if err != nil {
				log.Println("[goroutine-capture-answer] failed to capture user input: ", err)
				answerCh <- "ERR_OCCURED"
				return
			}
			answerCh <- userAnswer
		}()

		select {
		case <-timer.C:
			return errors.New("user ran out of time")
		case answer := <-answerCh:
			timer.Stop()
			if strings.TrimSpace(answer) == question.Answer {
				conn.Write([]byte("Correct!\n"))
				logEntry.QuizAttempt = append(logEntry.QuizAttempt, LogQuestion{
					QuestionId: question.ID,
					Question:   question.Question,
					Answer:     strings.TrimSpace(answer),
					Status:     "correct",
				})
			} else {
				conn.Write([]byte("You're wrong.. sadly. Better luck next time!\n"))
				logEntry.QuizAttempt = append(logEntry.QuizAttempt, LogQuestion{
					QuestionId: question.ID,
					Question:   question.Question,
					Answer:     strings.TrimSpace(answer),
					Status:     "incorrect",
				})
				return errors.New("received wrong answer from user")
			}
		}
	}
	return nil
}

func showFlag(conn net.Conn, flag string) {
	conn.Write([]byte("Congratulations! You conquered all of my challenges! Here is the flag.\n"))
	time.Sleep(2 * time.Second)
	conn.Write([]byte(flag + "\n"))
}
