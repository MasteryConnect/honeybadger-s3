package main

import (
	hb "github.com/MasteryConnect/honeybadger-s3/honeybadger"
	"github.com/MasteryConnect/honeybadger-s3/s3"
	log "github.com/Sirupsen/logrus"
	"strings"
	"time"
)

type Context struct {
	S3bucket           string
	S3prefix           string
	HoneybadgerKey     string
	ProjectIncludeList string
	LastRun            string
	RunData            *s3.RunData
	UploadedFiles      []string
	RateLimit          *hb.RateLimit
	NoticeLimit        int
}

func NewContext(bucket, directory, projects, key, lastRun string, noticeLimit int) *Context {
	return &Context{
		S3bucket:           bucket,
		S3prefix:           directory,
		ProjectIncludeList: projects,
		HoneybadgerKey:     key,
		LastRun:            lastRun,
		RateLimit: &hb.RateLimit{
			Limit:     0,
			Remaining: 0,
			Reset:     0,
			ZeroHits:  0,
		},
		NoticeLimit: noticeLimit,
	}
}

func backup(ctx *Context) {
	log.WithFields(log.Fields{"bucket": ctx.S3bucket, "prefix": ctx.S3prefix, "projects": ctx.ProjectIncludeList}).Info("Backup ctx: ")

	// s3.FindAllFailedUploads()

	s3.CleanUpFailedUploads(ctx.S3bucket, ctx.S3prefix)

	runNewBackup(ctx)

	if len(ctx.UploadedFiles) > 0 {
		log.Info("List of uploaded files:")
	}
	for _, v := range ctx.UploadedFiles {
		log.Info(v)
	}
}

func runNewBackup(ctx *Context) {
	// Get the RunData, including last run for now.
	ctx.RunData = s3.NewRunData(ctx.S3bucket, ctx.S3prefix+"/honeybadger-s3-run-data.txt", ctx.LastRun)

	// Get a list of honeybadger projects, filter to only those we want to backup
	projects := hb.NewProjects(ctx.ProjectIncludeList, ctx.HoneybadgerKey, ctx.RateLimit)
	// Create the project upload
	s3Projects := s3.NewUpload(ctx.S3bucket, constructS3FilePath(ctx.S3prefix, "projects"))
	err := s3Projects.CreateUpload()
	if err != nil {
		s3Projects.HandleError(err)
		return
	}
	for project, more := projects.Next(); more; project, more = projects.Next() {
		log.WithFields(log.Fields{"project": project.Name}).Info("Backing up")
		err := backupProject(ctx, project, s3Projects)
		if err != nil {
			return
		}
	}
	// Complete the project uploads
	location, err := s3Projects.CompleteUpload()
	if err != nil {
		s3Projects.HandleError(err)
		return
	}
	err = ctx.RunData.SaveNextRun()
	if err != nil {
		log.Fatal(err)
	}
	ctx.UploadedFiles = append(ctx.UploadedFiles, location)
}

func backupProject(ctx *Context, project *hb.Project, s3Projects *s3.Upload) error {
	// Create the fault upload
	s3Faults := s3.NewUpload(ctx.S3bucket, constructS3FilePath(ctx.S3prefix, project.Name, "faults"))
	err := s3Faults.CreateUpload()
	if err != nil {
		s3Faults.HandleError(err)
		return err
	}
	// Create the notice upload
	s3Notices := s3.NewUpload(ctx.S3bucket, constructS3FilePath(ctx.S3prefix, project.Name, "notices"))
	err = s3Notices.CreateUpload()
	if err != nil {
		s3Notices.HandleError(err)
		return err
	}

	// Get the projects faults
	lastRunTimestamp, err := ctx.RunData.GetPrevTimestamp(project.Name)
	if err != nil {
		s3Faults.HandleError(err) // FIXME: Should abort all uploads
		return err
	}
	faults := hb.NewFaults(project.Id, ctx.HoneybadgerKey, lastRunTimestamp, ctx.RateLimit)
	faultCount := 0
	for fault, more := faults.Next(); more; fault, more = faults.Next() {
		faultCount++
		log.WithFields(
			log.Fields{
				"count": faultCount,
				"total": project.FaultCount},
		).Info("Faults")
		err := backupFault(ctx, fault, s3Faults, s3Notices, faultCount, project.FaultCount, lastRunTimestamp)
		if err != nil {
			return err
		}
	}
	if faultCount == 0 {
		log.Info("No faults to backup")
	}
	// Complete the notice uploads
	noticesLocation, err := s3Notices.CompleteUpload()
	if err != nil {
		s3Notices.HandleError(err)
		return err
	}
	// Complete the fault uploads
	faultsLocation, err := s3Faults.CompleteUpload()
	if err != nil {
		s3Faults.HandleError(err)
		return err
	}
	// Upload this project
	err = s3Projects.Upload(project)
	if err != nil {
		s3Projects.HandleError(err)
		return err
	}
	ctx.UploadedFiles = append(ctx.UploadedFiles, faultsLocation, noticesLocation)

	return err
}

func backupFault(ctx *Context, fault *hb.Fault, s3Faults *s3.Upload, s3Notices *s3.Upload, faultCount, faultTotal int, lastRunTimestamp int64) error {
	// Get the projects faults
	notices := hb.NewNotices(fault.ProjectId, fault.Id, ctx.HoneybadgerKey, lastRunTimestamp, ctx.RateLimit)

	noticeCount := 0
	for notice, more := notices.Next(); more && (ctx.NoticeLimit > noticeCount); notice, more = notices.Next() {
		noticeCount++
		if fault.NoticesCount < 100 || noticeCount%100 == 0 {
			log.WithFields(
				log.Fields{
					"fault count":  faultCount,
					"fault total":  faultTotal,
					"notice count": noticeCount,
					"notice total": fault.NoticesCount},
			).Info("Notices")
		}
		// Upload this notice
		err := s3Notices.Upload(notice)
		if err != nil {
			s3Notices.HandleError(err)
			return err
		}
	}
	// Upload this fault
	err := s3Faults.Upload(fault)
	if err != nil {
		s3Faults.HandleError(err)
		return err
	}

	return err
}

func constructS3FilePath(s3Prefix string, names ...string) string {
	t := time.Now()
	var parsedNames []string
	for _, v := range names {
		parsedNames = append(parsedNames, strings.Replace(v, " ", "_", -1))
	}
	parsedNames = append(parsedNames, t.Format("20060102150405"))
	suffix := strings.Join(parsedNames, "-") + ".json"
	if len(s3Prefix) < 1 {
		return suffix
	} else {
		return s3Prefix + "/" + suffix
	}
}
