package utils

import (
	"fmt"
	"github.com/mitchellh/go-homedir"
	"io"
	"os"
)

// CreateFolderIfNotExist creates folder and assign permissive permission if it doesn't exist
func CreateFolderIfNotExist(folderPath string) error {
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		return os.Mkdir(folderPath, 7666)
	}
	return nil
}

func CloseReadCloser(closer io.ReadCloser, log func(x string)) {
	if err := closer.Close(); err != nil {
		log("failed to close")
	}
}

func GetSshFolderPath() string {
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return fmt.Sprintf("%v/.ssh", home)
}