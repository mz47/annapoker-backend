package service

import (
	"encoding/json"
	"github.com/go-stomp/stomp"
	"log"
	"marcel.works/stop-go/app/model"
	"os"
	"time"
)

const (
	_topicCommand   = "/topic/go_stomp_command"
	_topicBroadcast = "/topic/go_stomp_broadcast"
)

type StompService struct {
	Connection *stomp.Conn
	DbService  *RedisService
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
	subscription, err := s.Connection.Subscribe(_topicCommand, stomp.AckAuto)
	if err != nil {
		log.Fatal("cannot subscribe to", _topicCommand, ":", err.Error())
	}
	log.Println("subscribed to", _topicCommand)

	var command model.Command
	for true {
		message := <-subscription.C
		_ = json.Unmarshal(message.Body, &command)
		log.Println(">>> received", command.Cmd, "on", _topicCommand)
		switch command.Cmd {
		case "CREATE_SESSION":
			s.CreateSession(command.SessionId)
		case "SAVE_USER":
			s.SaveUser(command)
			s.PublishUsers(command.SessionId)
		case "GET_USERS":
			s.PublishUsers(command.SessionId)
		case "UPDATE_VOTING":
			s.UpdateVoting(command)
			s.PublishUsers(command.SessionId)
		case "RESET_VOTINGS":
			s.ResetVotings(command.SessionId)
			s.PublishUpdateUsers(command.SessionId)
		case "REMOVE_USER":
			s.RemoveUser(command)
			s.PublishUpdateUsers(command.SessionId)
		}
	}
}

func (s *StompService) CreateSession(sessionId string) {
	err := s.DbService.InsertSession(sessionId)
	if err != nil {
		log.Println("could not create new session:", err.Error())
	}
}

func (s *StompService) PublishUsers(sessionId string) {
	users, err := s.DbService.GetUsers(sessionId)
	if err != nil {
		log.Println("no users found or session nil:", err.Error())
		s.SendBroadcast("NO_SESSION_FOUND", sessionId, nil)
		return
	}
	s.SendBroadcast("GET_USERS", sessionId, users)
}

func (s *StompService) PublishUpdateUsers(sessionId string) {
	users, err := s.DbService.GetUsers(sessionId)
	if err != nil {
		log.Println("no users found or session nil:", err.Error())
		s.SendBroadcast("NO_SESSION_FOUND", sessionId, nil)
		return
	}
	s.SendBroadcast("UPDATE_USERS", sessionId, users)
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
	if counter == 0 {
		s.SendBroadcast("REVEAL_VOTINGS", command.SessionId, nil)
	}
}

func (s *StompService) SaveUser(command model.Command) {
	err := s.DbService.AddUserToSession(command.SessionId, command.User)
	if err != nil {
		log.Println("could not append user", err.Error())
	}
}

func (s *StompService) RemoveUser(command model.Command) {
	err := s.DbService.RemoveUserFromSession(command.SessionId, command.User)
	if err != nil {
		log.Println("could not remove user", err.Error())
	}
}

func (s *StompService) SendBroadcast(typ string, sessionId string, data interface{}) {
	broadcast := model.Broadcast{
		Type:      typ,
		Data:      data,
		Timestamp: time.Now(),
	}
	payload, _ := json.Marshal(broadcast)
	err := s.Connection.Send(_topicBroadcast+"."+sessionId, "text/plan", payload)
	if err != nil {
		log.Println("could not send broadcast", typ, ":", err.Error())
	}
	log.Println("<<< sent", broadcast.Type, "on", _topicBroadcast)
}
