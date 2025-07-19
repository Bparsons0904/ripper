package ripper

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/Bparsons0904/ripper/internal/config"
)

// CDInfo represents information about a CD
type CDInfo struct {
	Artist     string
	Album      string
	Year       string
	Genre      string
	TrackCount int
	Tracks     []TrackInfo
	DiscID     string
	CDDBDiscID string // CDDB format disc ID
	Offsets    []int  // Track offsets for CDDB/MusicBrainz
}

// TrackInfo represents information about a single track
type TrackInfo struct {
	Number   int
	Title    string
	Artist   string
	Duration string
}

// ProgressInfo represents ripping progress
type ProgressInfo struct {
	CurrentTrack int
	TotalTracks  int
	TrackName    string
	Progress     int
	Status       string
	Error        error
}

// CDRipper handles CD ripping operations
type CDRipper struct {
	config      *config.Config
	progressCh  chan ProgressInfo
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewCDRipper creates a new CD ripper instance
func NewCDRipper(cfg *config.Config) *CDRipper {
	ctx, cancel := context.WithCancel(context.Background())
	return &CDRipper{
		config:     cfg,
		progressCh: make(chan ProgressInfo, 10),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// GetProgressChannel returns the progress channel
func (r *CDRipper) GetProgressChannel() <-chan ProgressInfo {
	return r.progressCh
}

// Stop cancels the ripping operation
func (r *CDRipper) Stop() {
	r.cancel()
}

// DetectCD attempts to detect if a CD is present and get its information
func (r *CDRipper) DetectCD() (*CDInfo, error) {
	if r.config.Tools.CDDiscidPath == "" {
		return nil, fmt.Errorf("cd-discid tool not configured")
	}

	// Use cd-discid to get basic CD information
	cmd := exec.Command(r.config.Tools.CDDiscidPath, r.config.Drives.CDDrive)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to detect CD: %w", err)
	}

	// Parse cd-discid output
	// Format: discid numtracks offset1 offset2 ... offsetN length
	parts := strings.Fields(string(output))
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid cd-discid output")
	}

	discID := parts[0]
	trackCount, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid track count: %w", err)
	}

	// Parse track offsets for CDDB/MusicBrainz queries
	offsets := make([]int, 0, trackCount+1)
	for i := 2; i < 2+trackCount+1 && i < len(parts); i++ {
		if offset, err := strconv.Atoi(parts[i]); err == nil {
			offsets = append(offsets, offset)
		}
	}

	cdInfo := &CDInfo{
		DiscID:     discID,
		CDDBDiscID: discID, // cd-discid already provides CDDB format
		TrackCount: trackCount,
		Offsets:    offsets,
		Artist:     "Unknown Artist",
		Album:      "Unknown Album",
		Tracks:     make([]TrackInfo, trackCount),
	}

	// Initialize track information
	for i := 0; i < trackCount; i++ {
		cdInfo.Tracks[i] = TrackInfo{
			Number: i + 1,
			Title:  fmt.Sprintf("Track %02d", i+1),
			Artist: cdInfo.Artist,
		}
	}

	// Try to get additional metadata from CDDB if configured
	if r.config.CDRipping.CDDBMethod != "none" {
		// Send progress update about metadata lookup
		r.progressCh <- ProgressInfo{
			Status: fmt.Sprintf("Looking up metadata via %s...", r.config.CDRipping.CDDBMethod),
		}
		
		if err := r.lookupCDDB(cdInfo); err != nil {
			// Log warning but don't fail - basic disc info is still useful
			fmt.Printf("Warning: %s lookup failed: %v\n", r.config.CDRipping.CDDBMethod, err)
			r.progressCh <- ProgressInfo{
				Status: fmt.Sprintf("Metadata lookup failed, using basic disc info"),
			}
		} else {
			r.progressCh <- ProgressInfo{
				Status: fmt.Sprintf("Metadata retrieved successfully"),
			}
		}
	}

	return cdInfo, nil
}

