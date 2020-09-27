package fixed

import (
	"fmt"
	"os"

	"github.com/mitchellh/go-homedir"
)

func GetConfigFolderPath() string {
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return fmt.Sprintf("%v/.devenv", home)
}
