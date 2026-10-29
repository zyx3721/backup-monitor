<div align="center">

<h1>Backup Monitor Agent</h1>

<p><b>Standalone host collection agent</b> — read-only discovery of backup tasks, status and artifact metadata, HTTP reporting and WebSocket configuration/update commands</p>

<p><a href="README.md">简体中文</a> · <b>English</b></p>

A standalone Go agent that discovers backup tasks on the target host, reads status and artifact metadata, reports over HTTP on a schedule, and receives configuration changes, cron updates and self-update commands over WebSocket. The protocol is defined in [`docs/agent-protocol.md`](../docs/agent-protocol.md).

Production deployment steps are covered in [Manual 4.3](../docs/manual.md); this document focuses on the agent's build, configuration and operations.

<p>
  <a href="../LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue?labelColor=1f2937" alt="MIT License"></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white&labelColor=1f2937" alt="Go 1.22+"></a>
  <img src="https://img.shields.io/badge/Agent-Linux%20%7C%20Windows-0ea5e9?labelColor=1f2937" alt="Agent Linux Windows">
</p>

<p>
  <b><a href="#repository-layout">Repository layout</a></b> ·
  <b><a href="#requirements">Requirements</a></b> ·
  <b><a href="#build">Build</a></b> ·
  <b><a href="#configuration">Configuration</a></b> ·
  <b><a href="#run">Run</a></b> ·
  <b><a href="#install-as-a-service">Install as a service</a></b> ·
  <b><a href="#upgrade-and-rollback">Upgrade and rollback</a></b> ·
  <b><a href="#troubleshooting">Troubleshooting</a></b>
</p>

</div>

---

**Agent boundaries:**

- **Read-only collection** — never modifies business data, the server, the frontend or other repository directories
- **No backup content upload** — artifact checks read only file mtime and size
- **No external runtime dependencies** — only `github.com/gorilla/websocket` and `golang.org/x/sys`; no database or runtime installation is required

## Repository layout

```text
agent/
├── backup-agent.example.json       Example configuration (all secrets empty)
├── go.mod
├── cmd/backup-agent/
│   ├── main.go                     CLI entry point and long-running loop
│   └── version_marker.go           Version marker read by the server on upload
└── internal/
    ├── api/                        HTTP client and protocol structs
    ├── bootstrap/                  flags > environment > config file > defaults
    ├── collect/                    Task and execution assembly
    ├── config/                     Local config structs, roles and defaults
    ├── cron/                       Five-field cron parsing, next run, Chinese descriptions
    ├── discover/                   crontab / Windows scheduled task discovery and write-back
    ├── identity/                   machine-id / MachineGuid / persisted UUID
    ├── logging/                    Size-based rotating file logger
    ├── proc/                       Process probing
    ├── service/                    systemd / Windows service management
    ├── statusfile/                 start/end/size/exit/status status files
    ├── sysinfo/                    Hostname, private IP and OS name
    ├── update/                     Version comparison, download verification, replace and restart
    ├── veeam/                      veeamconfig table parsing
    └── wshb/                       WebSocket heartbeat, commands and exponential backoff
```

## Requirements

| Artifact | Minimum system |
|:--|:--|
| Standard package | Windows 10 / Server 2016 or later; mainstream Linux distributions (x86_64 / arm64) |
| `-legacy` package | Windows 7 SP1 / Server 2008 R2 SP1 or later; amd64 only |

Standard packages are built with Go 1.22+. The Go runtime no longer supports Windows 7 / 8 / 8.1 or Server 2008 / 2012. On those systems the process crashes during runtime initialization with `0xc0000005`, `PC=0x0`; use the `-legacy` package instead.

The console hosts five slots by **platform + architecture + variant**. Push and automatic update select a package using the CPU architecture and Windows build number reported by the host: Build ≥ 10240 uses the standard package (skip if missing; never falls back to legacy), while builds below 10240 use legacy. Hosts that have not reported these fields receive the amd64 standard package and are marked “unverified”.

