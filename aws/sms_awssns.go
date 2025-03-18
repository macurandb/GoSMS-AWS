package cloud

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	// "gitlab.com/mikado-platform/services/players/internal/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

//go:generate mockgen -destination=mocks/mock_smsservice.go -package=mocks . SMSService

// AWSAuthConfig contains AWS authentication credentials
type AWSAuthConfig struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
}

// SMSServiceConfig holds SMS service configuration
type SMSServiceConfig struct {
	SenderID   string
	SMSType    string // Transactional/Promotional
	TopicARN   string
	MaxRetries int
	RetryDelay time.Duration
}

// SMSService defines the methods for sending SMS.
type SMSService interface {
	SendVerificationSMS(ctx context.Context, phoneNumber string) (string, error)
}

// SMSServiceAWS handles SMS operations using AWS SNS
type SMSServiceAWS struct {
	client *sns.Client
	config SMSServiceConfig
}

// Ensure SMSService implements SMSServiceInterface
var _ SMSService = (*SMSServiceAWS)(nil)

// SMSError represents an SMS operation error
type SMSError struct {
	PhoneNumber string
	Code        string
	Err         error
}

func (e *SMSError) Error() string {
	return fmt.Sprintf("SMS error [%s] for %s: %v", e.Code, e.PhoneNumber, e.Err)
}

// NewSMSService creates a new SMS service instance
func NewSMSService(ctx context.Context, authConfig AWSAuthConfig, serviceConfig SMSServiceConfig) (SMSService, error) {
	// Load AWS configuration with credentials
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(authConfig.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			authConfig.AccessKeyID,
			authConfig.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Validate credentials
	_, err = cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("invalid AWS credentials: %w", err)
	}

	return &SMSServiceAWS{
		client: sns.NewFromConfig(cfg),
		config: serviceConfig,
	}, nil
}

// SendVerificationSMS sends a verification code SMS
func (s *SMSServiceAWS) SendVerificationSMS(ctx context.Context, phoneNumber string) (string, error) {
	code := "12345" // utils.GenerateVerificationCode(14)
	message := fmt.Sprintf("Your verification code: %s", code)

	_, err := s.sendSMSWithRetry(ctx, phoneNumber, message, 0)
	if err != nil {
		return "", err
	}

	return code, nil
}

// sendSMSWithRetry handles SMS sending with retry logic
func (s *SMSServiceAWS) sendSMSWithRetry(ctx context.Context, phoneNumber, message string, attempt int) (string, error) {
	input := &sns.PublishInput{
		// TopicArn: aws.String(s.config.TopicARN),
		PhoneNumber: aws.String(phoneNumber),
		Message:     aws.String(message),
		MessageAttributes: map[string]types.MessageAttributeValue{
			"AWS.SNS.SMS.SMSType": {
				DataType:    aws.String("String"),
				StringValue: aws.String(s.config.SMSType),
			},
		},
	}

	if s.config.SenderID != "" {
		input.MessageAttributes["AWS.SNS.SMS.SenderID"] = types.MessageAttributeValue{
			DataType:    aws.String("String"),
			StringValue: aws.String(s.config.SenderID),
		}
	}

	result, err := s.client.Publish(ctx, input)
	if err == nil {
		log.Printf("SMS sent to %s (MessageID: %s)", phoneNumber, *result.MessageId)
		return *result.MessageId, nil
	}

	if attempt < s.config.MaxRetries {
		log.Printf("Retrying SMS to %s (attempt %d)", phoneNumber, attempt+1)
		time.Sleep(s.config.RetryDelay)
		return s.sendSMSWithRetry(ctx, phoneNumber, message, attempt+1)
	}

	return "", &SMSError{
		PhoneNumber: phoneNumber,
		Code:        "SNS_MAX_RETRIES_EXCEEDED",
		Err:         err,
	}
}
