# Hotreload

Automatic Go project rebuilder and server restarter. Watch files, rebuild on change, restart server — zero external tool dependencies.

```bash
# Usage
hotreload --root ./myproject \
          --build "go build -o ./bin/server ./cmd/server" \
          --exec "./bin/server"
```

## Features

✅ **Recursive File Watching** — Monitors all project directories with OS-level notifications (<100ms latency)  
✅ **Smart Debouncing** — 150ms window collapses rapid saves into single rebuild  
✅ **Process Group Management** — Clean server restarts with SIGTERM → SIGKILL  
✅ **Crash Detection** — Auto-detects server crashes and rebuilds  
✅ **Crash-Loop Prevention** — 3 crashes in 10s triggers 5s backoff  
✅ **Real-Time Output** — No buffering, see build/server logs immediately  
✅ **Shell Command Support** — Quotes + complex args handled automatically  
✅ **.hotreloadignore** — Exclude files/folders via a simple ignore file in your project root  

## Installation

### Using Go (Recommended for developers)

Requires Go 1.24+

```bash
go install github.com/HITENDRAS940/hotreload@latest
```

This installs the `hotreload` binary to `$GOBIN` (usually `~/go/bin`). Add to PATH if not already:

```bash
export PATH=$HOME/go/bin:$PATH
```

### Install Latest from Source

```bash
git clone https://github.com/HITENDRAS940/hotreload.git
cd hotreload
go build -o hotreload .
```

## Usage

### Basic Example

```bash
hotreload \
  --root ./myproject \
  --build "go build -o ./bin/server ./cmd/server" \
  --exec "./bin/server"
```

**Flags:**
- `--root` — Project directory to watch (default: `.`)
- `--build` — Shell command to rebuild project (required)
- `--exec` — Shell command to run after build (required)

### Ignoring Files and Folders (.hotreloadignore)

Place a `.hotreloadignore` file in your project root to exclude files and folders. One pattern per line:

```
# My project ignore file

.cache
tmp
*.log
.env.local
generated/
```

**If `.hotreloadignore` is missing**, hotreload will warn you and ask:
```
[WARNING] .hotreloadignore not found in: /path/to/project
Without it, ALL file changes in the project will trigger rebuilds.

Would you like to continue without .hotreloadignore? (y/n):
```

- **`y`** → Continue, all file changes trigger rebuilds
- **`n`** → Automatically creates a `.hotreloadignore` with sensible defaults in your project root, then continues

**On every start**, hotreload logs which patterns are active:
```
level=INFO msg="loaded .hotreloadignore" patterns=5
level=INFO msg="  excluding" pattern=.cache
level=INFO msg="  excluding" pattern=tmp
level=INFO msg="  excluding" pattern=*.log
```

### Examples

**Web Server (Go + templ templates)**
```bash
hotreload \
  --root ./web \
  --build "sh -c 'templ generate && go build -o ./bin/server ./cmd/server'" \
  --exec "./bin/server --port 8080"
```

**Build Only (No Server)**
```bash
hotreload \
  --root . \
  --build "go build -o ./bin/app ./cmd/app"
```

**With Environment Variables**
```bash
hotreload \
  --root . \
  --build "sh -c 'export VERSION=\$(git describe) && go build -ldflags \"-X main.Version=\$VERSION\" -o ./bin/server'" \
  --exec "./bin/server"
```

## How It Works

1. **Startup** → Triggers first build immediately (no waiting for file change)
2. **File Change Detected** → Debounced 150ms to collapse rapid edits
3. **Server Stopped** → Gracefully (SIGTERM → 3s wait → SIGKILL)
4. **Build Started** → Build command executed with real-time output
5. **Build Succeeds** → New server started with fresh process
6. **Server Crashes** → Auto-detected and rebuilt
7. **Crash Loop** → 3+ crashes in 10s → 5s backoff to prevent thrashing

## CLI Help

```bash
hotreload --help
```

```
Usage: hotreload [flags]

Flags:
  -build string
        build command (e.g. "go build -o ./bin/server ./cmd/server")
  -exec string
        exec command to run after build (e.g. "./bin/server")
  -root string
        root directory of the project to watch (default ".")

Example:
  hotreload --root ./myproject --build "go build -o ./bin/server ./cmd/server" --exec "./bin/server"

Ignore patterns:
  Add a .hotreloadignore file in your project root to exclude files/folders.
```

## Installation Verification

After installing, verify it works:

```bash
hotreload --help
```

Should output the usage message above.

## Building from Source

```bash
git clone https://github.com/HITENDRAS940/hotreload.git
cd hotreload
go build -o hotreload .
```

## Performance

- **File change → server ready:** <500ms
- **Debounce window:** 150ms (prevents 5 saves triggering 5 rebuilds)
- **Graceful shutdown timeout:** 3 seconds (SIGTERM → SIGKILL)
- **Crash-loop backoff:** 5 seconds
- **Memory idle:** <15MB
- **CPU idle:** <1%

## Troubleshooting

### Rebuilds not triggering
Check that the file matches one of:
- `.go` source files
- Directory-level events for new directory detection

Ignored patterns: `.git/`, `node_modules/`, `go.sum`, editor temp files, build artifacts.

### Server not starting after build
- Verify build command works standalone: `go build -o ./bin/server ./cmd/server`
- Check that binary path is correct in `--exec`
- Build output will show errors

### Server logs not appearing
All output is streamed in real-time (no buffering). If you don't see server output:
- Verify `--exec` command is correct
- Check that server writes to stdout (not file)
- Server may have exited (check crash logs)

## Requirements

- **Go 1.24+** (for `go install`/`go build`)
- **macOS, Linux, or Windows** (cross-platform compatible)
- No other external dependencies at runtime

## Author

HITENDRAS940

---

