package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

func handleRequest(conn net.Conn, quizObj *Quiz) {
	var identity, token string
	defer conn.Close()

	// Set initial read timeout for token input (60 seconds)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	reader := bufio.NewReader(conn)

	conn.Write([]byte(banner("BKSEC CTF Quiz System")))

	if os.Getenv("STRICT_MODE") == "1" {
		conn.Write([]byte(prompt("Please enter your CTFd access token: ")))
		message, err := reader.ReadString('\n')
		if err != nil {
			log.Println("[token stage] failed to read user input: ", err)
			return
		}

		token = strings.TrimSpace(string(message))

		log.Printf("Received token: %s\n", token)
		conn.Write([]byte(info("Verifying your identity...") + "\n"))
		identity, err = verifyIdentity(token)
		if err != nil {
			conn.Write([]byte("\n" + failure("Authentication failed!") + "\n"))
			conn.Write([]byte(warning("Please check your token and try again.") + "\n"))
			return
		}
	} else {
		identity = "guest"
		token = "guest-token"
	}

	// Reset deadline for the rest of the interaction
	conn.SetReadDeadline(time.Time{})

	conn.Write([]byte(success(fmt.Sprintf("Welcome, %s!", identity)) + "\n"))
	conn.Write([]byte(divider() + "\n\n"))
	conn.Write([]byte("You are playing challenge: " + colorize(ColorBoldYellow, quizObj.Title) + " by " + colorize(ColorBoldYellow, quizObj.Author) + "\n\n"))
	conn.Write([]byte(colorize(ColorWhite, "You must prove your knowledge by answering a series of questions.\n")))
	conn.Write([]byte(colorize(ColorWhite, "Answer them all correctly to earn the flag.") + "\n\n"))
	conn.Write([]byte(warning(fmt.Sprintf("Each question has a %d second time limit.", quizObj.TimeoutAmount)) + "\n"))
	conn.Write([]byte(colorize(ColorBoldRed, "One wrong answer and it's game over!") + "\n\n"))
	conn.Write([]byte(divider() + "\n\n"))
	conn.Write([]byte(prompt("Press Enter when you're ready to begin...")))

	// Set timeout for "press any key" prompt (30 seconds)
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	_, err := reader.ReadString('\n')
	if err != nil {
		log.Println("[press-any-key] failed to capture user input: ", err)
		return
	}

	// Clear deadline for quiz questions (handled by askQuestions timeout logic)
	conn.SetReadDeadline(time.Time{})

	logEntry := &LogEntry{
		UserToken:         token,
		Username:          identity,
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
		QuizAttempt:       []LogQuestion{},
		QuestionsAnswered: 0,
		RetryCount:        getAndIncrementRetryCount(token),
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

	// Set questions_answered to the number of questions attempted
	logEntry.QuestionsAnswered = len(logEntry.QuizAttempt)

	if logEntry.Status == "completed" {
		showFlag(conn, quizObj.Flag)
	}
	logAttempt(*logEntry)
}

func askQuestions(conn net.Conn, reader *bufio.Reader, questionPack []Question, logEntry *LogEntry, timeout int) error {
	for _, question := range questionPack {
		timer := time.NewTimer(time.Duration(timeout) * time.Second)

		// Display question with nice formatting
		conn.Write([]byte("\n" + colorize(ColorBoldCyan, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━") + "\n"))
		conn.Write([]byte(colorize(ColorBoldWhite, "❓ Question:") + "\n"))
		conn.Write([]byte(colorize(ColorWhite, "   "+question.Question) + "\n\n"))
		conn.Write([]byte(prompt("Your answer: ")))

		answerCh := make(chan string, 1)
		errCh := make(chan error, 1)

		go func() {
			userAnswer, err := reader.ReadString('\n')
			if err != nil {
				log.Println("[goroutine-capture-answer] failed to capture user input: ", err)
				errCh <- err
				return
			}
			answerCh <- userAnswer
		}()

		var answer string
		select {
		case <-timer.C:
			// Write timeout message BEFORE closing connection
			conn.Write([]byte("\n\n" + failure("Time's up!") + "\n"))
			conn.Write([]byte(colorize(ColorBoldRed, "⏰ You ran out of time. Better luck next time!\n\n")))
			// Close connection to interrupt the blocked goroutine
			conn.Close()
			return errors.New("user ran out of time")
		case err := <-errCh:
			timer.Stop()
			return fmt.Errorf("error reading answer: %w", err)
		case answer = <-answerCh:
			// Properly clean up timer
			if !timer.Stop() {
				<-timer.C
			}
		}

		if strings.TrimSpace(answer) == question.Answer {
			conn.Write([]byte("\n" + success("Correct! Moving on...") + "\n"))
			logEntry.QuizAttempt = append(logEntry.QuizAttempt, LogQuestion{
				QuestionId: question.ID,
				Question:   question.Question,
				Answer:     strings.TrimSpace(answer),
				Status:     "correct",
			})
		} else {
			conn.Write([]byte("\n" + failure("Incorrect! You better try harder...") + "\n"))
			logEntry.QuizAttempt = append(logEntry.QuizAttempt, LogQuestion{
				QuestionId: question.ID,
				Question:   question.Question,
				Answer:     strings.TrimSpace(answer),
				Status:     "incorrect",
			})
			return errors.New("received wrong answer from user")
		}
	}
	return nil
}
