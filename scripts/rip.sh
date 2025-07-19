#!/bin/bash
set -x

export PATH="/usr/local/bin:/usr/bin:/bin:$PATH"

LOG="$HOME/cd-ripper.log"
RETRY_COUNT=3
RETRY_DELAY=5
INITIAL_WAIT=10
MUSIC_DIR="/mnt/nas/media/music"

log_event() {
    local level=$1
    local message=$2
    local timestamp=$(date --iso-8601=seconds)
    echo "{"timestamp":"$timestamp","level":"$level","event":"cd_rip","message":"$message"}" >> "$LOG"
}

# Check if the disc is an audio CD
check_audio_cd() {
    cd-discid /dev/sr0 >/dev/null 2>&1
    if [ $? -eq 0 ]; then
        return 0
    else
        log_event "info" "Non-audio disc detected, skipping"
        return 1
    fi
}

# Function to test CD readability and keep the TOC info
get_cd_info() {
    local attempt=1
    local toc_info=""
    local total_attempts=0
    while [ $attempt -le $RETRY_COUNT ]; do
        log_event "info" "Reading CD TOC (attempt $attempt)"
        ((total_attempts++))
        toc_info=$(cd-discid /dev/sr0 2>&1)
        if [ $? -eq 0 ]; then
            echo "$toc_info"
            return 0
        fi
        log_event "warn" "CD-discid failed on attempt $attempt with: $toc_info"
        if [ $attempt -lt $RETRY_COUNT ]; then
            sleep $RETRY_DELAY
        fi
        attempt=$((attempt + 1))
    done
    return 1
}

# Check if album already exists
check_existing_album() {
    local discid=$1
    # First try to get MusicBrainz info
    local mb_info=$(abcde-musicbrainz-tool --musicbrainz-id "$discid" 2>/dev/null)
    if [ $? -eq 0 ]; then
        local artist=$(echo "$mb_info" | grep "ARTIST=" | cut -d'=' -f2)
        local album=$(echo "$mb_info" | grep "ALBUM=" | cut -d'=' -f2)
        if [ -n "$artist" ] && [ -n "$album" ]; then
            local expected_dir="$MUSIC_DIR/${artist}/${album}"
            if [ -d "$expected_dir" ]; then
                log_event "info" "Album already exists: $artist - $album"
                return 0
            fi
        fi
    fi
    # Fallback check for Unknown Artist
    if [ -d "$MUSIC_DIR/Unknown Artist-Unknown Album" ]; then
        # Compare TOC info to be extra sure
        local existing_toc=$(find "$MUSIC_DIR" -name "*.toc" -type f -exec cat {} \; 2>/dev/null | grep "$discid")
        if [ -n "$existing_toc" ]; then
            log_event "info" "Album already exists (unknown artist)"
            return 0
        fi
    fi
    return 1
}

# Main execution
START_TIME=$(date +%s)
log_event "info" "Starting CD rip process"
if ! check_audio_cd; then
    log_event "info" "Not an audio CD, exiting"
    eject /dev/sr0
    exit 0
fi
log_event "info" "Waiting for drive to initialize"
sleep $INITIAL_WAIT
CD_INFO=$(get_cd_info)
if [ $? -eq 0 ]; then
    log_event "info" "CD detected: $CD_INFO"
    DISC_ID=$(echo "$CD_INFO" | cut -d' ' -f1)
    TRACK_COUNT=$(echo "$CD_INFO" | cut -d' ' -f2)
    if check_existing_album "$DISC_ID"; then
        log_event "info" "Album already exists, ejecting"
        eject /dev/sr0
        exit 0
    fi
    # Run abcde
    log_event "info" "Starting abcde rip"
    ABCDE_START=$(date +%s)
    ABCDE_OUT=$(CDDBMETHOD=musicbrainz OUTPUTDIR="$MUSIC_DIR" abcde -N -d /dev/sr0 -o flac 2>&1)
    RIP_STATUS=$?
    ABCDE_END=$(date +%s)
    if [ $RIP_STATUS -eq 0 ]; then
        log_event "info" "Rip completed successfully"
        # Save TOC info for future reference
        echo "$CD_INFO" > "$MUSIC_DIR/Unknown Artist-Unknown Album/$DISC_ID.toc" 2>/dev/null
        # Eject the CD after successful rip
        log_event "info" "Ejecting CD"
        eject /dev/sr0
        if [ $? -eq 0 ]; then
            log_event "info" "CD ejected successfully"
        else
            log_event "warn" "Failed to eject CD"
        fi
    else
        log_event "error" "abcde failed with status $RIP_STATUS"
        log_event "error" "abcde output: $ABCDE_OUT"
    fi
else
    log_event "error" "Could not read CD TOC"
    exit 1
fi
END_TIME=$(date +%s)
TOTAL_DURATION=$((END_TIME - START_TIME))
log_event "info" "Rip process completed in $TOTAL_DURATION seconds"
exit $RIP_STATUS