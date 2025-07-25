package config

import (
	"log"
	"os"
	"time"

	"github.com/nats-io/nats.go"
)

func JetStreamInit() nats.JetStreamContext {
	const maxRetries = 10
	const delay = 3 * time.Second

	url := os.Getenv("NATS_URL")
	if url == "" {
		url = nats.DefaultURL // fallback opcional
	}

	var nc *nats.Conn
	var js nats.JetStreamContext
	var err error

	for i := 1; i <= maxRetries; i++ {
		nc, err = nats.Connect(url)
		if err == nil {
			log.Printf("connected to NATS on attempt %d", i)
			break
		}
		log.Printf("attempt %d: failed to connect to NATS: %v", i, err)
		time.Sleep(delay)
	}
	if err != nil {
		log.Fatalf("could not connect to NATS after %d attempts: %v", maxRetries, err)
	}

	js, err = nc.JetStream(nats.PublishAsyncMaxPending(256))
	if err != nil {
		log.Fatal("failed to create JetStream context:", err)
	}
	log.Printf("created JetStream on NATS")

	err = CreateStream(js)
	if err != nil {
		log.Fatal("failed to create stream:", err)
	}
	log.Printf("created stream on NATS")

	return js
}

const (
	StreamName     = "PAYMENTS"
	StreamSubjects = "PAYMENTS.*"
)

func CreateStream(jetStream nats.JetStreamContext) error {
	stream, err := jetStream.StreamInfo(StreamName)

	if err != nil && err != nats.ErrStreamNotFound {
		return err
	}

	// stream not found, create it
	if stream == nil {
		log.Printf("Creating stream: %s\n", StreamName)

		_, err = jetStream.AddStream(&nats.StreamConfig{
			Name:     StreamName,
			Subjects: []string{StreamSubjects},
		})
		if err != nil {
			return err
		}
	}
	return nil
}
