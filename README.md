# Window Maximizer

Windows daemon that watches for new top-level windows and maximizes them according to rules in `config.yml`.

Typical use: keep MetaTrader (or any other app) maximized when it opens.

## Features

- Windows service (`WinMax`) that starts automatically at boot
- Per-user session worker so maximize works on the interactive desktop (Session 0 isolation)
- Rule matching by window title and/or process path
- `AND` / `OR` conditions, with `*` / `?` wildcards (plain strings are treated as substrings)
- Skips dialogs, tool windows, and windows that are already maximized
- `reload` without restarting the service
- Console / foreground mode for debugging
- File logging next to the executable

Windows only. Built with Go.

## Requirements

- Windows 10 / 11 (x64)
- Go 1.24+ to build from source
- Administrator rights to install or remove the service

Keep `winmax.exe` and `config.yml` in the same folder. The service worker looks for the config next to the executable.

## Quick start

```powershell
git clone https://github.com/rpanchyk/winmax.git
cd winmax
go test ./...
go build -o winmax.exe ./cmd/winmax
```

Try it without installing a service:

```powershell
.\winmax.exe console
```

Open a matching app (for example MetaTrader). The window should maximize. Stop with `Ctrl+C`.

Install as a service (elevated prompt):

```powershell
.\winmax.exe install
.\winmax.exe status
```

## Configuration

All rules live in `config.yml`.

```yaml
apps:
  - name: "MetaTrader"
    match:
      condition: "OR"
      title: "MetaTrader"
      process: "terminal64.exe"
```

| Field | Required | Meaning |
|---|---|---|
| `name` | yes | Label used only in logs |
| `match.condition` | no | `AND` (default) or `OR` |
| `match.title` | one of title/process | Window title pattern |
| `match.process` | one of title/process | Executable path or file name pattern |

`match` may also be written as a list of fields:

```yaml
apps:
  - name: "MetaTrader"
    match:
      - condition: "OR"
      - title: "MetaTrader"
      - process: "terminal64.exe"
```

### Matching rules

- Comparison is case-insensitive.
- A pattern without `*` or `?` is a substring. `MetaTrader` matches `MetaTrader 5` and `MetaTrader 4`. `terminal64.exe` matches `C:\Program Files\MetaTrader 5\terminal64.exe`.
- Explicit wildcards: `*` any sequence (including `\`), `?` one character. Example: `*terminal64.exe*`.
- `AND` — every field you set must match.
- `OR` — either `title` or `process` may match (only when both are set).
- If only `title` or only `process` is set, that field alone is enough.
- If several apps match the same window, the first entry in `apps` wins.

Broker terminals often use a custom title that does not contain `MetaTrader` (account number, company name, symbol). In that case use `condition: "OR"`, or match `process` only.

```yaml
apps:
  - name: "MetaTrader"
    match:
      condition: "OR"
      title: "MetaTrader"
      process: "terminal64.exe"
```

### Config file location

Resolved in this order:

1. `--config` / `-c` (console, foreground, and worker only)
2. `WINMAX_CONFIG` environment variable
3. `config.yml` next to `winmax.exe`
4. `config.yml` in the current working directory
5. `%LOCALAPPDATA%\winmax\config.yml`

After editing the file, apply it without a restart:

```powershell
.\winmax.exe reload
```

## Commands

```text
winmax help         show help
winmax install      install and start the Windows service (Administrator)
winmax uninstall    stop and delete the Windows service (Administrator)
winmax reload       reload config.yml in the running watcher
winmax foreground   run attached to this terminal with live logs (Ctrl+C)
winmax console      same as foreground
winmax status       show service state and binary path

winmax console --config C:\path\config.yml
```

`install` / `uninstall` must be run from an elevated prompt. `config.yml` must sit next to `winmax.exe`. `install` restarts the service if it is already running and waits until the desktop worker is alive. `status` does not need Administrator.

Only one watcher runs per user session. Starting `console` while the service worker is already running will fail with “already running”.

## How it works

A Windows service cannot see or resize windows on the user’s desktop (Session 0 isolation). WinMax therefore uses two processes:

1. **Service (`WinMax`)** — LocalSystem, auto-start. Watches logon sessions (active, connected, and disconnected) and starts a worker in each one. Logs to Event Viewer (source `WinMax`).
2. **Worker (`winmax worker`)** — runs as the logged-on user. Hooks window show / title-change events (`SetWinEventHook`), matches `config.yml`, and calls `ShowWindow(SW_MAXIMIZE)`. Logs to `winmax.log`.

`console` and `foreground` skip the service and run the watcher in the current terminal.

Windows that are not top-level, not visible, have no maximize box, or are tool windows are ignored. Already-maximized windows are left alone.

## Logging

- Watcher (`console`, `foreground`, service worker): append to `winmax.log` next to `winmax.exe`.
- `console` / `foreground` also print to stdout.
- Windows service lifecycle: Event Viewer, source `WinMax` (does not write the log file, so LocalSystem and the user do not share it).
- If the worker exits before it can open the log, the reason is appended to `winmax-worker.err` next to the exe.

Example:

```text
winmax: 2026/08/31 07:25:38 loaded config C:\Tools\winmax\config.yml
winmax: 2026/08/31 07:25:38 watching new windows for: [MetaTrader [AND title=MetaTrader process=terminal64.exe]]
winmax: 2026/08/31 07:25:38 maximized app=MetaTrader hwnd=0x5d05a2 pid=51848 title="..." exe=C:\Program Files\MetaTrader 5\terminal64.exe
```

## Build and test

```powershell
go test ./...
go build -o winmax.exe ./cmd/winmax
```

The project is Windows-only (`//go:build windows` on service, Win32, and session code). Tests that would install a service or maximize real windows are not included.

## Limitations

- Windows desktop apps only (not a general Unix daemon).
- The service must be installed with Administrator rights.
- Place the binary in a writable folder if you want `winmax.log` next to it (for example not under `Program Files` unless you adjust ACLs).
- `reload` signals the watcher in this session (`Local\\WinMax_Reload`) and, when available, `Global\\WinMax_Reload`. If nothing is running, it reports that the daemon is not running.

## License

[MIT](LICENSE)
