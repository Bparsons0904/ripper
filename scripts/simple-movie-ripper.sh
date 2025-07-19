#!/bin/bash

# Simple Movie Ripper - Clean start focused on video disc ripping
# Manual script with clear title information and user selection

set -e

# Configuration
OPTICAL_DEVICE="/dev/sr0"
MOVIES_DIR="/mnt/nas/media/movies"
LOG_FILE="$HOME/.cache/media-ripper/simple-ripper.log"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

# Create directories
mkdir -p "$(dirname "$LOG_FILE")"

# Logging
log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a "$LOG_FILE"
}

info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_requirements() {
    info "Checking requirements..."
    
    if [ ! -b "$OPTICAL_DEVICE" ]; then
        error "Optical device $OPTICAL_DEVICE not found"
        exit 1
    fi
    
    if ! command -v makemkvcon >/dev/null 2>&1; then
        error "MakeMKV not found. Please install it first."
        exit 1
    fi
    
    success "All requirements met"
}

# Wait for disc
wait_for_disc() {
    info "Waiting for disc to be ready..."
    local attempts=0
    
    while [ $attempts -lt 30 ]; do
        if [ -r "$OPTICAL_DEVICE" ]; then
            success "Disc is ready"
            return 0
        fi
        sleep 1
        ((attempts++))
    done
    
    error "Timeout waiting for disc"
    return 1
}

# Get disc information
get_disc_info() {
    info "Scanning disc (this may take a minute)..." >&2
    
    # Get basic disc info
    local disc_info
    disc_info=$(makemkvcon info dev:"$OPTICAL_DEVICE" 2>&1)
    
    if [ $? -ne 0 ]; then
        error "Failed to scan disc" >&2
        return 1
    fi
    
    # Extract disc title
    local disc_title="Unknown"
    if [[ $disc_info =~ Name\ \"([^\"]+)\" ]]; then
        disc_title="${BASH_REMATCH[1]}"
    elif command -v blkid >/dev/null && blkid -p "$OPTICAL_DEVICE" 2>/dev/null | grep -q LABEL; then
        disc_title=$(blkid -p "$OPTICAL_DEVICE" 2>/dev/null | grep -o 'LABEL="[^"]*"' | cut -d'"' -f2)
    fi
    
    info "Disc: $disc_title" >&2
    
    # Parse available titles
    echo "" >&2
    echo "Available Titles:" >&2
    echo "=================" >&2
    printf "%-5s %-15s %-12s %-10s %-30s\n" "Title" "Source File" "Duration" "Size" "Notes" >&2
    echo "--------------------------------------------------------------------------------" >&2
    
    local title_count=0
    local max_title=0
    declare -a title_list
    
    # First pass: collect basic title info
    while IFS= read -r line; do
        if [[ $line =~ File\ ([0-9]+\.mpls)\ was\ added\ as\ title\ #([0-9]+) ]]; then
            local file="${BASH_REMATCH[1]}"
            local title_num="${BASH_REMATCH[2]}"
            title_list+=("$title_num:$file")
            ((title_count++))
            [ $title_num -gt $max_title ] && max_title=$title_num
        fi
    done <<< "$disc_info"
    
    # Second pass: get detailed info for each title (with timeout to avoid hanging)
    info "Getting detailed title information..." >&2
    
    for entry in "${title_list[@]}"; do
        IFS=':' read -r title_num file <<< "$entry"
        
        local duration="unknown"
        local size="unknown"
        
        # Try to get detailed info with robot format (with timeout)
        local detailed_info
        detailed_info=$(timeout 30 makemkvcon -r --robot --minlength=0 info dev:"$OPTICAL_DEVICE" 2>/dev/null | grep "^TINFO:$title_num," || echo "")
        
        # Parse duration and size
        while IFS= read -r detail_line; do
            if [[ $detail_line =~ ^TINFO:$title_num,9,0,\"([^\"]+)\" ]]; then
                duration="${BASH_REMATCH[1]}"
            fi
            if [[ $detail_line =~ ^TINFO:$title_num,10,0,\"([^\"]+)\" ]]; then
                size="${BASH_REMATCH[1]}"
            fi
        done <<< "$detailed_info"
        
        # Guess content type based on file number and size
        local notes=""
        local color=""
        local reset="\033[0m"
        
        case "$file" in
            "00001.mpls"|"00002.mpls"|"00003.mpls")
                notes="🎬 Likely main feature"
                color="\033[1;32m"  # Green
                ;;
            "00800.mpls"|"00801.mpls")
                notes="📽️ Possible main feature"
                color="\033[1;33m"  # Yellow
                ;;
            *)
                notes="📺 Likely extra content"
                color="\033[0;37m"  # Gray
                ;;
        esac
        
        # Override color based on size if we got it
        if [[ $size =~ ([0-9.]+)\ *GB ]] && [ "$(echo "${BASH_REMATCH[1]} > 15" | bc 2>/dev/null || echo 0)" = "1" ]; then
            color="\033[1;32m"  # Green for large files
            notes="🎬 Main feature"
        elif [[ $size =~ ([0-9.]+)\ *GB ]] && [ "$(echo "${BASH_REMATCH[1]} > 5" | bc 2>/dev/null || echo 0)" = "1" ]; then
            color="\033[1;33m"  # Yellow for medium files
            notes="📽️ Feature content"
        fi
        
        printf "${color}%-5s %-15s %-12s %-10s %-30s${reset}\n" "$title_num" "$file" "$duration" "$size" "$notes" >&2
    done
    
    if [ $title_count -eq 0 ]; then
        error "No titles found on disc" >&2
        return 1
    fi
    
    echo "" >&2
    info "Found $title_count titles" >&2
    echo "🎬 = Main feature (usually titles 0-3)" >&2
    echo "📽️ = Alternative version" >&2  
    echo "📺 = Extras/bonus content" >&2
    
    echo "$max_title|$disc_title"
}

# Get user selection
get_title_selection() {
    local max_title="$1"
    local disc_title="$2"
    
    echo "" >&2
    local selected_title
    while true; do
        read -rp "Select title number (0-$max_title) or 'q' to quit: " input
        
        case "$input" in
            q|Q)
                info "Cancelled by user" >&2
                return 1
                ;;
            ''|*[!0-9]*)
                warn "Please enter a valid number or 'q'" >&2
                ;;
            *)
                if [ "$input" -ge 0 ] && [ "$input" -le "$max_title" ]; then
                    selected_title="$input"
                    break
                else
                    warn "Please enter a number between 0 and $max_title" >&2
                fi
                ;;
        esac
    done
    
    # Confirm selection
    info "Selected title: $selected_title" >&2
    read -rp "Is this correct? (y/n): " confirm
    if [[ ! $confirm =~ ^[Yy] ]]; then
        warn "Selection cancelled" >&2
        return 1
    fi
    
    echo "$selected_title|$disc_title"
}

# Get output name
get_output_name() {
    local disc_title="$1"
    
    echo "" >&2
    echo "Output naming:" >&2
    echo "Suggested name: $disc_title" >&2
    
    while true; do
        read -rp "Use this name? (y/n): " use_suggested
        
        case "$use_suggested" in
            [Yy]*)
                echo "$disc_title"
                return 0
                ;;
            [Nn]*)
                read -rp "Enter movie name: " custom_name
                if [ -n "$custom_name" ]; then
                    echo "$custom_name"
                    return 0
                else
                    warn "Name cannot be empty" >&2
                fi
                ;;
            *)
                warn "Please answer y or n" >&2
                ;;
        esac
    done
}

