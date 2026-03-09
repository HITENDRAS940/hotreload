# Hotreload

Automatic Go project rebuilder and server restarter. Watch files, rebuild on change, restart server — zero external tool dependencies.

## Features

✅ **Recursive File Watching** — Monitors all project directories with OS-level notifications (<100ms latency)  
✅ **Smart Debouncing** — 500ms window collapses rapid saves into single rebuild  
✅ **Process Group Management** — Clean server restarts with SIGTERM → SIGKILL  
✅ **Crash Detection** — Auto-detects server crashes and rebuilds  
✅ **Crash-Loop Prevention** — 3 crashes in 10s triggers 5s backoff  
✅ **Real-Time Output** — No buffering, see build/server logs immediately  
✅ **Shell Command Support** — Quotes + complex args handled automatically  
✅ **.hotreloadignore** — Exclude files/folders via a simple ignore file in your project root  

## Installation

### Using Go (Recommended for developers)

Requires Go 1.24+, Use this for downloading and updating the HOTRELOAD CLI 

```bash
go install github.com/HITENDRAS940/hotreload@latest
```

This installs the `hotreload` binary to `$GOBIN` (usually `~/go/bin`). Add to PATH if not already:

### Mac

```bash
nano ~/.zshrc
```
Add this line

```bash
export PATH=$HOME/go/bin:$PATH
```
press ^X -> press y -> press return

```bash
source ~/.zshrc
```

### Windows

1. Press **Windows Key** and search for **Environment Variables**.
2. Click **Edit the system environment variables**.
3. Click **Environment Variables**.
4. Under **User variables**, find **Path** and click **Edit**.
5. Click **New** and add:

```
%USERPROFILE%\go\bin
```

6. Click **OK** to save.
7. Restart your terminal.

After this, you should be able to run:

```bash
which hotreload
```

## Usage

### Basic Example

***For Mac***

Navigate to the project directory and run:

```bash
hotreload \
  --root . \
  --build "go build -o ./bin/server ." \
  --exec "./bin/server"
```

***For Windows***

```bash
hotreload \
  --root . \
  --build "go build -o ./bin/server.exe ." \
  --exec "./bin/server.exe"
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
- **`n`** → Asks you to create a `.hotreloadignore` automatically with sensible defaults in your project root, then continues

**On every start**, hotreload logs which patterns are active:
```
 ⚠  crash #1 detected in last 10s
  ●  build started
  ✗  build failed  (1ms)
  ✔  build complete  (190ms)
  ●  server starting
  ✔  server started  (pid: 63978)  (1ms)
  │  Server starting...

```

### Examples

**With Environment Variables**
```bash
hotreload \
  --root . \
  --build "sh -c 'export VERSION=\$(git describe) && go build -ldflags \"-X main.Version=\$VERSION\" -o ./bin/server'" \
  --exec "./bin/server"
```

## How It Works

1. **Startup** → Triggers first build immediately (no waiting for file change)
2. **File Change Detected** → Debounced 500ms to collapse rapid edits
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

## Component Map

```
main.go
  └── config.Parse()          ← CLI flags & validation
  └── watcher.NewWatcher()    ← FS event loop + debounce
  └── builder.NewBuilder()    ← Build command executor
  └── runner.NewRunner()      ← Process lifecycle manager
  └── orchestrator.New()      ← Central coordinator
        └── orch.Run()        ← goroutine: event dispatch loop
```

### Package responsibilities

| Package | Role |
|---|---|
| `config` | Parses `--root`, `--build`, `--exec` flags and validates them before anything else starts |
| `watcher` | Wraps `fsnotify`, walks the root directory recursively, applies ignore rules, debounces rapid events into a single signal |
| `builder` | Runs the build command as a subprocess, forwards stdout/stderr, supports `context.Context` cancellation |
| `runner` | Starts and stops the server process, manages its process group for clean termination, monitors for unexpected exits |
| `orchestrator` | The central brain — listens for watcher events, cancels in-flight builds, sequences build→start, and handles crash-loop detection |
| `ui` | All terminal output — writes exclusively to `stderr` so the server's `stdout` is never polluted |

---

## Data Flow

```
File change on disk
      │
      ▼
  [Watcher]  ──(fsnotify raw event)──►  filter (shouldIgnore?)
                                              │ passes
                                              ▼
                                       DebouncedSignal.Trigger()
                                              │ 500 ms quiet
                                              ▼
                                       w.events  ← chan struct{}
                                              │
      ┌───────────────────────────────────────┘
      ▼
  [Orchestrator.Run()]
      │
      ├─ cancel in-flight build (if any)
      ├─ stop running server (if any)
      └─ go runBuild(newCtx)
              │
              ▼
          [Builder.Build(ctx)]   ← blocks until done or ctx cancelled
              │ success
              ▼
          [Runner.Start()]       ← spawns new server process
              │
              └─ go waitForExit() ── on unexpected exit ──► handleServerCrash()
