# go-ctf-quiz

A TCP server to facilitate the process of making checker for multi-question CTF challenge.

Watch go-ctf-quiz in action:

[![asciicast](https://asciinema.org/a/rNXFX9ivQF7xTaZ0vv9ok3bp7.svg)](https://asciinema.org/a/rNXFX9ivQF7xTaZ0vv9ok3bp7)

# Features
- From JSON -> quiz server, ready to publish!
- Authentication through CTFd access token
- Log each quiz attempt into a single JSON file

# Getting started
In order to get started, prepare a question file named `question.json`. The file should look like this:
```json
{
    "title": "Sample Quiz",
    "created_at": 123,
    "author": "teebow1e",
    "flag": "FLAG{sample_flag}",
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

When everything is done, just `docker compose up -d`!

# Logging
- Logging should happen automatically in the container. Just copy the file out with:
```bash
docker cp go-ctf-quiz-quiz-app:/app/quiz_attempts.json ./quiz_attempts_from_docker.json
```
- Filtering with JQ (TODO)

# TODO
- ANSI support
