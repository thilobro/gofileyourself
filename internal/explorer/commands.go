package explorer

import (
	"bytes"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/thilobro/gofileyourself/internal/helper"
)

type CommandFunc func([]string) error

func commandWrapper(commandFunc CommandFunc, command string, minNumArgs int, maxNumArgs int) error {
	parts := strings.Split(command, " ")
	if maxNumArgs != -1 && len(parts) > maxNumArgs+1 {
		return errors.New("[Error] Too many arguments for command '" + command + "'")
	} else if len(parts) < minNumArgs+1 {
		return errors.New("[Error] Not enough arguments for command '" + command + "'")
	} else {
		return commandFunc(parts[1:])
	}
}

func (explorer *Explorer) runCdCommand(parts []string) error {
	cdArgs := append([]string{"query"}, parts...)
	cmd := exec.Command("zoxide", cdArgs...)
	out, _ := cmd.Output()
	out = bytes.TrimFunc(out, func(r rune) bool {
		return r <= 32 || r == 127 // Remove control characters
	})
	if string(out) == "" {
		return errors.New("[Error] No matching directory for '" + strings.Join(cdArgs[1:], " ") + "'")
	} else {
		explorer.setCurrentDirectory(string(out[:]))
	}
	return nil
}

func (explorer *Explorer) runQCommand(parts []string) error {
	explorer.context.App.Stop()
	return nil
}

func (explorer *Explorer) runMkDirCommand(parts []string) error {
	for _, dirName := range parts {
		helper.CreateDirectory(filepath.Join(explorer.context.CurrentPath, dirName))
	}
	explorer.setCurrentDirectory(explorer.context.CurrentPath)
	return nil
}

func (explorer *Explorer) runRenameCommand(parts []string) error {
	_, currentName := explorer.currentList.GetItemText(explorer.currentList.GetCurrentItem())
	currentPath := filepath.Join(explorer.context.CurrentPath, currentName)
	newPath := filepath.Join(explorer.context.CurrentPath, parts[0])
	helper.RenameFile(currentPath, newPath)
	explorer.setCurrentDirectory(explorer.context.CurrentPath)
	return nil
}

func (explorer *Explorer) runTouchCommand(parts []string) error {
	for _, fileName := range parts {
		helper.TouchFile(filepath.Join(explorer.context.CurrentPath, fileName))
	}
	explorer.setCurrentDirectory(explorer.context.CurrentPath)
	return nil
}

func (explorer *Explorer) runMRenameCommand(parts []string) error {
	explorer.renameMarkedFiles()
	return nil
}

func (explorer *Explorer) runCommand(command string) {
	explorer.context.RecentCommands = helper.AppendStringToUniqueList(explorer.context.RecentCommands, command)
	parts := strings.Split(command, " ")
	err := errors.New("[Error] Command '" + command + "' not found")
	switch parts[0] {
	case "cd":
		err = commandWrapper(explorer.runCdCommand, command, 1, 1)
	case "q":
		err = commandWrapper(explorer.runQCommand, command, 0, 0)
	case "mkdir":
		err = commandWrapper(explorer.runMkDirCommand, command, 1, -1)
	case "rename":
		err = commandWrapper(explorer.runRenameCommand, command, 1, 1)
	case "mrename":
		err = commandWrapper(explorer.runMRenameCommand, command, 0, 0)
	case "touch":
		err = commandWrapper(explorer.runTouchCommand, command, 1, -1)
	}
	if err != nil {
		explorer.footer.SetText(err.Error())
	}
}
