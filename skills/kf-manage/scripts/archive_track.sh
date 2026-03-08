#!/usr/bin/env bash

# Exit immediately if a command exits with a non-zero status
set -e

if [ -z "$1" ]; then
  echo "Usage: $0 <track-id> [archive-reason]"
  exit 1
fi

TRACK_ID="$1"
REASON="${2:-Completed}"
TRACK_DIR=".agent/conductor/tracks/$TRACK_ID"
ARCHIVE_DIR=".agent/conductor/tracks/_archive/$TRACK_ID"
METADATA_FILE="$TRACK_DIR/metadata.json"
TRACKS_MD=".agent/conductor/tracks.md"

if [ ! -d "$TRACK_DIR" ]; then
  echo "Error: Track directory '$TRACK_DIR' does not exist."
  exit 1
fi

# Fetch the track title from metadata.json before moving
TITLE=$(python3 -c "import json; print(json.load(open('$METADATA_FILE'))['title'])")
TODAY=$(date '+%Y-%m-%d')

echo "Archiving track: $TRACK_ID ($TITLE)..."

# 1. Update metadata.json status to 'archived', record reason & date.
python3 -c "
import json
from datetime import datetime
file_path = '$METADATA_FILE'
with open(file_path, 'r') as f:
    data = json.load(f)
    
data['status'] = 'archived'
data['archived'] = True
data['archived_at'] = datetime.now().astimezone().isoformat()
data['archive_reason'] = '$REASON'

with open(file_path, 'w') as f:
    json.dump(data, f, indent=2)
"

# 2. Move to archive
mkdir -p .agent/conductor/tracks/_archive
mv "$TRACK_DIR" "$ARCHIVE_DIR"

# 3. Update the master tracks.md registry
if [ -f "$TRACKS_MD" ]; then
  # Remove the row from the table
  sed -i '' "/|.*$TRACK_ID.*/d" "$TRACKS_MD"
  
  # Ensure an ## Archived Tracks header exists, or append it
  if ! grep -q "## Archived Tracks" "$TRACKS_MD"; then
    echo -e "\n## Archived Tracks\n" >> "$TRACKS_MD"
  fi
  
  # Append the specific track entry
  cat <<EOF >> "$TRACKS_MD"
### $TRACK_ID: $TITLE

**Reason:** $REASON
**Archived:** $TODAY
**Folder:** [./tracks/_archive/$TRACK_ID/](./tracks/_archive/$TRACK_ID/)
EOF
fi

# 6. Update the root conductor index.md
CONDUCTOR_INDEX=".agent/conductor/index.md"
if [ -f "$CONDUCTOR_INDEX" ]; then
  # Remove the track from the Active Tracks list
  sed -i '' "/.*$TRACK_ID.*/d" "$CONDUCTOR_INDEX"
  git add "$CONDUCTOR_INDEX"
fi

# 7. Commit the changes
git add "$ARCHIVE_DIR" "$TRACKS_MD"
git commit -m "chore(conductor): archive track '$TRACK_ID' (Reason: $REASON)"

echo "Successfully archived $TRACK_ID!"
