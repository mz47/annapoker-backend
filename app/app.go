package app

import (
	"log"
	"marcel.works/stop-go/app/service"
)

var (
	stop = make(chan bool)
)

type App struct {
	StompService *service.StompService
	DbService    *service.RedisService
}

func (a *App) Start() {
	err := a.DbService.Connect()
	if err != nil {
		log.Fatalln("could not connect to database:", err.Error())
	}
	log.Println("connected to database")

	err = a.StompService.Connect()
	if err != nil {
		log.Fatalln("could not connect to broker:", err.Error())
	}
	log.Println("connected to broker")
	go a.StompService.ReceiveCommands()
	<-stop
}
