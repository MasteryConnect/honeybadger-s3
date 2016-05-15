package s3

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/service/s3"
)

const MIN_BYTES = 5 * 1024 * 1024 // Minimum multipart upload size 5 MB

type Upload struct {
	UploadId       *string
	Bucket         string
	Key            string
	PartNumber     int64
	HasData        bool // Did we call Upload() at least once
	Body           *bytes.Buffer
	CompletedParts []*s3.CompletedPart
}

type RunData struct {
	Bucket            string
	Key               string
	Loaded            bool  // S3 data was loaded
	OverrideTimestamp int64 // Overrides all other timestamps
	PrevTimestamp     map[string]int64
	NextTimestamp     map[string]int64
}

func NewUpload(bucket, key string) *Upload {
	return &Upload{Bucket: bucket, Key: key, Body: bytes.NewBuffer([]byte{})}
}

func NewRunData(bucket, key, lastRun string) *RunData {
	if lastRun != "" {
		timestamp, err := time.Parse("20060102150405", lastRun)
		if err != nil {
			log.Fatal(err)
		}
		log.WithFields(log.Fields{
			"last run string": lastRun,
			"last run":        timestamp,
		}).Debug("run data")
		return &RunData{Bucket: bucket, Key: key, OverrideTimestamp: timestamp.Unix(), PrevTimestamp: make(map[string]int64), NextTimestamp: make(map[string]int64)}
	} else {
		return &RunData{Bucket: bucket, Key: key, PrevTimestamp: make(map[string]int64), NextTimestamp: make(map[string]int64)}
	}
}

// Create the multipart upload
func (p *Upload) CreateUpload() error {
	params := &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(p.Bucket), // Required
		Key:         aws.String(p.Key),    // Required
		ContentType: aws.String("application/json"),
	}
	resp, err := S3().CreateMultipartUpload(params)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"aws_response": awsutil.Prettify(resp),
	}).Debug("response")

	// Set the upload id used when performing all uploads to this one file/object
	p.UploadId = resp.UploadId

	return err
}

// Save a honeybadger project to the multipart upload
// S3's multipart upload requires that each part (except for the last part)
// be a minimum of 5 MB's in size. The last part, whether that is the only
// part or the last of many,  can be any size
func (p *Upload) Upload(hbRecord interface{}) error {
	b, err := json.Marshal(hbRecord)
	if err != nil {
		return err
	}
	p.Body.Write(b)
	p.HasData = true
	// S3's multipart upload requires that each part (except for the last part)
	// be a minimum of 5 MB's in size. The last part, whether that is the only
	// part or the last of many,  can be any size
	if p.Body.Len() >= MIN_BYTES {
		p.flush()
	}
	return err
}

func (p *Upload) flush() error {
	p.PartNumber++
	params := &s3.UploadPartInput{
		Bucket:     aws.String(p.Bucket),    // Required
		Key:        aws.String(p.Key),       // Required
		PartNumber: aws.Int64(p.PartNumber), // Required
		UploadId:   p.UploadId,              // Required
		Body:       bytes.NewReader(p.Body.Bytes()),
	}

	resp, err := S3().UploadPart(params)
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"aws_response": awsutil.Prettify(resp),
	}).Debug("response")

	// Add completed part
	p.CompletedParts = append(p.CompletedParts,
		&s3.CompletedPart{
			ETag:       resp.ETag,
			PartNumber: aws.Int64(p.PartNumber),
		})

	p.Body.Reset() // Start buffering again
	return err
}

