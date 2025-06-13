# Go File Yourself

## Motivation

This is a ranger FM clone in go. Motivation behind this is that I want to learn go.

## Installation

To install, please run:
```
make install
```

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

Keys:

- `Ctrl-H` - Toggle hidden files
- `Ctrl-C` - Quit
- `Ctrl-F` - Open finder
- `Ctrl-R` - Open finder for recently opened files

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

Commands:

- `:q` - Quit
- `:mkdir <directory>` - Create directory
- `:rename <new name>` - Rename file
- `:mrename` - Bulk rename marked files
- `:touch <file>` - Create file

### Finder

Keys:

- `keyUp/keyDown` - Move cursor down/up
- `Enter` - Open file
- `Esc` - Go back to explorer


## Project Status

This project is under active development. Features and APIs may change.

