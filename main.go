package main

import (
	"context"
	"fmt"
	"log"
	"time"

	cloud "macurandb/GoSMS-AWS/aws"

	"github.com/spf13/viper"
)

// LoadConfig loads AWS credentials from a configuration file
func LoadConfig() (cloud.AWSAuthConfig, error) {
	viper.SetConfigName("config") // Config file name (without extension)
	viper.SetConfigType("yaml")   // Config file type
	viper.AddConfigPath(".")      // Look in the current directory

	// Read the configuration file
	if err := viper.ReadInConfig(); err != nil {
		return cloud.AWSAuthConfig{}, fmt.Errorf("error reading config file: %w", err)
	}

	// Parse the credentials
	authConfig := cloud.AWSAuthConfig{
		AccessKeyID:     viper.GetString("aws.access_key_id"),
		SecretAccessKey: viper.GetString("aws.secret_access_key"),
		Region:          viper.GetString("aws.region"),
	}

	return authConfig, nil
}

func main() {
	// Load AWS credentials
	authConfig, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// SMS service configuration
	serviceConfig := cloud.SMSServiceConfig{
		// TopicARN:   "arn:aws:sns:eu-west-2:698642784647:OTPService",
		SenderID:   "OTPService",
		SMSType:    "Transactional",
		MaxRetries: 3,
		RetryDelay: 2 * time.Second,
	}

	// Create a context
	ctx := context.Background()

	// Create an instance of the SMS service
	smsService, err := cloud.NewSMSService(ctx, authConfig, serviceConfig)
	if err != nil {
		log.Fatalf("Error initializing SMS service: %v", err)
	}

	// Define a test phone number
	phoneNumber := "+447770428172"

	// Send verification SMS
	code, err := smsService.SendVerificationSMS(ctx, phoneNumber)
	if err != nil {
		log.Fatalf("Error sending SMS: %v", err)
	}

	fmt.Printf("Verification code sent successfully: %s\n", code)
}
