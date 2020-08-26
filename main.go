package main

import (
	"marcel.works/stop-go/app"
	"marcel.works/stop-go/app/service"
)

func main() {
	//dbService := service.RethinkService{}
	dbService := service.RedisService{}
	stompService := service.StompService{DbService: &dbService}
	a := app.App{
		StompService: &stompService,
		DbService:    &dbService,
	}
	a.Start()
}