```

---

## Key Design Decisions

### 1. Context cancellation for preemptable builds

Every build is started with a fresh `context.WithCancel`. When a new file-change event arrives while a build is already running, `triggerRebuild` calls the previous build's `CancelFunc` before spawning a new goroutine. This means a burst of saves (e.g. auto-format on save) never queues up multiple builds — only the latest one wins.

```go
// orchestrator.go
if o.buildCancel != nil {
    o.buildCancel()   // cancels in-flight build
}
buildCtx, cancel := context.WithCancel(o.mainCtx)
o.buildCancel = cancel
go o.runBuild(buildCtx)
```

### 2. Process group termination (Unix)

On Unix, the server process is started with `Setpgid: true`, which puts it in a new process group. When stopping, `SIGTERM` (then `SIGKILL` on timeout) is sent to `-pgid` — the negative value targets the entire group. This ensures child processes spawned by the server (e.g. worker goroutines that fork, CGO helpers) are also terminated rather than becoming orphans.

On Windows, the equivalent is `taskkill /F /T /PID`, which walks the process tree. Platform selection happens at compile time via build tags (`runner_unix.go` / `runner_windows.go`).

### 3. 500 ms debounce

`DebouncedSignal` uses `time.AfterFunc` with a 500 ms window. Each call to `Trigger()` resets the timer; the downstream channel only receives a single pulse after the burst settles. This collapses editor "save storms" (e.g. writing multiple files at once, or tools that write a temp file then rename it) into a single rebuild.

### 4. Build-output directory exclusion

The directory containing the exec binary is resolved at startup and is unconditionally excluded from both the directory walk and the event loop. Without this, writing the compiled binary would trigger another rebuild, creating an infinite loop.

```go
// watcher.go — always skip, regardless of ignore rules
if abs == w.buildOutDir { return filepath.SkipDir }
```

### 5. `.hotreloadignore` with interactive fallback

If `.hotreloadignore` is absent, the watcher enters `watchAll` mode (every file change triggers a rebuild) rather than silently guessing what to ignore. The user is asked interactively whether to continue without the file or have a sensible default created. This surfaces the tradeoff explicitly instead of hiding it.

A built-in set of `defaultIgnorePatterns` (`.git`, `node_modules`, `vendor`, `bin`, editor folders, etc.) is always applied on top of any user-defined patterns.

### 6. Crash-loop detection with sliding window

The orchestrator tracks server crash timestamps in a slice and prunes entries older than 10 seconds on every crash. If 3 crashes occur within that window, the tool prints a prominent warning and stops attempting restarts. This prevents a broken binary from consuming a CPU core in a tight loop.

```go
// orchestrator.go
cutoff := now.Add(-10 * time.Second)
// ... filter crashTimes to only those after cutoff
if crashCount >= 3 {
    ui.Error("too many crashes — stopping auto-restart")
    return
}
```

### 7. All UI output on stderr

Every `ui.*` function writes to `os.Stderr`. The managed server process inherits a `ServerWriter` that prefixes its lines and also writes to `stderr`. This means `stdout` of the `hotreload` process is entirely the server's own `stdout`, making it safe to pipe (`hotreload ... | jq`) without hotreload's own log messages corrupting the stream.

### 8. Mutex discipline for concurrent state

Three distinct mutexes guard different state domains:

| Mutex | Guards |
|---|---|
| `buildMutex` | `buildCancel` — the cancel func for the current in-flight build |
| `serverMutex` | `runner.IsRunning()` / `runner.Start()` / `runner.Stop()` calls |
| `crashMutex` | `crashTimes` slice |

Keeping them separate prevents the common mistake of holding a broad lock across a blocking operation (e.g. holding `buildMutex` while waiting for a build to finish).

### 9. Optional runner

If `--exec` is omitted, `r` is `nil` and the orchestrator skips all runner/crash logic. This lets hotreload be used as a pure build-on-save tool for projects where the user manages process lifecycle externally (e.g. systemd, Docker).

---

## Directory Structure

```
hotreload/
├── main.go                    Entry point — wiring + OS signal handling
├── go.mod
├── internal/
│   ├── config/
│   │   └── config.go          Flag parsing & root path validation
│   ├── watcher/
│   │   ├── watcher.go         fsnotify wrapper, directory walk, event loop
│   │   ├── debounce.go        Timer-based DebouncedSignal
│   │   ├── filter.go          shouldIgnore + default ignore patterns
│   │   └── ignorefile.go      .hotreloadignore read/create/prompt
│   ├── builder/
│   │   └── builder.go         Shell-command parser + context-aware build runner
│   ├── runner/
│   │   ├── runner.go          Process start/stop/monitor + portable shell parser
│   │   ├── runner_unix.go     Setpgid + SIGTERM/SIGKILL to process group
│   │   └── runner_windows.go  taskkill /F /T fallback
│   ├── orchestrator/
│   │   └── orchestrator.go    Event dispatch, build cancellation, crash detection
│   └── ui/
│       └── ui.go              Coloured stderr output (fatih/color)
├── testserver/
│   └── main.go                Minimal HTTP server used for manual testing
└── bin/                       Compiled binaries (excluded from watch)
```

---

## Dependencies

| Dependency | Purpose |
|---|---|
| `github.com/fsnotify/fsnotify` | Cross-platform filesystem event notifications |
| `github.com/fatih/color` | ANSI colour output with automatic TTY detection |
| `github.com/mattn/go-colorable` | Windows ANSI compatibility (transitive via fatih/color) |
| `github.com/mattn/go-isatty` | TTY detection (transitive via fatih/color) |



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
- **macOS or Windows** (cross-platform compatible)
- No other external dependencies at runtime

## Author

HITENDRAS940

---