Windows 7 / Server 2008 R2 ships with older root certificates. If the server uses HTTPS with a certificate chain from a newer CA, update the root certificates first; otherwise the agent stays stuck at certificate verification.

## Build

Go 1.22 or later is required; the `go` directive in `agent/go.mod` is authoritative. The Windows 7 / Server 2008 R2 compatibility package requires Go 1.20; see “Windows legacy package” below.

```bash
cd agent
go build ./...
go vet ./...
go test ./...
```

These three commands are the verification baseline for this directory and are not repeated in later sections.

Cross-compilation (the server hosts both amd64 and arm64 packages and selects by target architecture):

```bash
# Linux amd64 / arm64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/backup-agent-linux ./cmd/backup-agent
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o dist/backup-agent-linux-arm64 ./cmd/backup-agent
```

```powershell
# Windows amd64 / arm64
$env:CGO_ENABLED = "0"
$env:GOOS = "windows"; $env:GOARCH = "amd64"
go build -trimpath -ldflags "-s -w" -o dist/backup-agent-windows-amd64.exe ./cmd/backup-agent

$env:GOOS = "windows"; $env:GOARCH = "arm64"
go build -trimpath -ldflags "-s -w" -o dist/backup-agent-windows-arm64.exe ./cmd/backup-agent
```

`-trimpath` removes build-machine paths, and `-ldflags "-s -w"` removes the symbol table and debug information, usually reducing size by another 20–30%. The trade-off is that `dlv` and `gdb` can no longer debug the binary directly; build an unstripped copy when production debugging is needed.

### Windows legacy package

Windows 7 / Server 2008 R2 requires a Go 1.20 build. `go.legacy.mod` and `go.legacy.sum` are copies of the main `go.mod` / `go.sum` with only the `go` directive lowered to 1.20 and dependencies kept identical; the main `go.mod` is not changed. Build with `-modfile` pointing at the legacy files:

```powershell
# First use: install and download Go 1.20.14
go install golang.org/dl/go1.20.14@latest
go1.20.14 download

$env:CGO_ENABLED = "0"; $env:GOOS = "windows"; $env:GOARCH = "amd64"
go1.20.14 build -modfile "go.legacy.mod" -trimpath -ldflags "-s -w" `
  -o dist/backup-agent-windows-amd64-legacy.exe ./cmd/backup-agent
