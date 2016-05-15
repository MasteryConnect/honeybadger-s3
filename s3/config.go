package s3

import (
	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var config = &aws.Config{
	Region: aws.String("us-east-1"),
}

var s3conn *s3.S3

func S3() *s3.S3 {
	if s3conn == nil {
		// Try the to load the env credentials
		// AWS_ACCESS_KEY_ID or AWS_ACCESS_KEY
		// AWS_SECRET_ACCESS_KEY or AWS_SECRET_KEY
		config.Credentials = credentials.NewEnvCredentials()
		_, err := config.Credentials.Get()
		sess := session.New()
		if err != nil {
			log.Warnln("No env variables found for access or secret, trying shared provider")
			// No env variables found for access or secret, use shared provider:
			// Filename: if empty uses AWS_SHARED_CREDENTIALS_FILE then
			// $HOME/.aws/credentials
			// Profile: if empty uses AWS_PROFILE then default
			config.Credentials = credentials.NewSharedCredentials("", "")
			// Try now loading shared credentials
			_, err := config.Credentials.Get()
			if err != nil {
				log.Warnln("No shared credentials found, trying ec2 role provider")
				config.Credentials = ec2rolecreds.NewCredentialsWithClient(ec2metadata.New(sess, config))
				_, err := config.Credentials.Get()
				if err != nil {
					log.Warnln("Unable to use ec2 role provder")
					log.Fatalln(err.Error())
				} else {
					log.Info("Using ec2 role provider")
				}
			} else {
				log.Info("Using shared credentials provider")
			}
		} else {
			log.Info("Using env varaibles credentials provider")
		}

		s3conn = s3.New(sess, config)
	}

	return s3conn
}
