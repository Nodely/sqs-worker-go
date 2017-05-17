package worker

import (
	"log"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// HandlerFunc is used to define the Handler that is run on for each message
type HandlerFunc func(msg *sqs.Message) error

// HandleMessage is used for the actual execution of each message
func (f HandlerFunc) HandleMessage(msg *sqs.Message) error {
	return f(msg)
}

// Handler interface
type Handler interface {
	HandleMessage(msg *sqs.Message) error
}

// Exported Variables
var (
	// what is the queue url we are connecting to, Defaults to empty
	QueueURL string
	// The maximum number of messages to return. Amazon SQS never returns more messages
	// than this value (however, fewer messages might be returned). Valid values
	// are 1 to 10. Default is 10.
	MaxNumberOfMessage int64 = 10
	// The duration (in seconds) for which the call waits for a message to arrive
	// in the queue before returning. If a message is available, the call returns
	// sooner than WaitTimeSeconds.
	WaitTimeSecond int64 = 20
)

// Start starts the polling and will continue polling till the application is forcibly stopped
func Start(svc *sqs.SQS, h Handler) {
	for {
		// log.Println("worker: Start Polling")
		params := &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(QueueURL), // Required
			MaxNumberOfMessages: aws.Int64(MaxNumberOfMessage),
			MessageAttributeNames: []*string{
				aws.String("All"), // Required
			},
			WaitTimeSeconds: aws.Int64(WaitTimeSecond),
		}

		resp, err := svc.ReceiveMessage(params)
		if err != nil {
			log.Println(err)
			continue
		}
		if len(resp.Messages) > 0 {
			run(svc, h, resp.Messages)
		}
	}
}

// poll launches goroutine per received message and wait for all message to be processed
func run(svc *sqs.SQS, h Handler, messages []*sqs.Message) {
	numMessages := len(messages)

	var wg sync.WaitGroup
	wg.Add(numMessages)
	for i := range messages {
		go func(m *sqs.Message) {
			// launch goroutine
			defer wg.Done()
			if err := handleMessage(svc, m, h); err != nil {
				log.Println(err.Error())
			}
		}(messages[i])
	}

	wg.Wait()
}

func handleMessage(svc *sqs.SQS, m *sqs.Message, h Handler) error {
	err := h.HandleMessage(m)
	if err != nil {
		return err
	}

	params := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(QueueURL), // Required
		ReceiptHandle: m.ReceiptHandle,      // Required
	}
	_, err = svc.DeleteMessage(params)
	if err != nil {
		return err
	}

	return nil
}