```

The output filename always includes `-legacy`; it is amd64-only and uses the same version number as the standard package. Filename and system requirements distinguish the two. `-modfile` and its value must be separated by a space; in PowerShell, `-modfile=go.legacy.mod` is split into two arguments and fails.

**Always build the package directory `./cmd/backup-agent`**: `version_marker.go` in that directory embeds the `backup-agent-version:X.Y.Z` marker used by the server to detect the version and architecture on upload. Building only `main.go` still produces a runnable program, but without the marker the upload form requires the version to be entered manually.

When releasing a new version, update both version strings: `Version` in `internal/config/types.go` (runtime reporting) and `embeddedVersion` in `cmd/backup-agent/version_marker.go` (server detection).

## Configuration

Load order: **command-line flags > environment variables > JSON config file > defaults**.

Config file lookup order:

1. The path passed to `-config` (must exist)
2. The path in `BM_CONFIG_FILE`
3. `backup-agent.json` in the executable's directory
4. Linux: `/etc/backup-agent/backup-agent.json`; Windows: `%ProgramData%\backup-agent\backup-agent.json`

Item 3 means the **directory containing the binary**. When debugging with `go run ./cmd/backup-agent`, the binary lives in a system temporary directory, so a config file in the project directory will not be found; pass `-config` explicitly (or set `BM_CONFIG_FILE`). After compiling and placing the binary in its deployment directory, the flag can be omitted.

Configuration fields:

| Field | Description | Default |
|:--|:--|:--|
| `serverUrl` | Server address; must be an absolute http/https URL | empty |
| `token` | `X-Agent-Token` token | empty |
| `hostname` | Reported hostname; empty uses the system hostname | empty |
| `intervalSec` | Local fallback reporting interval; 0 uses server configuration | 600 |
| `role` | Host role: `database`, `web`, `application`, `file`, `virtualization` | `database` |
| `logFile` | Log file path; empty writes `agent.log` next to the binary | `agent.log` in the same directory |
| `logMaxMb` | Maximum size of one log file (MB); 0 or empty uses the default | 10 |
| `logBackups` | Number of historical log files to keep; 0 or empty uses the default; N files besides the current one | 2 |

The fields map one-to-one to `Local` in `internal/config/types.go`. Two additional notes:

- `intervalSec` is only a **fallback**: the effective interval is chosen in this order: “server-delivered collection profile > server global settings > local `intervalSec` > default 600”. `0` delegates entirely to the server; a negative value fails startup; values `1`–`9` are raised to the minimum of 10
- Invalid `role` values are ignored and fall back to `database`. The server uses the value only on the host's first registration; afterwards the role configured in the console wins and reports never overwrite manual changes

### Environment variables

Every field above has a corresponding environment variable (names are not always identical; use this table). They are convenient for containers and service managers:

| Variable | Corresponds to | Description |
|:--|:--|:--|
| `BM_SERVER_URL` | `-server` / `serverUrl` | Server address |
| `BM_AGENT_TOKEN` | `-token` / `token` | Agent token |
| `BM_HOSTNAME` | `-hostname` / `hostname` | Reported hostname |
| `BM_ROLE` | `-role` / `role` | Host role |
| `BM_INTERVAL_SEC` | `-interval` / `intervalSec` | Reporting interval fallback (seconds) |
| `BM_CONFIG_FILE` | `-config` | Config file path |
| `BM_LOG_FILE` | `-log` / `logFile` | Log file path |
| `BM_LOG_MAX_MB` | `-log-max-mb` / `logMaxMb` | Maximum size of one log file (MB) |
| `BM_LOG_BACKUPS` | `-log-backups` / `logBackups` | Number of historical log files |

A non-integer value for an integer variable fails immediately and names the variable (for example `invalid BM_LOG_MAX_MB: "abc" is not an integer`); it is never silently ignored.

### Example configuration

JSON does not support comments; field meanings are in the table above:

```json
{
  "serverUrl": "",
  "token": "",
  "hostname": "",
  "intervalSec": 0,
  "role": "database",
  "logFile": "",
  "logMaxMb": 10,
  "logBackups": 2
}
```

Typical Linux service configuration (place it in the default location; no flags are needed after installation):

```json
{
  "serverUrl": "http://10.0.0.10:8080",
  "token": "generate on the console Agent Tokens page",
  "role": "database"
}
```

Typical Windows service configuration (pin the hostname explicitly to distinguish multiple instances):

```json
{
  "serverUrl": "https://backup.example.com",
  "token": "generate on the console Agent Tokens page",
  "hostname": "db-prod-02",
  "role": "database"
}
```

`serverUrl` and `token` must not be committed to the repository; inject them through the config file or environment variables during deployment. Windows JSON paths must use double backslashes.

## Run

```bash
# Long-running: periodic HTTP reporting + WebSocket heartbeat/commands + auto-update
./backup-agent

# Collect and report once, then exit
./backup-agent -once

