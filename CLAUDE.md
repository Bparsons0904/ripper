# CLAUDE.md

This file provides guidance to Claude Code when working with the Media Ripper TUI project.

## Project Overview

This is a Terminal User Interface (TUI) application for ripping CDs, DVDs, and Blu-rays built with Go and the Charm library suite. The project converts existing bash scripts into a modern, interactive TUI with support for both native tool execution and containerized workflows.

## Architecture

The project follows a clean architecture pattern:

- `cmd/media-ripper/` - Main application entry point
- `internal/` - Internal packages (config, backends, TUI, types)
- `scripts/` - Legacy bash scripts to be integrated
- `docker/` - Container support files

## Key Technologies

- **Language**: Go 1.24
- **TUI Framework**: Bubble Tea (Charm)
- **Styling**: Lipgloss (Charm) 
- **External Tools**: abcde, MakeMKV, cd-discid
- **Configuration**: TOML

## Development Guidelines

### Command Usage
- Never use `cd` directly (conflicts with zoxide alias)
- Use absolute paths: `/home/bobparsons/Development/ripper/`
- Use `\cd` if directory changes are required

### Code Standards
- Follow Go conventions with camelCase file names
- Use interfaces for backend abstraction
- Implement proper error handling and logging
- No business logic in tests

## Common Commands

```bash
# Development with hot reload (recommended for TUI)
./dev.sh

# Manual run
go run cmd/media-ripper/main.go

# Build binary
go build -o media-ripper cmd/media-ripper/main.go

# Air (has TUI issues - use dev.sh instead)
air

# Test
go test ./...

# Dependencies
go mod tidy
```

## Project Plan Reference

See `media_ripper_project.md` for the complete project specification, including:
- Detailed architecture overview
- User experience mockups
- Development phases
- Success criteria
- File structure

## Current Status

- ✅ Go module initialized
- ✅ Bubble Tea dependency added
- 🚧 Basic TUI implementation
- ⏸️ Script integration
- ⏸️ Container support