// lookupCDDB attempts to lookup CD information from CDDB/MusicBrainz
func (r *CDRipper) lookupCDDB(cdInfo *CDInfo) error {
	switch r.config.CDRipping.CDDBMethod {
	case "musicbrainz":
		return r.lookupMusicBrainz(cdInfo)
	case "cddb":
		return r.lookupCDDBClassic(cdInfo)
	case "none":
		return nil
	default:
		return fmt.Errorf("unknown CDDB method: %s", r.config.CDRipping.CDDBMethod)
	}
}


// lookupMusicBrainz queries the MusicBrainz API using CDDB disc ID
func (r *CDRipper) lookupMusicBrainz(cdInfo *CDInfo) error {
	// For now, try a simple MusicBrainz search by release
	// This is a fallback approach since proper disc ID calculation is complex
	
	// Try to use abcde's built-in CDDB/MusicBrainz support instead
	// by letting abcde handle the metadata lookup during ripping
	return fmt.Errorf("MusicBrainz lookup via API not fully implemented - will use abcde's built-in support during ripping")
}

// calculateMusicBrainzDiscID creates a MusicBrainz disc ID from track offsets
func (r *CDRipper) calculateMusicBrainzDiscID(offsets []int, trackCount int) (string, error) {
	// MusicBrainz disc ID calculation is complex and would require
	// implementing the SHA-1 based algorithm. For now, we'll use
	// the CDDB disc ID as a fallback since many MusicBrainz queries
	// also work with CDDB IDs in some contexts.
	
	// In a production implementation, you would:
	// 1. Calculate proper MusicBrainz disc ID using SHA-1
	// 2. Use the libdiscid library or implement the algorithm
	
	return "", fmt.Errorf("MusicBrainz disc ID calculation not implemented - using CDDB fallback")
}

// lookupCDDBClassic queries a classic CDDB server
func (r *CDRipper) lookupCDDBClassic(cdInfo *CDInfo) error {
	// Try to use external tools for CDDB lookup since the protocol is complex
	// Most CD ripping tools like abcde handle this automatically
	
	// Check if we can use cd-info or similar tools
	if cdInfoCmd, err := exec.LookPath("cd-info"); err == nil {
		return r.lookupWithCdInfo(cdInfoCmd, cdInfo)
	}
	
	// Fallback: let abcde handle CDDB lookup during ripping
	return fmt.Errorf("no CDDB lookup tools available - abcde will handle metadata during ripping")
}

// lookupWithCdInfo uses cd-info to get CD metadata
func (r *CDRipper) lookupWithCdInfo(cdInfoPath string, cdInfo *CDInfo) error {
	// cd-info can provide additional metadata about the CD
	cmd := exec.Command(cdInfoPath, "--no-header", "--no-disc-mode", r.config.Drives.CDDrive)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("cd-info command failed: %w", err)
	}
	
	// Parse cd-info output for any available metadata
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "CD-Text") {
			// Look for CD-Text information if available
			if strings.Contains(line, "TITLE") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					cdInfo.Album = strings.TrimSpace(parts[1])
				}
			}
			if strings.Contains(line, "PERFORMER") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					cdInfo.Artist = strings.TrimSpace(parts[1])
				}
			}
		}
	}
	
	return nil
}


// RipCD starts the CD ripping process
func (r *CDRipper) RipCD(cdInfo *CDInfo) error {
	if r.config.Tools.AbcdePath == "" {
		return fmt.Errorf("abcde tool not configured")
	}

	// Create output directory
	outputDir := r.config.Paths.Music
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Send initial progress
	r.progressCh <- ProgressInfo{
		CurrentTrack: 0,
		TotalTracks:  cdInfo.TrackCount,
		Status:       "Initializing rip...",
		Progress:     0,
	}

	// Prepare abcde command
	cmd := r.prepareAbcdeCommand(cdInfo, outputDir)
	
	// Start the command
	cmd.Dir = outputDir
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start abcde: %w", err)
	}

	// Monitor progress
	go r.monitorAbcdeProgress(stdout, stderr, cdInfo.TrackCount)

	// Wait for completion or cancellation
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-r.ctx.Done():
		// Cancel the process
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		r.progressCh <- ProgressInfo{
			Status: "Ripping cancelled",
			Error:  fmt.Errorf("operation cancelled"),
		}
		return fmt.Errorf("operation cancelled")
	case err := <-done:
		if err != nil {
			r.progressCh <- ProgressInfo{
				Status: "Ripping failed",
				Error:  err,
			}
			return fmt.Errorf("abcde failed: %w", err)
		}
	}

	r.progressCh <- ProgressInfo{
		CurrentTrack: cdInfo.TrackCount,
		TotalTracks:  cdInfo.TrackCount,
		Status:       "Ripping completed successfully!",
		Progress:     100,
	}

	return nil
}

