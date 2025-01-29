package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
)

type Question struct {
	ID       int    `json:"id"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type Quiz struct {
	Title         string     `json:"title"`
	Author        string     `json:"author"`
	Questions     []Question `json:"questions"`
	Flag          string     `json:"flag"`
	TimeoutAmount int        `json:"timeout_amount"`
}

type QuizServer struct {
	Host       string
	Port       string
	QuizObject *Quiz
}

func NewQuizServer(host string, port string, quizObj *Quiz) *QuizServer {
	return &QuizServer{
		Host:       host,
		Port:       port,
		QuizObject: quizObj,
	}
}

var (
	quiz             *Quiz
	logFile          *os.File
	logMutex         sync.Mutex
	questionFileName string = "./question.json"
	logFileName      string = "quiz_attempts.json"
	listeningHost    string = os.Getenv("HOST")
	listeningPort    string = os.Getenv("PORT")
)

func (server *QuizServer) Run() {
	log.Printf("Quiz Server listening on %s:%s...\n", server.Host, server.Port)
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", server.Host, server.Port))
	if err != nil {
		log.Fatalln("[tcp-listening] error occured during listening:", err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalln("[tcp-accept-conn] error occured during accepting new conn:", err)
		}
		log.Printf("received connection from %s\n", conn.RemoteAddr().String())

		go handleRequest(conn, server.QuizObject)
	}
}

func main() {
	questionFile, err := os.ReadFile(questionFileName)
	if err != nil {
		log.Fatalln("[question-init] failed to open questions file:", err)
	}

	err = json.Unmarshal(questionFile, quiz)
	if err != nil {
		log.Fatalln("[question-init] failed to unmarshal question json:", err)
	}

	log.Println(quiz)

	logFile, err = os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("[logging-init] failed to open the log file: %v\n", err)
	}
	defer logFile.Close()

	quizServer := NewQuizServer(listeningHost, listeningPort, quiz)
	quizServer.Run()
}
