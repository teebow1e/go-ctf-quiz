package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
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
	Auth       *Authenticator
	semaphore  chan struct{}
}

func NewQuizServer(host string, port string, quizObj *Quiz, auth *Authenticator, maxConnections int) *QuizServer {
	return &QuizServer{
		Host:       host,
		Port:       port,
		QuizObject: quizObj,
		Auth:       auth,
		semaphore:  make(chan struct{}, maxConnections),
	}
}

var (
	logFile       *os.File
	logMutex      sync.Mutex
	logFileName   string
	listeningHost string = os.Getenv("HOST")
	listeningPort string = os.Getenv("PORT")
)

func sanitizeTitle(title string) string {
	safe := strings.ToLower(title)
	safe = strings.ReplaceAll(safe, " ", "_")

	// Remove any characters that aren't alphanumeric, underscore, or hyphen
	reg := regexp.MustCompile("[^a-z0-9_-]+")
	safe = reg.ReplaceAllString(safe, "")

	return safe
}

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
			log.Printf("[tcp-accept-conn] error accepting connection: %v\n", err)
			continue
		}
		log.Printf("received connection from %s\n", conn.RemoteAddr().String())

		// Try to acquire semaphore slot
		select {
		case server.semaphore <- struct{}{}:
			// Successfully acquired slot, handle connection
			go func(c net.Conn) {
				defer func() {
					<-server.semaphore // Release slot when done
				}()
				handleRequest(c, server.QuizObject, server.Auth)
			}(conn)
		default:
			// No slots available, reject connection
			log.Printf("[connection-limit] rejecting connection from %s - max connections reached\n", conn.RemoteAddr().String())
			rejectionMsg := "\n" +
				"╔═══════════════════════════════════════════════════════════════════════════════╗\n" +
				"║                                                                               ║\n" +
				"║                     🚫 SERVER AT MAXIMUM CAPACITY 🚫                          ║\n" +
				"║                                                                               ║\n" +
				"║              The quiz server is currently handling the maximum                ║\n" +
				"║              number of concurrent connections.                                ║\n" +
				"║                                                                               ║\n" +
				"║              Please try again in a few moments.                               ║\n" +
				"║                                                                               ║\n" +
				"╚═══════════════════════════════════════════════════════════════════════════════╝\n\n"
			conn.Write([]byte(rejectionMsg))
			conn.Close()
		}
	}
}

func main() {
	quiz := &Quiz{}

	questionFilePath := "./question.json"
	// questionFilePath := os.Args[1]
	questionFile, err := os.ReadFile(questionFilePath)
	if err != nil {
		log.Fatalln("[question-init] failed to open questions file:", err)
	}

	err = json.Unmarshal(questionFile, quiz)
	if err != nil {
		log.Fatalln("[question-init] failed to unmarshal question json:", err.Error())
	}

	if quiz.Flag == "" {
		log.Panicln("[question-init] No flag defined in question file!")
	}
	if quiz.TimeoutAmount == 0 {
		log.Panicln("[question-init] No timeout defined in question file!")
	}

	log.Printf("Loaded challenge %s by %s, found %v challenges.\n", quiz.Title, quiz.Author, len(quiz.Questions))

	logDir := "./log"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("[logging-init] failed to create log directory: %v\n", err)
	}

	sanitizedTitle := sanitizeTitle(quiz.Title)
	logFileName = fmt.Sprintf("%s/quiz_attempts_%s.json", logDir, sanitizedTitle)
	log.Printf("Log file will be: %s\n", logFileName)

	logFile, err = os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("[logging-init] failed to open the log file: %v\n", err)
	}
	defer logFile.Close()

	// Initialize async logging
	initLogging(logFile)

	// Get max connections from env or use default
	maxConnections := 100
	if maxConnStr := os.Getenv("MAX_CONNECTIONS"); maxConnStr != "" {
		if parsed, err := strconv.Atoi(maxConnStr); err == nil && parsed > 0 {
			maxConnections = parsed
		}
	}
	log.Printf("[config] Maximum concurrent connections: %d\n", maxConnections)

	authenticator, err := newAuthenticator()
	if err != nil {
		log.Fatalf("[auth-init] %v\n", err)
	}

	quizServer := NewQuizServer(listeningHost, listeningPort, quiz, authenticator, maxConnections)
	quizServer.Run()
}
