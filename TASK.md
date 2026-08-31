# Window Maximizer

Original task specification. **User-facing documentation lives in [README.md](README.md).**

The system daemon listens for new top-level windows and maximizes them by rules in `config.yml`.

## Configuration

All configuration is in `config.yml`:

```yaml
apps:
  - name: "MetaTrader" # Logging label only.
    match:
      condition: "OR" # AND or OR. Default is AND.
      title: "MetaTrader" # Window title wildcard / substring.
      process: "*terminal*.exe" # Executable path or file name wildcard / substring.
```

`name` is used in logs. `match.title` and `match.process` identify the window. If both are set, `condition` decides whether both must match (`AND`) or either may match (`OR`). First matching app wins. Broker titles often omit the word MetaTrader, so `OR` (or process-only) is the usual match. MetaTrader 4 is `terminal.exe`; MetaTrader 5 is `terminal64.exe`.

See [README.md](README.md) for matching rules, config file search order, and examples.

## Implementation

### Language

Go (Windows only).

### Architecture

- **Windows service (`WinMax`)** — LocalSystem, auto-start. Supervises per-session workers for active, connected, and disconnected logon sessions.
- **Worker (`winmax worker`)** — internal; runs as the logged-on user. Hooks window show/hide/title/destroy events, matches `config.yml`, calls `ShowWindow(SW_MAXIMIZE)`.

Foreground mode (`winmax foreground`) runs the worker in the current terminal for debugging.

The watcher maximizes the main top-level window only. Child windows, tool windows, and owned dialogs (login / connect) are ignored. If a modal is open, it waits until that dialog closes, then maximizes the owner. If the previous session was closed maximized, a brief maximized flash on the next start is not treated as done — the window is maximized once after startup settles.

Only one worker runs per user session (mutex `Local\WinMax_UserLogonDaemon`).

### Logging

- Watcher: `winmax.log` next to the executable; foreground mode also prints to stdout.
- Service lifecycle: Event Viewer, source `WinMax`.
- Early worker failure: `winmax-worker.err` next to the executable.

## Commands

| Command | Notes |
|---|---|
| `winmax help` | Usage |
| `winmax install` | Install and start service (Administrator). Requires `config.yml` next to `winmax.exe`. |
| `winmax uninstall` | Stop and remove service (Administrator) |
| `winmax reload` | Reload config in the running worker; re-scans open windows |
| `winmax status` | Service state and binary path |
| `winmax foreground` | Run watcher in this terminal (`Ctrl+C` to stop). Optional `--config` / `-c`. |

Full details: [README.md](README.md).
