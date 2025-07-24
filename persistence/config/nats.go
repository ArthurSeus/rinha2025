package config

import (
	"log"
	"os"
	"time"

	"github.com/nats-io/nats.go"
)

var natsConn *nats.Conn
var NatsJS nats.JetStreamContext

func InitNATS() {
	url := os.Getenv("NATS_URL")
	var err error
	for i := 0; i < 10; i++ {
		natsConn, err = nats.Connect(url)
		if err == nil {
			break
		}
		log.Printf("Tentativa %d: Erro ao conectar no NATS: %v", i+1, err)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		log.Fatalf("Erro ao conectar no NATS: %v", err)
	}
	NatsJS, err = natsConn.JetStream()
	if err != nil {
		log.Fatalf("Erro ao criar contexto JetStream: %v", err)
	}

	_, err = NatsJS.AddStream(&nats.StreamConfig{
		Name:      "WORKERS_STREAM",
		Subjects:  []string{"persist"},
		Retention: nats.WorkQueuePolicy,
	})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		log.Fatal(err)
	}
}
