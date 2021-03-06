package main

import (
  "os"
  "path"
  "github.com/codegangsta/cli"
  log "github.com/Sirupsen/logrus"






)

func main() {
	for _, f := range os.Args {
		if f == "-D" || f == "--debug" || f == "-debug" {
			os.Setenv("DEBUG", "1")
			initLogging(log.DebugLevel)
		}
	}


	app := cli.NewApp()
	app.Name = path.Base(os.Args[0])
	app.Usage = "Create and serve installation instructions"
	app.Commands = Commands
	app.CommandNotFound = cmdNotFound
	app.Version = VERSION


	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug, D",
			Usage: "Enable debug mode",
		},
		cli.StringFlag{
			EnvVar: "STASIS_HOST_STORAGE_PATH",
			Name:   "storage-path",
			Usage:  "Configures storage path",
		},
	}


	app.Run(os.Args)

}