package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"os"
)

func init() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:     true,
	})
	// log.SetLevel(log.DebugLevel)
	log.SetLevel(log.InfoLevel)
}

func main() {
	app := cli.NewApp()
	app.Name = "honeybadger-s3"
	app.Version = "1.0"
	app.Usage = `
   backup honeybadger.io faults to AWS S3.

   For S3 access credentials, one of the following is required:
   1. set up the following environment variables:
   AWS_ACCESS_KEY_ID or AWS_ACCESS_KEY
   AWS_SECRET_ACCESS_KEY or AWS_SECRET_KEY
   2. set up ~/.aws/credentials (shared credentials)
   3. run from an ec2 machine and user that has permission to S3 (ec2 role)
	`
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "s3-bucket, b",
			Usage:  "AWS S3 bucket to backup to",
			EnvVar: "S3_BUCKET",
		}, cli.StringFlag{
			Name:   "s3-directory, d",
			Usage:  "(optional) the directory in the AWS S3 bucket to back up to",
			EnvVar: "S3_DIRECTORY",
		}, cli.StringFlag{
			Name:   "projects, p",
			Usage:  "(optional) comma separated list of projects to backup. If not set, all projects are backed up",
			EnvVar: "PROJECTS",
		}, cli.StringFlag{
			Name:   "honeybadger-key, k",
			Usage:  "your Honeybadger.io API key",
			EnvVar: "HB_API_KEY",
		}, cli.StringFlag{
			Name:   "last-run, l",
			Usage:  "the last time this process ran, the time from which this will search for new faults. Use the following format: <year><month><day><hour><minute><second> e.g. 20150430140508",
			EnvVar: "LAST_RUN",
		},
	}
	app.Action = func(c *cli.Context) {
		if len(c.String("s3-bucket")) <= 0 {
			log.Fatal("s3-bucket argument is required!")
		}
		if len(c.String("honeybadger-key")) <= 0 {
			log.Fatal("honeybadger-key argument is required!")
		}
		backup(
			&Context{
				S3bucket:           c.String("s3-bucket"),
				S3prefix:           c.String("s3-directory"),
				ProjectIncludeList: c.String("projects"),
				HoneybadgerKey:     c.String("honeybadger-key"),
				LastRun:            c.String("last-run"),
			},
		)
	}

	app.Run(os.Args)
}
