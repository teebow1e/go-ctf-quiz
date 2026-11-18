# go-ctf-quiz

A TCP server to facilitate the process of making checker for multi-question CTF challenge.

Watch go-ctf-quiz in action:

[![asciicast](https://asciinema.org/a/rNXFX9ivQF7xTaZ0vv9ok3bp7.svg)](https://asciinema.org/a/rNXFX9ivQF7xTaZ0vv9ok3bp7)

# Features
- From JSON -> quiz server, ready to publish!
- Authentication through CTFd access token or rCTF team token
- Log each quiz attempt into a single JSON file

# Getting started
In order to get started, prepare a question file named `question.json`. The file should look like this:
```json
{
    "title": "Sample Quiz",
    "created_at": 123,
    "author": "teebow1e",
    "flag": "FLAG{sample_flag}",
    "timeout_amount": 30,
    "questions": [
        {
            "id": 1,
            "question": "What is question A?",
            "answer": "Option A"
        },
        {
            "id": 2,
            "question": "What is question B?",
            "answer": "Option B"
        },
        {
            "id": 3,
            "question": "What is question C?",
            "answer": "Option C"
        }
    ]
}
```
After that, edit necessary information in the `.env` file, including CTFd link, listening host/port.

# Sample docker-compose
```yaml
services:
  part1:
    image: ghcr.io/teebow1e/tcpquiz:latest
    container_name: part1
    ports:
      - "127.0.0.1:7002:1337"
    volumes:
      - ./question.json:/app/question.json
      - ../logs:/app/log
    environment:
      - HOST=0.0.0.0
      - PORT=1337
      - CTFD_URL=https://example.com
      - MAX_CONNECTIONS=100
      # Only when you want CTFd authentication
      - STRICT_MODE=1 
    restart: unless-stopped
```

When everything is done, just `docker compose up -d`!

# Authentication
Set `STRICT_MODE=1` to require players to log in. `AUTH_PROVIDER` selects the platform:

| `AUTH_PROVIDER` | Required variable | Players enter |
|---|---|---|
| `ctfd` (default) | `CTFD_URL` | CTFd access token |
| `rctf` | `RCTF_URL` | rCTF team token or invite link (`https://ctf.example.com/login?token=...`) |

For rCTF, replace the CTFd variables in the compose file with:
```yaml
      - STRICT_MODE=1
      - AUTH_PROVIDER=rctf
      - RCTF_URL=https://ctf.example.com
```

# Logging
All quiz attempts are logged to `./log/quiz_attempts_<sanitized_title>.json`. The log directory is automatically created by the application.

When using Docker Compose, the logs are persisted on the host machine through volume mounting:
```yaml
volumes:
  - ./question.json:/app/question.json
  - ./log:/app/log
```

Each log entry contains detailed information about the attempt including timestamp, user token, answers, and whether they completed the quiz successfully. With rCTF, the `user_token` field holds the team ID, because team tokens are long-lived credentials and are never written to the logs.