// Complete the multipart upload of honeybadger projects if there is at least
// one project to upload. Abort the upload if no projects need to be uplaoded
func (p *Upload) CompleteUpload() (string, error) {
	if p.HasData {
		// Write any remaining bytes to S3 before closing the upload. There may be
		// some left to write if we didn't finish exactly on a 5 MB chunk
		if p.Body.Len() > 0 {
			p.flush()
		}
		params := &s3.CompleteMultipartUploadInput{
			Bucket:   aws.String(p.Bucket), // Required
			Key:      aws.String(p.Key),    // Required
			UploadId: p.UploadId,           // Required
			MultipartUpload: &s3.CompletedMultipartUpload{
				Parts: p.CompletedParts,
			},
		}
		resp, err := S3().CompleteMultipartUpload(params)

		if err != nil {
			return p.FileLocation(), err
		}
		log.WithFields(log.Fields{
			"aws_response": awsutil.Prettify(resp),
		}).Debug("response")
	} else {
		p.AbortUpload()
	}
	return p.FileLocation(), nil
}

// Complete the multipart upload of honeybadger projects
func (p *Upload) AbortUpload() {
	abort(aws.String(p.Bucket), aws.String(p.Key), p.UploadId)
}

func abort(bucket, key, uploadId *string) {
	params := &s3.AbortMultipartUploadInput{
		Bucket:   bucket,   // Required
		Key:      key,      // Required
		UploadId: uploadId, // Required
	}
	_, err := S3().AbortMultipartUpload(params)

	if err != nil {
		log.WithFields(log.Fields{
			"bucket": bucket,
			"key":    key,
		}).Error(err)
	}
}

func (p *Upload) HandleError(err error) {
	log.WithFields(log.Fields{
		"bucket": p.Bucket,
		"key":    p.Key,
	}).Error(err)
	p.AbortUpload()
}

func (p *Upload) FileLocation() string {
	return p.Bucket + "/" + p.Key
}

// Clean up any unfinished uploads
func CleanUpFailedUploads(bucket, prefix string) {
	params := &s3.ListMultipartUploadsInput{
		Bucket: aws.String(bucket), // Required
		Prefix: aws.String(prefix),
	}
	cleanedCount := 0
	err := S3().ListMultipartUploadsPages(params,
		func(page *s3.ListMultipartUploadsOutput, lastPage bool) bool {
			for _, u := range page.Uploads {
				cleanedCount++
				abort(page.Bucket, u.Key, u.UploadId)
			}
			return !lastPage
		})

	if err != nil {
		log.WithFields(log.Fields{
			"bucket": bucket,
			"prefix": prefix,
		}).Error(err)
	}
	log.WithFields(log.Fields{
		"count": cleanedCount,
	}).Info("Cleaned up failed uploads")
}

// Clean up any unfinished uploads
func FindAllFailedUploads() {
	var params *s3.ListBucketsInput
	resp, err := S3().ListBuckets(params)
	if err != nil {
		log.Fatal(err)
		return
	}

	uploadCount := 0
	totalSize := int64(0)
	for _, b := range resp.Buckets {
		count := 0
		size := int64(0)

		params := &s3.ListMultipartUploadsInput{
			Bucket: b.Name, // Required
		}

		err := S3().ListMultipartUploadsPages(params,
			func(page *s3.ListMultipartUploadsOutput, lastPage bool) bool {
				for _, u := range page.Uploads {
					count++
					params := &s3.ListPartsInput{
						Bucket:   page.Bucket,
						Key:      u.Key,
						UploadId: u.UploadId,
					}
					resp, err := S3().ListParts(params)
					if err != nil {
						log.Fatal(err)
						return false
					}
					for _, p := range resp.Parts {
						size = size + *p.Size
					}
				}
				return !lastPage
			})

		if err != nil {
			log.Fatal(err)
		}
		log.WithFields(log.Fields{
			"bucket": *b.Name,
			"found":  count,
			"size":   size,
		}).Info("Checked")
		uploadCount = uploadCount + count
		totalSize = totalSize + size
	}

	log.WithFields(log.Fields{
		"total count": uploadCount,
		"total size":  totalSize,
	}).Info("failed uploads")
}

