package service

import (
	"encoding/json"
	"github.com/go-stomp/stomp"
	"log"
	"marcel.works/stop-go/app/model"
	"os"
	"time"
)

var (
	topicCommand   = "/topic/go_stomp_command"
	topicBroadcast = "/topic/go_stomp_broadcast"
)

type StompService struct {
	Connection *stomp.Conn
	//DbService  *RethinkService
	DbService *RedisService
}

func (s *StompService) Connect() error {
	brokerHost := os.Getenv("ANNAPOKER_BROKER_HOST")
	if brokerHost == "" {
		brokerHost = "localhost:61613"
	}
	brokerUser := os.Getenv("ANNAPOKER_BROKER_USER")
	brokerPass := os.Getenv("ANNAPOKER_BROKER_PASS")
	options := []func(conn *stomp.Conn) error{
		stomp.ConnOpt.Login(brokerUser, brokerPass),
		stomp.ConnOpt.Host("/"),
	}
	connection, err := stomp.Dial("tcp", brokerHost, options...)
	if err != nil {
		return err
	}
	s.Connection = connection
	return nil
}

func (s *StompService) ReceiveCommands() {
	subscription, err := s.Connection.Subscribe(topicCommand, stomp.AckAuto)
	if err != nil {
		log.Fatal("cannot subscribe to", topicCommand, ":", err.Error())
	}
	log.Println("subscribed to", topicCommand)

	var command model.Command
	for true {
		message := <-subscription.C
		_ = json.Unmarshal(message.Body, &command)
		log.Println("received command:", command)

		switch command.Cmd {
		case "CREATE_SESSION":
			s.CreateSession(command.SessionId)
		case "SAVE_USER":
			s.SaveUser(command)
			s.PublishUsers(command.SessionId)
		case "REMOVE_USER":
			log.Println("Removing user..")
		case "GET_USERS":
			s.PublishUsers(command.SessionId)
		case "UPDATE_VOTING":
			s.UpdateVoting(command)
			s.PublishUsers(command.SessionId)
		case "RESET_VOTINGS":
			s.ResetVotings(command.SessionId)
			s.PublishUpdateUsers(command.SessionId)
		case "RESET_USERS":
			log.Println("should reset users")
		}
	}
}

func (s *StompService) CreateSession(sessionId string) {
	err := s.DbService.InsertSession(sessionId)
	if err != nil {
		log.Println("could not create new session:", err.Error())
	}
	log.Println("Created Session with Id", sessionId)
}

func (s *StompService) PublishUsers(sessionId string) {
	users, err := s.DbService.GetUsers(sessionId)
	if err != nil {
		log.Println("could not get users from session:", err.Error())
	}

	broadcast := model.Broadcast{
		Type:      "GET_USERS",
		Data:      users,
		Timestamp: time.Now(),
	}
	payload, _ := json.Marshal(broadcast)

	err = s.Connection.Send(topicBroadcast+"."+sessionId, "text/plain", payload)
	if err != nil {
		log.Println("could not send user broadcast:", err.Error())
	}
	log.Println("published users:", users)
}

func (s *StompService) PublishUpdateUsers(sessionId string) {
	users, err := s.DbService.GetUsers(sessionId)
	if err != nil {
		log.Println("could not get users from session:", err.Error())
	}
	broadcast := model.Broadcast{
		Type:      "UPDATE_USERS",
		Data:      users,
		Timestamp: time.Now(),
	}
	payload, _ := json.Marshal(broadcast)

	err = s.Connection.Send(topicBroadcast+"."+sessionId, "text/plain", payload)
	if err != nil {
		log.Println("could not send update user broadcast:", err.Error())
	}
	log.Println("published update users:", string(payload))
}

func (s *StompService) PublishRevealVotings(sessionId string) {
	broadcast := model.Broadcast{
		Type:      "REVEAL_VOTINGS",
		Data:      nil,
		Timestamp: time.Time{},
	}
	payload, _ := json.Marshal(broadcast)

	err := s.Connection.Send(topicBroadcast+"."+sessionId, "text/plain", payload)
	if err != nil {
		log.Println("could not send reveal votings broadcast:", err.Error())
	}
}

func (s *StompService) ResetVotings(sessionId string) {
	err := s.DbService.ResetVotings(sessionId)
	if err != nil {
		log.Println("could not reset votings:", err.Error())
	}
}

func (s *StompService) UpdateVoting(command model.Command) {
	err := s.DbService.UpdateUser(command.SessionId, command.User)
	if err != nil {
		log.Println("could not update user:", err.Error())
	}

	counter, err := s.DbService.CountVotings(command.SessionId)
	if err != nil {
		log.Println("could not count votings:", err.Error())
	}
	log.Println("counted", counter, "users with voting 0")

	if counter == 0 {
		s.PublishRevealVotings(command.SessionId)
	}
}

func (s *StompService) SaveUser(command model.Command) {
	err := s.DbService.AddUserToSession(command.SessionId, command.User)
	if err != nil {
		log.Println("could not append user", err.Error())
	}
	log.Println("added user:", command.User, "to session", command.SessionId)
}