# Print the version
./backup-agent -version
```

Common flags (the Chinese descriptions from `-h` match this table; equivalent environment variables are listed in the same row):

| Flag | Description | Environment variable |
|:--|:--|:--|
| `-server` | Server address | `BM_SERVER_URL` |
| `-token` | Agent token | `BM_AGENT_TOKEN` |
| `-hostname` | Reported hostname; empty uses the system hostname | `BM_HOSTNAME` |
| `-role` | Host role | `BM_ROLE` |
| `-interval` | Local reporting interval fallback (seconds); 0 uses server configuration | `BM_INTERVAL_SEC` |
| `-config` | Config file path | `BM_CONFIG_FILE` |
| `-log` | Log file path; empty writes `agent.log` next to the binary | `BM_LOG_FILE` |
| `-log-max-mb` | Maximum size of one log file (MB); empty uses 10 | `BM_LOG_MAX_MB` |
| `-log-backups` | Historical log files to keep; empty uses 2 (three files total) | `BM_LOG_BACKUPS` |
| `-once` | Collect and report once, then exit | - |
| `-name` | Service name; default `backup-agent` | `BM_SERVICE_NAME` |
| `-displayname` | Service display name; empty derives it from the service name (Linux writes it to the unit `Description`) | `BM_DISPLAY_NAME` |
| `-service` | Run in service mode (no stdout logging); supplied by the service manager. On Windows it must be installed first when used alone | - |
| `-install` | Install and start the system service | - |
| `-uninstall` | Stop and remove the system service | - |
| `-version`, `-v` | Print the version and exit | - |

`-xxx` and `--xxx` are equivalent (Go's `flag` package accepts both). The legacy form `backup-agent run -config xxx.json` is supported: the leading `run` is ignored and the remaining flags are parsed normally.

Long-running mode first fetches `/api/agent/config`, then runs the server-delivered collection profiles. If the fetch fails, it keeps reporting host heartbeats and retries with backoff. When `profile` is `null`, it reports only host heartbeats and no tasks or executions.

## Install as a service

The default service name is `backup-agent` and the default display name is `Backup Monitor Agent`; multiple instances automatically append `(<service name>)`. Use `-displayname` to override it (Windows service display name / Linux unit `Description`). `-install`, `-uninstall` and `-service` all accept `-name <service name>` (or `BM_SERVICE_NAME`) for multiple instances on one host or alignment with existing naming.

- **No duplicate creation** — `-install` checks for an existing service first: on Linux it checks `<name>.service` under `/etc/systemd/system`, `/run/systemd/system`, `/usr/lib/systemd/system` and `/lib/systemd/system` in that order; on Windows it checks the SCM. If found, it reports the existing service and does not overwrite or duplicate it
- **Consistent removal scope** — `-uninstall` searches and removes units in the same directory scope; on Windows it stops and deletes the service by SCM name

Installation records the **absolute path of the binary at install time** in the service definition, so the binary may live in any directory as long as:

- **The path is stable** — do not use `/tmp`, throwaway extraction directories or locations that get cleaned
- **Place it before installing** — after moving or renaming the binary, run `-uninstall` and then `-install`; otherwise the service still points at the old path
- **Keep configuration in a default location** — the service starts with `-service` and the unit has no `-config`, so `backup-agent.json` must be next to the binary or in the system default directory (see the lookup order in “Configuration”)

Recommended deployment directories (reference only, not mandatory):

| Platform | Binary | Configuration | Logs |
|:--|:--|:--|:--|
| Linux | `/opt/backup-agent/backup-agent` or `/usr/local/bin/backup-agent` | Next to the binary or `/etc/backup-agent/backup-agent.json` | `agent.log` next to the binary |
| Windows | `C:\ProgramData\backup-agent\backup-agent.exe` | `C:\ProgramData\backup-agent\backup-agent.json` | `agent.log` next to the binary |

### Linux (systemd)

Root is required. The default unit file is written to `/etc/systemd/system/backup-agent.service`:

```bash
# Install (the unit records the absolute path of this binary)
sudo ./backup-agent -install
systemctl status backup-agent

# Custom service name
sudo ./backup-agent -install -name backup-agent-2
systemctl status backup-agent-2

