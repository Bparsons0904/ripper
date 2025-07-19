# Media Ripper TUI

A Terminal User Interface (TUI) application for ripping CDs, DVDs, and Blu-rays with support for both native tools and containerized execution.

## Project Overview

### Current State
- Working bash scripts for CD ripping (abcde-based)
- Working bash scripts for movie ripping (MakeMKV-based)
- JSON logging and retry logic implemented
- Manual disc detection and duplicate checking

### Goals
- Convert to Go-based TUI using Charm libraries
- Support both native tool execution and Docker containers
- Clean, interactive user experience
- Configurable settings with TOML
- Real-time progress feedback

## Architecture

### Core Components

#### Backend Abstraction
```go
type RipperBackend interface {
    CheckAudioCD(device string) error
    GetCDInfo(device string) (*CDInfo, error)
    ScanMovieTitles(device string) ([]Title, error)
    RipCD(device, outputDir string, progress chan<- Progress) error
    RipMovie(device, title int, outputDir string, progress chan<- Progress) error
}
```

#### Implementation Types
- **NativeBackend**: Executes local tools directly
- **ContainerBackend**: Orchestrates Docker containers

#### Configuration
- TOML-based configuration in `~/.config/media-ripper/config.toml`
- Runtime detection of available tools and drives
- User-configurable paths and preferences

### Technology Stack
- **Language**: Go
- **TUI Framework**: Bubble Tea (Charm)
- **Styling**: Lipgloss (Charm)
- **Configuration**: TOML
- **External Tools**: abcde, MakeMKV, cd-discid
- **Containerization**: Docker (optional)

## Features

### Phase 1: Core TUI
- [ ] Drive detection and selection
- [ ] Configuration management (paths, settings)
- [ ] Basic disc detection (audio CD vs video disc)
- [ ] Settings screen with form inputs
- [ ] Tool availability detection

### Phase 2: Native Execution
- [ ] Break down bash scripts into focused components
- [ ] CD ripping with real-time progress
- [ ] Movie title scanning and selection
- [ ] Movie ripping with progress feedback
- [ ] Error handling and retry logic

### Phase 3: Container Support
- [ ] Docker image with all dependencies
- [ ] Path mapping between host and container
- [ ] Container orchestration from TUI
- [ ] Unified backend switching

### Phase 4: Advanced Features
- [ ] Batch processing
- [ ] Rip queue management
- [ ] Advanced logging and history
- [ ] Profile-based configurations

## User Experience

### Main Flow
1. **Startup**: Detect drives, scan for tools/Docker
2. **Configuration**: Set output directories, preferences
3. **Disc Detection**: Auto-identify disc type and content
4. **Execution Method**: Choose native tools or container
5. **Processing**: Real-time progress with logs
6. **Completion**: Summary and next actions

### TUI Screens

#### Main Menu
```
┌─ Media Ripper ─────────────────────────┐
│                                        │
│ Drive: /dev/sr1 (BD-ROM) [Change]      │
│ Music: /mnt/nas/music [Change]         │
│ Movies: /mnt/nas/movies [Change]       │
│                                        │
│ ┌─ Detected Disc ─┐                   │
│ │ Audio CD        │                   │
│ │ 12 tracks       │                   │
│ │ Unknown Artist  │                   │
│ └─────────────────┘                   │
│                                        │
│ Execute with:                          │
│ ● Native (abcde ✓, cd-discid ✓)       │
│ ○ Container (Docker ✓)                │
│                                        │
│ [Rip Disc] [Settings] [Quit]          │
└────────────────────────────────────────┘
```

#### Settings Screen
```
┌─ Settings ─────────────────────────────┐
│                                        │
│ Paths:                                 │
│ Music Directory:   [/mnt/nas/music   ] │
│ Movies Directory:  [/mnt/nas/movies  ] │
│                                        │
│ Hardware:                              │
│ Optical Drive:     [/dev/sr1         ] │
│                                        │
│ Behavior:                              │
│ Retry Count:       [3                ] │
│ Initial Wait:      [10s              ] │
│ Auto-eject:        [✓] Yes  [ ] No     │
│                                        │
│ Execution:                             │
│ Preferred Backend: [Native           ] │
│                                        │
│ [Save] [Cancel] [Reset to Defaults]   │
└────────────────────────────────────────┘
```

#### Progress Screen
```
┌─ Ripping: Unknown Album ───────────────┐
│                                        │
│ Track 3 of 12: "Song Title"            │
│ ████████████████████░░░░░░░░ 65%       │
│                                        │
│ Status: Encoding FLAC...               │
│ Elapsed: 00:03:45                     │
│ Remaining: ~00:02:30                   │
│                                        │
│ Recent Activity:                       │
│ • Track 2 completed successfully       │
│ • Starting track 3 extraction          │
│ • CDDB lookup successful               │
│                                        │
│ [Cancel] [View Full Log]               │
└────────────────────────────────────────┘
```

