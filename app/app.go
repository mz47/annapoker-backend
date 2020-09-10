package app

import (
	"go.uber.org/zap"
	"log"
	"marcel.works/stop-go/app/service"
)

var stop = make(chan bool)

type App struct{}

func (a *App) Start() {
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.OutputPaths = []string{
		"./logs/annapoker.log",
	}
	logger, err := loggerConfig.Build()
	if err != nil {
		log.Fatalln("Error creating logger:", err.Error())
	}
	defer logger.Sync()

	dbService := service.RedisService{Logger: logger}
	stompService := service.StompService{Logger: logger, DbService: &dbService}

	err = dbService.Connect()
	if err != nil {
		logger.Error("could not connect to redis", zap.Error(err))
	}
	logger.Info("connected to redis")

	err = stompService.Connect()
	if err != nil {
		logger.Error("could not connect to rabbitmq", zap.Error(err))
	}
	logger.Info("connected to rabbitmq")
	log.Println("Started Application...")
	go stompService.ReceiveCommands()
	<-stop
}
