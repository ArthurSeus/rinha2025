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
	for i := range 10 {
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
		Subjects:  []string{"jobs"},
		Retention: nats.WorkQueuePolicy,
	})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		log.Fatal(err)
	}

	_, err = NatsJS.AddStream(&nats.StreamConfig{
		Name:      "WORKERS_STREAM",
		Subjects:  []string{"persist"},
		Retention: nats.WorkQueuePolicy,
	})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		log.Fatal(err)
	}

	//AddStreamss(NatsJS)
}

// func AddStreamss(js nats.JetStreamContext) {
// 	// Stream de workers (equivalente a payments_stream)
// 	_, err := NatsJS.AddStream(&nats.StreamConfig{
// 		Name:      "PAYMENTS_STREAM",
// 		Subjects:  []string{"payments"},
// 		Retention: nats.WorkQueuePolicy, // garante entrega única para consumidores
// 		//Storage:   nats.FileStorage,     // persiste em disco
// 		MaxMsgs: -1, // sem limite de mensagens
// 	})
// 	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
// 		log.Fatalf("Erro ao criar stream PAYMENTS_STREAM: %v", err)
// 	}

// 	// Stream de persistence (equivalente a payments_to_persist)
// 	_, err = NatsJS.AddStream(&nats.StreamConfig{
// 		Name:      "PAYMENTS_TO_PERSIST",
// 		Subjects:  []string{"payments_to_persist"},
// 		Retention: nats.WorkQueuePolicy, // garante entrega única para consumidores
// 		//Storage:   nats.FileStorage,     // persiste em disco
// 		MaxMsgs: -1, // sem limite de mensagens
// 	})
// 	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
// 		log.Fatalf("Erro ao criar stream PAYMENTS_TO_PERSIST: %v", err)
// 	}
// }
