package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	//"net/mail"
	"net/smtp"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

)

func HandleRequest(event events.SimpleEmailEvent) error {
	mailBucket := os.Getenv("MAILREDIR_S3_BUCKET")
	//fmt.Printf("mailBucket: %v\n", mailBucket)
	// Create our AWS SDK configuration and clients
	cfg := aws.NewConfig()
	sess, err := session.NewSession(cfg)
	if err != nil {
		return fmt.Errorf("could not create session: %w", err)
	}

	s3Client := s3.New(sess)

	for _, record := range event.Records {
		fmt.Printf("record.SES.Mail.MessageID: %v\n", record.SES.Mail.MessageID)
		// Retrieve mail contents from S3
		obj, err := s3Client.GetObject(&s3.GetObjectInput{
			Bucket: aws.String(mailBucket),
			Key:    aws.String(record.SES.Mail.MessageID),
		})
		if err != nil {
			return fmt.Errorf("could not get object: %w", err)
		}

		rawEmail, err := ioutil.ReadAll(obj.Body)
		if err != nil {
			log.Fatal(err)
		}

		fromAddress := "default@default.com"
		re := regexp.MustCompile(`(?m)^From: .*<(.+)>`)
		matches := re.FindSubmatch(rawEmail)
		if matches != nil {
			fromAddress = string(matches[1])
		}
		fmt.Printf("fromAddress: %v\n", fromAddress)
		if err != nil {
			return fmt.Errorf("Failed to parse address: %w", err)
		}

		smtpServerHost := os.Getenv("MAILREDIR_SMTP_SERVER_HOST")
		smtpServerPort := os.Getenv("MAILREDIR_SMTP_SERVER_PORT")

		// Send the email via SMTP
		err = smtp.SendMail(smtpServerHost+":"+smtpServerPort, nil, fromAddress, []string{"spam@spam.com"}, rawEmail)
		if err != nil {
			return fmt.Errorf("Failed to send e-mail: %w", err)
		}

		/* 			// Delete from bucket if everything worked
		   			_, err = s3Client.DeleteObject(&s3.DeleteObjectInput{
		   				Bucket: aws.String(mailBucket),
		   				Key:    aws.String(record.SES.Mail.MessageID),
		   			})
		   			if err != nil {
		   				return nil, fmt.Errorf("could not delete email from s3: %w", err)
		   			}
		*/
	}

	return nil
}

func main() {
	lambda.Start(HandleRequest)
}
