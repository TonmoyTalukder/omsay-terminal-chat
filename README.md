# OMSAY - Terminal Chat Server

OMSAY is a terminal-based local network chat application with sound, effects, and elegant terminal UI.

## Features

- Auto-discovery of server via UDP
- Styled CLI interface
- Typing animations
- Sound effects (connect/message)
- Native Windows notifications

## Usage

1. Start the server:
```bash
go run server/main.go
```

2. Start the client:
```bash
go run client/main.go
```

3. Build the client:
```bash
go build -o omsay.exe
```