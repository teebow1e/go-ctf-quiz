package main

import (
	"bufio"
	"encoding/json"
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
	log.Printf("Received token: %s", string(message))
	identity, err := verifyIdentity(strings.TrimSpace(string(message)))
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
	// Question part goes here
	askQuestions(conn, reader, questionPack.Questions)
	showFlag(conn, questionPack.Flag)
	conn.Close()
}

func askQuestions(conn net.Conn, reader *bufio.Reader, qpack []Question) {
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
			conn.Close()
			return
		case answer := <-answerCh:
			if strings.TrimSpace(answer) == question.Answer { // single question correct -> continue
				conn.Write([]byte("Correct!\n"))
			} else { // single question wrong -> should close conn here
				conn.Write([]byte("You're wrong.. sadly. Better luck next time!\n"))
				conn.Close()
				return
			}
		}
	}
}

func showFlag(conn net.Conn, flag string) {
	conn.Write([]byte("Congratulations! You conquered all of my challenges! Here is the flag.\n"))
	time.Sleep(2 * time.Second)
	conn.Write([]byte(flag + "\n"))
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
