package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

type Question struct {
	ID       int    `json:"id"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type Quiz struct {
	Title     string     `json:"title"`
	Author    string     `json:"author"`
	Questions []Question `json:"questions"`
	Flag      string     `json:"flag"`
}

type Config struct {
	Host string
	Port string
}

type Server struct {
	Host string
	Port string
}

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

func NewServer(config *Config) *Server {
	return &Server{
		Host: config.Host,
		Port: config.Port,
	}
}

func (server *Server) Run(questionPack *Quiz) {
	log.Printf("Quiz Server listening on %s:%s...\n", server.Host, server.Port)
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", server.Host, server.Port))
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go handleRequest(conn, questionPack)
	}
}

func handleRequest(conn net.Conn, questionPack *Quiz) {
	reader := bufio.NewReader(conn)
	conn.Write([]byte("Please enter your token: "))
	message, err := reader.ReadString('\n')
	if err != nil {
		conn.Close()
		return
	}

	token := strings.TrimSpace(string(message))

	log.Printf("Received token: %s", token)
	identity, err := verifyIdentity(token)
	if err != nil {
		conn.Write([]byte("Failed to verify. Please double-check your token.\n"))
		conn.Close()
		return
	}
	conn.Write([]byte(fmt.Sprintf("Verification finished. Welcome %s.\n", identity)))
	conn.Write([]byte("In order to prove that you are worthy to receive the flag, please answer the following questions.\n"))
	conn.Write([]byte("The timeout amount is 60 seconds.\n\n"))
	conn.Write([]byte("Press any key to continue..."))
	_, err = reader.ReadString('\n')
	if err != nil {
		conn.Close()
		return
	}

	// Logging starts here
	logEntry := &LogEntry{
		UserToken:   token,
		Username:    identity,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		QuizAttempt: []LogQuestion{},
	}

	timeFormat := "2006-01-02_150405"
	logFileName := fmt.Sprintf("./%s_%s.json", time.Now().Format(timeFormat), identity)

	// Question part goes here
	err = askQuestions(conn, reader, questionPack.Questions, logEntry)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "out of time") {
			logEntry.Status = "time_out"
		} else if strings.Contains(errMsg, "wrongly") {
			logEntry.Status = "wrong_answer"
		}
		conn.Close()
		logAttempt(*logEntry, logFileName)
		return
	}
	showFlag(conn, questionPack.Flag)
	logEntry.Status = "completed"
	conn.Close()
	logAttempt(*logEntry, logFileName)
}

func askQuestions(conn net.Conn, reader *bufio.Reader, qpack []Question, logEntry *LogEntry) error {
	timer := time.NewTimer(time.Duration(30) * time.Second)
	defer timer.Stop()

	for _, question := range qpack {
		conn.Write([]byte(question.Question + "\n"))
		answerCh := make(chan string, 1)

		go func() { // unsafe here: no error handling :D
			userAnswer, _ := reader.ReadString('\n')
			answerCh <- userAnswer
		}()

		select {
		case <-timer.C: // time's up -> should close conn here
			// conn.Write([]byte("Time's up!\n"))
			return errors.New("user ran out of time")
		case answer := <-answerCh:
			if strings.TrimSpace(answer) == question.Answer { // single question correct -> continue
				conn.Write([]byte("Correct!\n"))
				logEntry.QuizAttempt = append(logEntry.QuizAttempt, LogQuestion{
					QuestionId: question.ID,
					Question:   question.Question,
					Answer:     strings.TrimSpace(answer),
					Status:     "correct",
				})
			} else { // single question wrong -> should close conn here
				conn.Write([]byte("You're wrong.. sadly. Better luck next time!\n"))
				logEntry.QuizAttempt = append(logEntry.QuizAttempt, LogQuestion{
					QuestionId: question.ID,
					Question:   question.Question,
					Answer:     strings.TrimSpace(answer),
					Status:     "incorrect",
				})
				return errors.New("user answered wrongly")
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

func logAttempt(logEntry LogEntry, logFile string) {
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("failed to open log file: %v\n", err)
		return
	}
	defer file.Close()

	data, err := json.MarshalIndent(logEntry, "", "  ")
	if err != nil {
		log.Printf("failed to serialize log entry: %v\n", err)
		return
	}

	_, err = file.Write(append(data, '\n'))
	if err != nil {
		log.Printf("failed to write log entry: %v\n", err)
	}
}

func main() {
	quiz := &Quiz{}
	questionFilename := "./question.json"

	data, err := os.ReadFile(questionFilename)
	if err != nil {
		log.Fatalln("failed to open questions file")
	}

	err = json.Unmarshal(data, quiz)
	if err != nil {
		log.Fatalln("failed to unmarshal question json", err)
	}

	log.Println(quiz)

	config := &Config{
		Host: "127.0.0.1",
		Port: "9998",
	}

	quizServer := NewServer(config)
	quizServer.Run(quiz)
}