# Clean filename
clean_filename() {
    echo "$1" | tr ' ' '_' | tr -cd '[:alnum:]_.-'
}

# Rip the disc
rip_disc() {
    local title="$1"
    local movie_name="$2"
    
    local clean_name
    clean_name=$(clean_filename "$movie_name")
    local output_dir="$MOVIES_DIR/$clean_name"
    
    echo "" >&2
    info "Rip settings:" >&2
    echo "Title: $title" >&2
    echo "Movie: $movie_name" >&2
    echo "Output: $output_dir" >&2
    
    read -rp "Proceed with rip? (y/n): " proceed
    if [[ ! $proceed =~ ^[Yy] ]]; then
        warn "Rip cancelled" >&2
        return 1
    fi
    
    # Create output directory
    mkdir -p "$output_dir"
    
    # Start ripping
    info "Starting MakeMKV rip..." >&2
    log "Starting rip: $movie_name (title $title)"
    
    if makemkvcon mkv dev:"$OPTICAL_DEVICE" "$title" "$output_dir"; then
        # Find and rename output file
        local mkv_file
        mkv_file=$(find "$output_dir" -name "*.mkv" -type f | head -n 1)
        
        if [ -n "$mkv_file" ] && [ -f "$mkv_file" ]; then
            local final_name="$output_dir/${clean_name}.mkv"
            
            if [ "$mkv_file" != "$final_name" ]; then
                mv "$mkv_file" "$final_name"
            fi
            
            success "Rip completed: $final_name" >&2
            log "Rip completed: $final_name"
            
            # Show file info
            local file_size
            file_size=$(du -h "$final_name" | cut -f1)
            info "File size: $file_size" >&2
            
            return 0
        else
            error "No output file found" >&2
            log "Rip failed: no output file"
            return 1
        fi
    else
        error "MakeMKV rip failed" >&2
        log "Rip failed: MakeMKV error"
        return 1
    fi
}

# Main function
main() {
    echo -e "${GREEN}Simple Movie Ripper${NC}"
    echo "Focused tool for ripping video discs"
    echo ""
    
    # Check requirements
    check_requirements
    
    # Wait for disc
    if ! wait_for_disc; then
        exit 1
    fi
    
    # Get disc information and available titles
    local disc_result
    disc_result=$(get_disc_info)
    if [ $? -ne 0 ]; then
        error "Failed to get disc information"
        exit 1
    fi
    
    # Parse results
    local max_title disc_title
    IFS='|' read -r max_title disc_title <<< "$disc_result"
    
    # Get user title selection
    local selection_result
    selection_result=$(get_title_selection "$max_title" "$disc_title")
    if [ $? -ne 0 ]; then
        exit 1
    fi
    
    # Parse selection
    local selected_title
    IFS='|' read -r selected_title disc_title <<< "$selection_result"
    
    # Get output name
    local movie_name
    movie_name=$(get_output_name "$disc_title")
    if [ $? -ne 0 ]; then
        exit 1
    fi
    
    # Rip the disc
    if rip_disc "$selected_title" "$movie_name"; then
        success "All done! Movie ripped successfully."
    else
        error "Rip failed"
        exit 1
    fi
}

# Run main function
main "$@"