// prepareAbcdeCommand prepares the abcde command with proper configuration
func (r *CDRipper) prepareAbcdeCommand(cdInfo *CDInfo, outputDir string) *exec.Cmd {
	args := []string{
		"-o", r.config.CDRipping.OutputFormat,
		"-d", r.config.Drives.CDDrive,
		"-c", "/dev/null", // Don't use system config file
	}

	// Add CDDB configuration
	if r.config.CDRipping.CDDBMethod != "none" {
		args = append(args, "-D")
	} else {
		args = append(args, "-L")
	}

	// Add other options
	if r.config.CDRipping.AutoEject {
		args = append(args, "-e")
	}

	// Add verbose mode if enabled
	if r.config.Execution.VerboseLogging {
		args = append(args, "-v")
	}

	cmd := exec.CommandContext(r.ctx, r.config.Tools.AbcdePath, args...)
	
	// Set environment variables for abcde
	env := os.Environ()
	env = append(env, fmt.Sprintf("OUTPUTDIR=%s", outputDir))
	env = append(env, fmt.Sprintf("OUTPUTFORMAT=${ARTISTFILE}/${ALBUMFILE}/${TRACKNUM}_${TRACKFILE}"))
	cmd.Env = env

	return cmd
}

// monitorAbcdeProgress monitors the abcde output for progress information
func (r *CDRipper) monitorAbcdeProgress(stdout, stderr io.ReadCloser, totalTracks int) {
	defer stdout.Close()
	defer stderr.Close()

	// Create scanners for both stdout and stderr
	stdoutScanner := bufio.NewScanner(stdout)
	stderrScanner := bufio.NewScanner(stderr)

	// Regex patterns to match abcde output
	trackPattern := regexp.MustCompile(`Grabbing track (\d+)`)
	encodePattern := regexp.MustCompile(`Encoding track (\d+)`)
	
	currentTrack := 0
	
	// Monitor stdout
	go func() {
		for stdoutScanner.Scan() {
			line := stdoutScanner.Text()
			
			if matches := trackPattern.FindStringSubmatch(line); len(matches) > 1 {
				if track, err := strconv.Atoi(matches[1]); err == nil {
					currentTrack = track
					progress := (currentTrack * 50) / totalTracks // Ripping is ~50% of process
					r.progressCh <- ProgressInfo{
						CurrentTrack: currentTrack,
						TotalTracks:  totalTracks,
						Status:       fmt.Sprintf("Ripping track %d of %d...", currentTrack, totalTracks),
						Progress:     progress,
					}
				}
			}
			
			if matches := encodePattern.FindStringSubmatch(line); len(matches) > 1 {
				if track, err := strconv.Atoi(matches[1]); err == nil {
					progress := 50 + ((track * 50) / totalTracks) // Encoding is remaining 50%
					r.progressCh <- ProgressInfo{
						CurrentTrack: track,
						TotalTracks:  totalTracks,
						Status:       fmt.Sprintf("Encoding track %d of %d...", track, totalTracks),
						Progress:     progress,
					}
				}
			}
		}
	}()

	// Monitor stderr for errors
	go func() {
		for stderrScanner.Scan() {
			line := stderrScanner.Text()
			if strings.Contains(strings.ToLower(line), "error") {
				r.progressCh <- ProgressInfo{
					Status: fmt.Sprintf("Error: %s", line),
					Error:  fmt.Errorf("abcde error: %s", line),
				}
			}
		}
	}()
}

// HasMedia checks if there's a CD in the drive
func (r *CDRipper) HasMedia() bool {
	if r.config.Drives.CDDrive == "" {
		return false
	}

	// Try to access the device
	file, err := os.Open(r.config.Drives.CDDrive)
	if err != nil {
		return false
	}
	defer file.Close()

	// If we can open it without error, assume media is present
	// In a more sophisticated implementation, you would use ioctl calls
	return true
}