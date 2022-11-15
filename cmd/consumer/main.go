package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/matheusF23/go-rabbitmq-grafana/internal/order/infra/database"
	"github.com/matheusF23/go-rabbitmq-grafana/internal/order/usecase"
	"github.com/matheusF23/go-rabbitmq-grafana/pkg/rabbitmq"
	_ "github.com/mattn/go-sqlite3"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	db, err := sql.Open("sqlite3", "./orders.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	repository := database.NewOrderRepository(db)
	uc := usecase.CalculateFinalPriceUseCase{OrderRepository: repository}
	ch, err := rabbitmq.OpenChannel()
	if err != nil {
		panic(err)
	}
	defer ch.Close()
	out := make(chan amqp.Delivery)
	forever := make(chan bool)
	go rabbitmq.Consume(ch, out) // Thread 2
	qtdWorks := 5
	for i := 1; i <= qtdWorks; i++ {
		go worker(out, &uc, i)
	}
	<-forever
}

func worker(deliveryMessage <-chan amqp.Delivery, uc *usecase.CalculateFinalPriceUseCase, workerID int) {
	for msg := range deliveryMessage {
		var inputDTO usecase.OrderInputDTO
		err := json.Unmarshal(msg.Body, &inputDTO)
		if err != nil {
			panic(err)
		}
		outputDTO, err := uc.Execute(inputDTO)
		if err != nil {
			panic(err)
		}
		msg.Ack(false)
		fmt.Printf("Worker %d has processed order %s\n", workerID, outputDTO.ID)
		time.Sleep(time.Second)
	}
}
