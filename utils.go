package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"html/template"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
)

func gitDownload(path, uri string) error {
	var (
		cmdOut []byte
		err    error
	)
	if path == "" {
		log.Fatal("Git: Path must be set")
		os.Exit(1)
	}
	cmdName := "git"
	cmdArgs := []string{"clone", uri, path}
	cmdOut, err = exec.Command(cmdName, cmdArgs...).Output()

	fmt.Println(string(cmdOut))

	return err
}

func ValidateHostName(name string) bool {
	validHostNamePattern := regexp.MustCompile(`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])+$`)
	return validHostNamePattern.MatchString(name)
}

func ValidateMacaddr(mac string) (string, error) {
	validMacaddrPattern := regexp.MustCompile(`^([0-9a-fA-F]{2}[-]){5}([0-9a-fA-F]{2})+$`)
	if !validMacaddrPattern.MatchString(mac) {
		return mac, fmt.Errorf("Invalid mac address %q, it must match %s", mac, validMacaddrPattern)

	}
	return mac, nil
}

func ValidateTemplates(path, extension string) {
	filenames := []string{}
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && filepath.Ext(path) == extension {
			filenames = append(filenames, path)
		}
		return err
	})

	if err != nil {
		log.Fatalln(err)
	}

	if len(filenames) == 0 {
		log.Errorf("There is no %s templates in: %q", extension, path)
		os.Exit(1)
	}

	templates, err = template.ParseFiles(filenames...)
	if err != nil {
		log.Fatalln(err)
	}
}

func listTemplates(path string) {
	files, _ := ioutil.ReadDir(path)
	for _, f := range files {
		fmt.Println(f.Name())
	}
}

func GetHomeDir() string {
	if runtime.GOOS == "windows" {
		return os.Getenv("USERPROFILE")
	}
	return os.Getenv("HOME")
}