// Get the previous timestamp for project with name projectName. This will
// read in the saved run data from S3 if an override timestamp has not
// been specified. If the run data has a timestamp for the project, it will
// be saved to the RunData PrevTimestamp map and returned going forward.
// If there is no timestamp in the run data, then the default 0 value is
// saved to the RunData.PrevTimestamp map and retured.
func (r *RunData) GetPrevTimestamp(projectName string) (ts int64, err error) {
	projectName = strings.ToLower(projectName)
	if ts, ok := r.PrevTimestamp[projectName]; ok {
		return ts, err
	} else if r.OverrideTimestamp == 0 {
		if !r.Loaded {
			// We haven't loaded the run data from S3 before, so load it
			params := &s3.GetObjectInput{
				Bucket: aws.String(r.Bucket), // Required
				Key:    aws.String(r.Key),    // Required
			}
			resp, err := S3().GetObject(params)
			if err != nil {
				if strings.HasPrefix(err.Error(), "NoSuchKey") {
					// This is the first time we've tried loading the run data, so it
					// doesn't exist in S3 yet. All timestamps should default to 0
					r.PrevTimestamp[projectName] = 0 // default
				} else { // Otherwise some other error occurred
					return ts, err
				}
			} else {
				log.WithFields(log.Fields{
					"aws_response": awsutil.Prettify(resp),
				}).Debug("response")

				scanner := bufio.NewScanner(resp.Body)
				// Read each projects timestamp
				for scanner.Scan() {
					projectTs := strings.Split(scanner.Text(), ":")
					if len(projectTs) > 0 { // Ignore empty lines
						timestamp, _ := strconv.ParseInt(strings.TrimSpace(projectTs[1]), 10, 64)
						// Clean up project name with ToLower and Trim in case it was
						// manually edited
						r.PrevTimestamp[strings.ToLower(strings.TrimSpace(projectTs[0]))] = timestamp
					}
				}
			}
			r.Loaded = true
		} else {
			// We've previously loaded the run data from S3 however a project with
			// the name projectName did not exist i.e. it's the first time we've
			// tried backing it up. So use the default 0 timestamp
			r.PrevTimestamp[projectName] = 0 // default
		}
	} else {
		// An override timestamp was passed in, so use that for all projects
		r.PrevTimestamp[projectName] = r.OverrideTimestamp
	}
	// Get the previous timestamp to return
	ts = r.PrevTimestamp[projectName]
	// This is the first time we've called this function for projectName, set
	// the next timestamp
	r.NextTimestamp[projectName] = time.Now().Unix()

	log.WithFields(log.Fields{
		"previous run": time.Unix(ts, 0),
		"next run":     time.Unix(r.NextTimestamp[projectName], 0),
	}).Info("run data")

	return ts, err
}

// Save all of the RunData.NextTimetamp's to S3 for the next run to use
// The format is:
// project-name-1:next-timestamp1
// project-name-2:next-timestamp2
func (r *RunData) SaveNextRun() error {
	buf := bytes.NewBufferString("")
	projectNext := map[string]bool{}
	for project, nextTs := range r.NextTimestamp {
		projectNext[project] = true
		buf.WriteString(project)
		buf.WriteString(":")
		buf.WriteString(strconv.FormatInt(nextTs, 10))
		buf.WriteString("\n")
	}
	// Any project whose previous timestamp was found in S3, and never had
	// GetPrevTimestamp called i.e. wasn't backed up during this run, save back
	// their timestamp to S3 so it is not lost
	for project, nextTs := range r.PrevTimestamp {
		if !projectNext[project] {
			buf.WriteString(project)
			buf.WriteString(":")
			buf.WriteString(strconv.FormatInt(nextTs, 10))
			buf.WriteString("\n")
		}
	}
	params := &s3.PutObjectInput{
		Bucket: aws.String(r.Bucket), // Required
		Key:    aws.String(r.Key),    // Required
		Body:   bytes.NewReader(buf.Bytes()),
	}
	resp, err := S3().PutObject(params)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"aws_response": awsutil.Prettify(resp),
	}).Debug("response")

	return err
}