## Configuration Structure

### TOML Configuration
```toml
[drives]
selected = "/dev/sr1"
available = ["/dev/sr0", "/dev/sr1", "/dev/cdrom"]

[paths]
music = "/mnt/nas/media/music"
movies = "/mnt/nas/media/movies"
config = "~/.config/media-ripper"

[execution]
preferred_backend = "native"
retry_count = 3
retry_delay = 5
initial_wait = 10
auto_eject = true

[tools]
abcde_path = "/usr/bin/abcde"
makemkv_path = "/usr/bin/makemkvcon"
cd_discid_path = "/usr/bin/cd-discid"

[logging]
level = "info"
format = "json"
file = "~/.cache/media-ripper/ripper.log"

[container]
image = "media-ripper:latest"
pull_policy = "if_not_present"
```

## File Structure

```
media-ripper/
├── cmd/
│   └── media-ripper/
│       └── main.go
├── internal/
│   ├── config/
│   │   ├── config.go
│   │   └── toml.go
│   ├── backends/
│   │   ├── interface.go
│   │   ├── native.go
│   │   └── container.go
│   ├── tui/
│   │   ├── app.go
│   │   ├── main_menu.go
│   │   ├── settings.go
│   │   ├── progress.go
│   │   └── styles.go
│   ├── scripts/
│   │   ├── cd/
│   │   └── movie/
│   └── types/
│       ├── disc.go
│       ├── progress.go
│       └── config.go
├── scripts/
│   ├── cd/
│   │   ├── check-audio-cd.sh
│   │   ├── get-cd-info.sh
│   │   ├── check-existing-album.sh
│   │   └── rip-cd.sh
│   └── movie/
│       ├── detect-disc.sh
│       ├── scan-titles.sh
│       └── rip-title.sh
├── docker/
│   ├── Dockerfile
│   └── entrypoint.sh
├── configs/
│   └── default.toml
├── go.mod
├── go.sum
└── README.md
```

## Dependencies

### Go Modules
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Styling
- `github.com/charmbracelet/bubbles` - UI components
- `github.com/pelletier/go-toml/v2` - TOML parsing
- `github.com/spf13/afero` - Filesystem abstraction

### External Tools
- **abcde** - CD ripping
- **cd-discid** - CD identification
- **MakeMKV** - DVD/Blu-ray ripping
- **Docker** - Container support (optional)

### Container Dependencies
- Ubuntu/Debian base image
- All required tools pre-installed
- Proper device access permissions

## Development Phases

### Phase 1: Foundation (Week 1-2)
- Set up Go project structure
- Implement basic TUI with Bubble Tea
- Drive detection and configuration management
- Settings screen with form handling

### Phase 2: Script Integration (Week 3-4)
- Break down existing bash scripts
- Implement native backend with subprocess execution
- Add progress tracking and real-time feedback
- Error handling and logging

### Phase 3: Container Support (Week 5-6)
- Create Docker image with dependencies
- Implement container backend
- Path mapping and volume management
- Backend selection UI

### Phase 4: Polish & Testing (Week 7-8)
- Comprehensive error handling
- Performance optimization
- Documentation and examples
- Testing on different distributions

## Success Criteria

### Functional Requirements
- [ ] Successfully rip CDs with same quality as current bash scripts
- [ ] Successfully rip DVDs/Blu-rays with title selection
- [ ] Work on both PopOS (primary) and macOS
- [ ] Handle drive detection automatically
- [ ] Provide clear progress feedback
- [ ] Support both native and container execution

### User Experience Requirements
- [ ] Intuitive TUI navigation
- [ ] Clear error messages and recovery options
- [ ] Reasonable performance (no noticeable lag)
- [ ] Configurable without editing files
- [ ] Maintain existing output structure and quality

### Technical Requirements
- [ ] Clean, maintainable Go code
- [ ] Proper error handling and logging
- [ ] Testable architecture with interfaces
- [ ] Container image under 500MB
- [ ] Memory usage under 50MB during idle

## Future Considerations

### Potential Enhancements
- Web UI for remote operation
- Music metadata enhancement integration
- Automatic quality verification
- Cloud storage integration
- Mobile app for monitoring
- Integration with media servers (Plex, Jellyfin)

### Platform Expansion
- Windows support (if there's demand)
- ARM architecture support
- Raspberry Pi deployment
- NAS appliance integration

## Questions & Decisions

### Open Questions
1. Should we support multiple concurrent rips?
2. How to handle MakeMKV licensing in containers?
3. Configuration migration strategy for existing bash script users?
4. Logging format - keep JSON or move to structured Go logging?

### Design Decisions
- ✅ TUI over CLI flags for better UX
- ✅ TOML over SQLite for configuration simplicity
- ✅ Backend abstraction for native/container support
- ✅ Subprocess execution over native Go libraries
- ✅ Focused scripts over monolithic functions

---

*This document will evolve as we develop and discover new requirements or constraints.*