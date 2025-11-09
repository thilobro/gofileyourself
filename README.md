# Go File Yourself

## Motivation

This is a [Ranger FM](https://github.com/ranger/ranger/) clone in go. Motivation behind this is that I want to learn go.

## Installation

To install, please run:
```
make install
```

Requires `zoxide`.

## Run

To start, please run:

```
gofileyourself
```

## Usage

gofileyourself is a terminal-based file manager with vim-like keybindings:

### General

Flags:

- `-h` - Show help
- `--debug` - Print debug log to debug.log
- `--choosefiles=<file>` - Use as a file chooser that writes selected files to the given file
- `--selectfile=<file>` - Select the given file path when opening the file manager
- `--config=<path>` - Default is `~/.gofindyourself.yaml`
- `--startwidget=<widget>` - Start widget, default is `"explore"`, other possibilities are `"find"`, `"findrecent"`, `"grep"`

Keys:

- `Ctrl-S` - Toggle hidden files
- `Ctrl-C` - Quit
- `Ctrl-F` - Open finder
- `Ctrl-R` - Open finder for recently opened files
- `Ctrl-G` - Open finder with full text search (experimental)

### Explorer

Keys:

- `j/k` - Move cursor down/up
- `h/l` - Go to parent directory / Enter directory or open file
- `Ctrl-D/U` - Move cursor down/up (half list)
- `/` - Search in current directory
- `q` - Quit
- `S` - Quit and jump to last directory
- `r` - Cycle through recently opened files
- `R` - Cycle backwards through recently opened files
- `yy` - Yank selected file or directory
- `pp` - Paste yanked file or directory
- `dd` - Delete selected file
- `DD` - Delete selected file or directory
- `mm` / `M` - Toggle mark file / directory
- `mu` - Unmark all files / directories
- `md` - Delete marked files
- `mD` - Delete marked files / directories
- `my` - Yank marked files
- `mp` - Paste marked files
- `A<key>` - Set anchor for key
- `a<key>` - Jump to anchor for key
- `keyUp/keyDown` - Cycle through last searches or commands

Commands:

- `:q` - Quit
- `:mkdir <directory>` - Create directory
- `:rename <new name>` - Rename file
- `:mrename` - Bulk rename marked files
- `:touch <file>` - Create file
- `:cd <path>` - Change path (`zoxide` supported)

### Finder

Keys:

- `keyUp/keyDown` - Move cursor down/up
- `Enter` - Open file
- `Esc` - Go back to explorer

### Config

You can set a config file with `--config=<path>`. The default is `~/.gofindyourself.yaml`. If no config file is found, a default config used:
```yaml
history_len: 50
theme_path: "{$HOME}/.gofileyourself_theme.yaml"
```

If no theme config is found, the default theme is used:
```yaml
bg0: "#282828"
bg1: "#3c3836"
fg0: "#fbf1c7"
fg1: "#ebdbb2"
palette0: "#928374"
palette1: "#fb4934"
palette2: "#b8bb26"
palette3: "#fabd2f"
palette4: "#83a598"
palette5: "#d3869b"
palette6: "#8ec07c"
palette7: "#fe8019"
palette8: "#000000"
```

## Neovim Plugin

For a basic Neovim plugin, please check out [gofindyourself.nvim](https://github.com/thilobro/gofindyourself.nvim).


## Project Status

This project is under active development. Features and APIs may change.

