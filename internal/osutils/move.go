package osutils

import (
	"io"
	"os"
	"path/filepath"
)

// Move attempts to use os.Rename and if that fails (such as for a cross-device move),
// it instead copies the source to the destination and then removes the source.
func Move(sourcePath, destPath string) error {
	// Try to rename the source to the destination
	err := os.Rename(sourcePath, destPath)
	if err == nil {
		return nil // Successful move
	}

	// Rename failed, so copy the source to the destination
	err = copyDirOrFile(sourcePath, destPath)
	if err != nil {
		return err
	}

	// Copy successful, remove the original source
	err = os.RemoveAll(sourcePath)
	if err != nil {
		return err
	}

	return nil
}

func copyDirOrFile(sourcePath, destPath string) error {
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}

	if sourceInfo.IsDir() {
		return copyDir(sourcePath, destPath, sourceInfo)
	} else if sourceInfo.Mode().IsRegular() {
		return copyFile(sourcePath, destPath, sourceInfo)
	} else {
		// ignore non-regular files
		return nil
	}
}

func copyDir(sourcePath, destPath string, sourceInfo os.FileInfo) error {
	err := os.MkdirAll(destPath, sourceInfo.Mode())
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(sourcePath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		sourceEntry := filepath.Join(sourcePath, entry.Name())
		destEntry := filepath.Join(destPath, entry.Name())

		err = copyDirOrFile(sourceEntry, destEntry)
		if err != nil {
			return err
		}
	}

	return nil
}

func copyFile(sourcePath, destPath string, sourceInfo os.FileInfo) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, sourceInfo.Mode())
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return nil
}
