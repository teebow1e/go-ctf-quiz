# CTF Challenge Structure

This directory contains individual CTF quiz challenges, each with its own Docker Compose configuration.

## Directory Structure

```
challenges/
├── README.md
├── example-web-part1/
│   ├── compose.yml
│   └── question.json
└── [your-challenge]/
    ├── compose.yml
    └── question.json
```

## How to Use

### 1. Create a New Challenge

```bash
# Create new challenge directory
mkdir challenges/your-challenge-name

# Copy template files
cp challenges/example-crypto-part1/compose.yml challenges/your-challenge-name/
cp challenges/example-crypto-part1/question.json challenges/your-challenge-name/

# Edit the files to match your challenge
```

### 2. Configure compose.yml

Edit the `compose.yml` in your challenge directory:

```yaml
services:
  your-challenge-name:
    image: teebow1e/go-ctf-quiz
    container_name: ctf-your-challenge-name  # Make this unique
    ports:
      - "7XXX:1337"  # Use unique external port
    volumes:
      - ./question.json:/app/question.json
      - ../../logs:/app  # All challenges share logs directory
    environment:
      - HOST=0.0.0.0
      - PORT=1337
      - CTFD_URL=https://your-ctfd-instance.com
      - MAX_CONNECTIONS=100
    restart: unless-stopped
```

**Important:**
- Change `container_name` to be unique for each challenge
- Use a unique external port (7001, 7002, 7003, etc.)
- All challenges mount `../../logs:/app` - logs are separated by filename, not directory

### 3. Configure question.json

Edit your `question.json`:

```json
{
    "title": "Your Challenge Title",
    "author": "your_name",
    "flag": "FLAG{your_flag_here}",
    "timeout_amount": 60,
    "questions": [
        {
            "id": 1,
            "question": "Your question here?",
            "answer": "Expected answer"
        }
    ]
}
```

**The `title` field is critical** - it generates the log filename as `quiz_attempts_<sanitized_title>.json`

### 4. Start Your Challenge

```bash
# Navigate to your challenge directory
cd challenges/your-challenge-name

# Start the challenge
docker compose up -d

# View logs
docker compose logs -f

# Stop the challenge
docker compose down
```

### 5. Start Multiple Challenges

```bash
# From project root
cd challenges/example-crypto-part1 && docker compose up -d && cd ../..
cd challenges/example-web-part1 && docker compose up -d && cd ../..

# Or use a helper script (create one if needed)
```

## Log Files

All challenges write logs to `../../logs/` directory with filenames based on challenge title:

- `logs/quiz_attempts_crypto_challenge_part_1.json`
- `logs/quiz_attempts_web_exploitation_part_1.json`

Since filenames are different, all challenges can safely share the same log directory.

## Analyzing Logs

Use the analyze script to parse all logs recursively:

```bash
# Analyze all logs in logs/ directory
uv run analyze_logs.py logs/

# Or from project root
uv run analyze_logs.py .
```

The script will automatically find all `quiz_attempts_*.json` files and generate combined analytics.

## Port Allocation

Suggested port allocation scheme:
- 7000-7099: Crypto challenges
- 7100-7199: Web challenges
- 7200-7299: Pwn challenges
- 7300-7399: Reverse engineering
- 7400-7499: Forensics
- 7500+: Misc challenges

## Tips

1. **Unique Challenge Titles**: Each challenge should have a unique title to avoid log file conflicts
2. **Port Conflicts**: Make sure each challenge uses a different external port
3. **Container Names**: Use unique container names to avoid Docker conflicts
4. **Testing**: Test challenges individually before deploying multiple simultaneously
5. **Resource Limits**: Consider adding memory/CPU limits in compose.yml for production
