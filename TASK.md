# Window Maximizer
The system daemon listens to new open windows and maximizes them by predefined rules.

## Configuration
All configuration is done in the `config.yml` file.

```yaml
apps:
  - name: "MetaTrader" # Logging label only.
    match:
      condition: "OR" # AND or OR. Default is AND.
      title: "MetaTrader" # Window title wildcard / substring.
      process: "*terminal*.exe" # Executable path or file name wildcard / substring.
```

`name` is used in logs. `match.title` and `match.process` identify the window. If both are set, `condition` decides whether both must match (`AND`) or either may match (`OR`). First matching app wins. Broker titles often omit the word MetaTrader, so `OR` (or process-only) is the usual match. MetaTrader 4 is `terminal.exe`; MetaTrader 5 is `terminal64.exe`.

## Implementation

### Language
Code is written in Go.

### Daemon
The system daemon is implemented as a Windows service.
It listens to new open windows and maximizes them by predefined rules.

It maximizes the main top-level window only. Child windows, tool windows, and owned dialogs (login / connect) are ignored. If a modal is open, it waits until that dialog closes, then maximizes the owner. If the previous session was closed maximized, a brief maximized flash on the next start is not treated as done — the window is maximized once after startup settles.

It can be run in console mode with `winmax console` for debugging.

### Logging
All watcher logs are written to `winmax.log` next to the executable.
In console / foreground mode, the same lines are also printed to the console.
The Windows service writes lifecycle messages to the Event Viewer (source `WinMax`).

## Usage
The system daemon can be used with the `winmax` command.

### Help
`winmax help` shows command usage.

### Install
`winmax install` installs and starts the Windows service (Administrator).
`config.yml` must sit next to `winmax.exe`.

### Uninstall
`winmax uninstall` stops and removes the Windows service (Administrator).

### Reload
`winmax reload` reloads `config.yml` in the running watcher.

### Foreground
`winmax foreground` runs the watcher in this terminal with live logs. Stop with `Ctrl+C`.

### Console
`winmax console` is the same as foreground.