# Uninstall (stops the service and removes the unit; configuration and logs are kept; add -name for custom names)
sudo ./backup-agent -uninstall
sudo ./backup-agent -uninstall -name backup-agent-2
```

The unit is equivalent to the following (using `<service name>` as the service name):

```ini
[Unit]
Description=<service display name>
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=<absolute path of the binary at install time> -service -name <service name>
Restart=always
RestartSec=10
StandardOutput=null
StandardError=null
SyslogIdentifier=<service name>

[Install]
WantedBy=multi-user.target
```

`StandardOutput` and `StandardError` are both `null`; the agent writes logs only to `agent.log` next to the binary and does not write to journald. `SyslogIdentifier` follows the service name and is reserved for a future switch back to `journal`; `journalctl -t <service name>` can filter by it directly.

### Windows

Run as Administrator. It is recommended to place the binary in `C:\ProgramData\backup-agent\` first and install from there:

```powershell
cd C:\ProgramData\backup-agent
.\backup-agent.exe -install
Get-Service backup-agent

# Custom service name
.\backup-agent.exe -install -name backup-agent-2
Get-Service backup-agent-2

.\backup-agent.exe -uninstall
```

The default service name is `backup-agent`, the default display name is `Backup Monitor Agent` (multiple instances append `(<service name>)`; `-displayname` overrides it), the startup type is Automatic, and the start arguments are `-service -name <service name>`. Re-running `-install` when a service with the same name exists reports it and does not create a duplicate; running without Administrator returns a clear permission error.

## Logs

- By default logs are written to `agent.log` **next to the binary**; override with `-log` / `logFile` / `BM_LOG_FILE`
- Rotation is size-based and both values are configurable:
  - Maximum size per file: `-log-max-mb` / `logMaxMb` / `BM_LOG_MAX_MB`, default **10** (MB)
  - Historical files to keep: `-log-backups` / `logBackups` / `BM_LOG_BACKUPS`, default **2**, i.e. three files total (`agent.log`, `agent.log.1`, `agent.log.2`)
- `0` or an omitted value uses the default; a negative value fails startup
- Log content is in Chinese; fixed prefixes such as `[ws]` are preserved for easy searching
- Service mode does not write to stdout; if the log file cannot be opened, output falls back to stderr for the service manager to collect
- Logs follow the binary; replacing the binary during an upgrade appends to the same files and does not lose history
- On Windows, newly created log files include a UTF-8 BOM so Notepad opens them correctly; Linux does not write a BOM. Old files created before the upgrade have no BOM and may still look garbled in Notepad; open them with VS Code / Notepad++ or delete and regenerate them

## Collection notes

`includeKeywords`, `excludeKeywords`, `artifactCheck`, `artifactMaxDepth`, `manualTriggerMinutes` and similar fields mentioned in this section are **server-delivered collection profile fields** (Console → Agent Management → Profiles), not fields in the local `backup-agent.json`. Built-in defaults apply when the server does not deliver them.

### Linux crontab

The agent reads the current user's `crontab -l`, `/etc/crontab` and `/etc/cron.d/*`, and supports macros such as `@daily`, `@hourly`, `@weekly`, `@monthly`, `@yearly` and `@reboot`. Tasks that match `includeKeywords` / `includePaths` and do not match `excludeKeywords` are collected.

Add a tag above the schedule line to force inclusion and override the task name, stable key and artifact information:

```cron
# backup-monitor: name=Daily database backup;key=db-daily;status=/data/backup/db.status;artifact=/data/backup
0 23 * * * /opt/backup/db.sh
```

Tag values support `;`, `,` or space-separated key-value pairs; quote values that contain spaces:

```cron
# backup-monitor: name="Daily database backup" key=db-daily status=/data/backup/db.status
0 23 * * * /opt/backup/db.sh
```

| Key | Description |
|:--|:--|
| `name` | Task name |
| `key` | Stable task identifier; the same task still matches after a rename |
| `script` | Script path used for status files and process probing |
| `status` / `statusFile` | Explicit status file path |
| `artifact` / `path` / `dir` | Backup artifact file or directory |
| `proc` / `process` | Process keyword; when a local process matches, the task is considered running. Empty disables probing |

On Linux, `proc` matches the full command line (`/proc/<pid>/cmdline`); on Windows it matches the process name (`tasklist`). Precedence: a status file that reports success or failure wins; without a status-file conclusion, a matching process marks the task as running; when no process matches, the original judgement is unchanged. A matching process with neither a status file nor a scheduled time produces no execution record.

Inline tags are also supported:

```cron
15 3 * * * /opt/backup/inline.sh # backup-monitor: inline-task
```

Without a tag, the task key is `cron:<user>:<cron expression>`; with a tag, the tag's `key` wins.

### Windows scheduled tasks

The agent prefers PowerShell `Get-ScheduledTask` + `Get-ScheduledTaskInfo` + `ConvertTo-Json`; on failure it falls back to `schtasks /query /fo CSV /v`. Windows 7 / Server 2008 R2, which only have PowerShell 2.0 and report “cannot load column resources” from schtasks, fall back further to COM `Schedule.Service`. `Daily`, `Weekly`, `Monthly`, `Time`, `Boot` and `Logon` triggers are supported and converted to cron expressions where possible; schedules that cannot be expressed keep a readable description and an empty `scheduleCron`.

Task comments (`Comment`/`Description`) also support `# backup-monitor: ...` tags. Without a tag, filtering uses keywords, script paths and exclude keywords.

### Status files

When a task has `status` / `statusFile`, the agent parses the file line by line (`=` or `:` separator):

```text
start=2026-10-20 15:00:01
end=2026-10-20 15:08:35
size=1.2 GB
transferred=1.1 GB
exit=0
status=success
```

Supported keys include `start`, `end`, `time`, `size`, `transferred`, `exit` and `status`; sizes accept B/KB/MB/GB/TB and binary units.

### Backup artifacts

When `artifactCheck=true` and the task has an artifact path, the agent checks file mtime and size and creates an execution record:

- Status-file execution record: appends `；产物 <path>（<size>）` (the exact Chinese status text)
- Artifact fallback record without a status file: `检测到备份产物：<path>` (the exact Chinese status text)

Directory artifacts are searched recursively up to `artifactMaxDepth` for the newest file; `.status`, `.tmp`, `.part` and `veeam-status.txt` are ignored.

### Veeam

On Linux and Windows the agent looks for `veeamconfig` (`/usr/local/sbin`, `/usr/sbin`, `/opt/veeam`, `PATH` and the Windows installation directory) and parses the `job list`, `session list` and `session info` tables. If Veeam is not installed, it skips silently without affecting other tasks.

On Windows, when `veeamconfig` is unavailable, it also reads `veeam-status.txt`:

- `%ProgramData%\Veeam\Backup\veeam-status.txt`
- `%ProgramData%\Veeam\Endpoint Backup\veeam-status.txt`
- `veeam-status.txt` next to the executable

### State and trigger semantics

- Exit code `0` means `success`
- Windows `0x41301` (267009) means `running`
- Windows `0x41303` (267011, not yet run) produces no execution record
- Windows `0x41306` (267014, terminated) and other non-zero codes mean `failed`
- If the start time deviates from the nearest scheduled time by more than `manualTriggerMinutes`, `trigger` is `manual`; otherwise it is `scheduled`
- The lookback window for the nearest `scheduleCron` time is 7 days, matching the server's judgement

## Self-update

When the WebSocket receives an `update` command:

1. Reply with `update_result` first; `message` is `开始更新` (“starting update”)
2. Download the server binary to a temporary file
3. Verify the response headers and file SHA-256
4. Back up the old binary as `<exe>.bak` (deleted only after the new version reports successfully for the first time)
5. Replace the current binary
6. Restart according to the runtime environment: if submitting the restart fails, restore `.bak` and return an error. Linux systemd restarts are submitted asynchronously; if the new version fails to start, roll back manually. On Windows a supervisor child process verifies startup and rolls back automatically; see “Rollback”

| Runtime mode | Restart method |
|:--|:--|
| Linux systemd service | Match the unit name from `ExecStart` (override with `BM_SERVICE_NAME`), then use `systemd-run` in a separate unit to run `systemctl restart --no-block`, so the restart command is not stopped by systemd along with this unit. If `systemd-run` is unavailable, call `systemctl` directly; a command killed by a signal is treated as “restart submitted” |
| Linux non-service / no unit installed | `exec` replaces the process in place; the PID is unchanged |
| Windows service | The old binary launches a restart supervisor: wait for the old process to exit → wait for `STOPPED` from the SCM → start → verify `RUNNING`; on failure restore `.bak` and restart the old version. The process writes to `update.log` next to the executable. If the supervisor cannot start, fall back to a delayed script (`sc stop` → poll `STOPPED` → `sc start`) |
| Windows non-service | Start a detached new process with the same arguments; the new process waits for the old PID to exit and continues |

**When no service is installed**: on Linux, running with `-service` directly or inside a container makes `systemctl restart` fail because the unit does not exist; the agent automatically falls back to restarting the process directly (log: `未安装系统服务，改为直接重启进程`, “no system service installed; restarting the process directly”) and does not roll back the version. On Windows, an unregistered service does not enter the update flow; `-service` exits during startup.

**Automatic update checks** are controlled by the server-delivered collection profile field `autoUpdateMinutes` (not a local JSON field). Set it in Console → Agent Management → Profiles. When `> 0`, the agent queries `/api/agent/binary/latest?platform=<linux|windows>` at that interval and runs the same update flow when a higher version is found; `0` disables automatic checks and only responds to server pushes. After the interval changes (including from `0` to a non-zero value), the agent restarts timing from the change and checks only after a full new interval. `platform` is supplied automatically based on the runtime platform.

**Failure backoff**: after an automatic update fails, the agent writes “target version + failure time” to `<exe>.update-state` and does not retry the same version automatically for 30 minutes. Manual pushes are not subject to backoff; the record is cleared after a successful update.

Manual pushes do not compare versions: the same version as the current one is still re-delivered and restarted. Automatic checks trigger only when the hosted version is higher.

## Upgrade and rollback

### Upgrade

Two methods with the same effect:

| Method | How |
|:--|:--|
| Server push | Upload the new binary in Console → Agent Update, then click “Push update”; the agent receives the `update` command and performs download, verification, replacement and restart automatically |
| Automatic check | Enable `autoUpdateMinutes` in the collection profile; the agent queries and upgrades at that interval |

Manual upgrade (when server push is unreachable): stop the service → overwrite the original file with the new binary → start the service. The path must be the one registered with the service; otherwise the service still starts the old file.

The `-legacy` package can be uploaded directly to the Windows amd64 legacy slot in the console (the server archives it by Go version) without overwriting the standard package. During a push, Windows 7 / Server 2008 R2 hosts automatically select it. For a manual upgrade, stop the service and overwrite the legacy artifact for the matching architecture.

```bash
# Linux
sudo systemctl stop backup-agent
sudo install -m 0755 backup-agent /opt/backup-agent/backup-agent
sudo systemctl start backup-agent
sudo systemctl status backup-agent
```

```powershell
# Windows (Administrator)
Stop-Service backup-agent
Copy-Item .\backup-agent.exe C:\ProgramData\backup-agent\backup-agent.exe -Force
Start-Service backup-agent
```

### Rollback

Before replacing the binary, self-update renames the old binary to `<exe>.bak` (for example `/opt/backup-agent/backup-agent.bak` or `C:\ProgramData\backup-agent\backup-agent.exe.bak`). In addition:

- If replacing the new binary fails, `.bak` is restored automatically
- If the restart fails: on Linux, a failed restart submission restores `.bak`; if the new version fails to start, roll back manually as described below. On Windows, the supervisor verifies startup and restores automatically
- `.bak` is deleted only after the new version reports successfully for the first time; if it fails to start or crashes after startup, the file remains for manual rollback

Therefore, when the new version fails to start, rename `.bak` back and restart the service:

```bash
# Linux
sudo systemctl stop backup-agent
sudo mv /opt/backup-agent/backup-agent.bak /opt/backup-agent/backup-agent
sudo systemctl start backup-agent
```

```powershell
# Windows (Administrator)
Stop-Service backup-agent
Move-Item C:\ProgramData\backup-agent\backup-agent.exe.bak C:\ProgramData\backup-agent\backup-agent.exe -Force
Start-Service backup-agent
```

After rolling back, change the platform's hosted binary back to the old version in the console to prevent automatic checks from upgrading the host again.

## Troubleshooting

Logs are the first place to look: by default `agent.log` next to the binary; service mode does not write to stdout.

**Exits immediately on startup**

Run `./backup-agent -once` in the foreground to see the error directly. Errors during configuration loading go only to stderr (the log file is not open yet); runtime errors go to `agent.log`. Common causes:

- `server URL is required` — configuration was not loaded, or `serverUrl` is empty
- `invalid BM_LOG_MAX_MB: "abc" is not an integer` — the environment variable is not an integer
- `invalid role "xxx"` — the role is not in the allowed set
- The two most common reasons configuration is not loaded: service mode only checks default paths (see “Install as a service”), or the binary runs from a temporary directory under `go run`

**Crashes immediately with 0xc0000005**

The process exits before reaching `main`; the console or Event Viewer shows `Exception 0xc0000005`, `PC=0x0`, and the top frame is `internal/runtime/syscall/windows_amd64.s`. This is a Go runtime / OS version mismatch: the standard package requires Windows 10 / Server 2016 or later; Windows 7 / Server 2008 R2 must use the `-legacy` package.

**Host does not come online**

1. Is the agent process running: `systemctl status backup-agent` / `Get-Service backup-agent`
2. Does `agent.log` show connection failures, 401/403 or profile fetch failures
3. Is `serverUrl` reachable from that host (`curl <serverUrl>/api/health`)
4. When token enforcement is enabled, does `token` match the value in Console → Agent Management → Agent Tokens

**Service installation fails**

- Linux requires root (`sudo`); Windows requires an Administrator PowerShell
- If the service already exists, the agent reports it explicitly; run `-uninstall` first
- Containers have no systemd, so `-install` does not apply; run `-service` directly under the container orchestrator

**Configuration changes do not take effect**

The service starts with `-service` and without `-config`, so the config file must be in a default lookup location (next to the binary or the system default directory). Restart the service after changes: `systemctl restart backup-agent` / `Restart-Service backup-agent`.

## Related documents

| Document | Content |
|:--|:--|
| [docs/agent-protocol.md](../docs/agent-protocol.md) | Full fields for reporting, profile fetch, WebSocket downstream messages and receipts |
| [docs/manual.md](../docs/manual.md) | Section 2.4 for local debugging, Section 4.3 for production deployment and systemd examples |
| [README.md](../README.md) | Overall project introduction, deployment options and FAQ |

## Known limitations and risks

- Windows `set_cron` does not yet support writing back scheduled tasks and returns a clear Chinese message; Linux supports the current user's crontab, `/etc/crontab` and `/etc/cron.d/*`
- Writing `/etc/crontab` and `/etc/cron.d/*` requires root, and atomic replacement may change the file's SELinux context
- The `schtasks` CSV fallback output depends on the system code page; Chinese task names may be garbled. Prefer the PowerShell path
- Veeam CLI output varies by version and language; the parser has header and position fallbacks, but complex tables may still be missed
- Windows artifacts and scheduled-task times are parsed in the local time zone; multi-time-zone deployments should keep host time zones consistent
- Automatic updates depend on the server-hosted binary and SHA-256; verification failures keep the old binary
