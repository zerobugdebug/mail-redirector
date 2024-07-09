package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/smtp"
	"os"

	"github.com/DusanKasan/parsemail"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	defaultFromEmail = "nobody@nobody.none"
	defaultToEmail   = "nobody@nobody.none"
)

func getEmailValue(email string, emailMap map[string]string) string {
	// Iterate over the emails until match a key in the map
	value, exists := emailMap[email]
	if exists {
		return value
	}

	// Return empty string if no key was found
	return ""
}

func HandleRequest(event events.SimpleEmailEvent) error {
	//Init the e-mail key-value map
	emailMapJson := os.Getenv("MAILREDIR_EMAIL_MAP")
	//fmt.Printf("emailMapJson: %v\n", emailMapJson)
	// Define a map to hold the parsed JSON
	emailMap := make(map[string]string)

	// Unmarshal the JSON into the map
	err := json.Unmarshal([]byte(emailMapJson), &emailMap)
	if err != nil {
		return fmt.Errorf("error while parsing EMAIL_MAP: %w", err)
	}

	mailBucket := os.Getenv("MAILREDIR_S3_BUCKET")
	// Create AWS SDK configuration and clients
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

		rawEmail, err := io.ReadAll(obj.Body)
		if err != nil {
			log.Fatal(err)
		}

		/* 		//Parsing from address from FROM: header
		   		fromAddress := defaultFromEmail
		   		re := regexp.MustCompile(`(?m)^From: .*<(.+)>`)
		   		matches := re.FindSubmatch(rawEmail)
		   		if matches != nil {
		   			fromAddress = string(matches[1])
		   		}
		   		fmt.Printf("fromAddress: %v\n", fromAddress)
		   		if err != nil {
		   			return fmt.Errorf("failed to parse address: %w", err)
		   		}

		   		//Parsing to address from TO: header
		   		// Define a regex to match the "To:" field in the email
		   		regex := regexp.MustCompile(`(?s)\nTo: ([\s\S]*?)(?:\n[^ \t\n\r]|$)`)
		   		match := regex.FindStringSubmatch(string(rawEmail))
		   		fmt.Printf("match: %v\n", match)

		   		var emails []string
		   		if len(match) > 1 {
		   			toField := match[1]
		   			// Define a regex to match the emails within angle brackets
		   			emailRegex := regexp.MustCompile(`<([\w.-]+@[\w.-]+\.\w+)>`)
		   			emailMatches := emailRegex.FindAllStringSubmatch(toField, -1)

		   			for _, emailMatch := range emailMatches {
		   				if len(emailMatch) > 1 {
		   					emails = append(emails, emailMatch[1])
		   				}
		   			}
		   		}

		   		fmt.Printf("emails: %v\n", emails)
		   		toAddress := getEmailValue(emails, emailMap)
		   		if toAddress == "" {
		   			toAddress = defaultToEmail
		   			toAddress = os.Getenv("MAILREDIR_DEFAULT_TO")
		   		}

		   		fmt.Printf("toAddress: %v\n", toAddress) */

		fmt.Printf("---MAIL PARSER---\n")

		email, err := parsemail.Parse(bytes.NewReader(rawEmail)) // returns Email struct and error
		if err != nil {
			return fmt.Errorf("failed to parse email: %w", err)
		}

		fmt.Printf("email.From: %v\n", email.From)
		fmt.Printf("email.Subject: %v\n", email.Subject)
		fmt.Printf("email.To: %v\n", email.To)

		toAddressSlice := []string{}
		for _, address := range email.To {
			fmt.Printf("address.Address: %v\n", address.Address)
			toAddress := getEmailValue(address.Address, emailMap)
			if toAddress != "" {
				fmt.Printf("Matched toAddress: %v\n", toAddress)
				toAddressSlice = append(toAddressSlice, toAddress)
			}
		}

		if len(toAddressSlice) == 0 {
			toAddress := os.Getenv("MAILREDIR_DEFAULT_TO")
			fmt.Printf("No matches, using environment variable MAILREDIR_DEFAULT_TO: %v\n", toAddress)
			if toAddress == "" {
				toAddress = defaultToEmail
				fmt.Printf("No environment variable, using default e-mail address: %v\n", toAddress)
			}
			toAddressSlice = []string{toAddress}
		}

		fmt.Printf("Final toAddressSlice: %v\n", toAddressSlice)
		fmt.Printf("---MAIL PARSER---\n")

		smtpServerHost := os.Getenv("MAILREDIR_SMTP_SERVER_HOST")
		smtpServerPort := os.Getenv("MAILREDIR_SMTP_SERVER_PORT")

		// Send the email via SMTP
		err = smtp.SendMail(smtpServerHost+":"+smtpServerPort, nil, email.From[0].Address, toAddressSlice, rawEmail)
		if err != nil {
			return fmt.Errorf("failed to send e-mail: %w", err)
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
