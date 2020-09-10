package service

import (
	"encoding/json"
	"github.com/go-stomp/stomp"
	"go.uber.org/zap"
	"marcel.works/stop-go/app/model"
	"os"
	"time"
)

const (
	_topicCommand   = "/topic/go_stomp_command"
	_topicBroadcast = "/topic/go_stomp_broadcast"
)

type StompService struct {
	Logger     *zap.Logger
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
		s.Logger.Error("cannot subscribe to" + _topicCommand + ":" + err.Error())
	}
	s.Logger.Info("successfully subscriped to topic", zap.String("topic", _topicCommand))

	var command model.Command
	for true {
		message := <-subscription.C
		_ = json.Unmarshal(message.Body, &command)
		s.Logger.Info(">>> received command", zap.String("command", command.Cmd), zap.String("topic", _topicCommand))
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
		s.Logger.Error("could not create new session", zap.Error(err))
	}
}

func (s *StompService) PublishUsers(sessionId string) {
	users, err := s.DbService.GetUsers(sessionId)
	if err != nil {
		s.Logger.Error("no users found or session null", zap.Error(err))
		s.SendBroadcast("NO_SESSION_FOUND", sessionId, nil)
		return
	}
	s.SendBroadcast("GET_USERS", sessionId, users)
}

func (s *StompService) PublishUpdateUsers(sessionId string) {
	users, err := s.DbService.GetUsers(sessionId)
	if err != nil {
		s.Logger.Error("could not create new session", zap.Error(err))
		s.SendBroadcast("NO_SESSION_FOUND", sessionId, nil)
		return
	}
	s.SendBroadcast("UPDATE_USERS", sessionId, users)
}

func (s *StompService) ResetVotings(sessionId string) {
	err := s.DbService.ResetVotings(sessionId)
	if err != nil {
		s.Logger.Error("could not reset votings", zap.Error(err))
	}
}

func (s *StompService) UpdateVoting(command model.Command) {
	err := s.DbService.UpdateUser(command.SessionId, command.User)
	if err != nil {
		s.Logger.Error("could not update users", zap.Error(err))
	}
	counter, err := s.DbService.CountVotings(command.SessionId)
	if err != nil {
		s.Logger.Error("could not count votings", zap.Error(err))
	}
	if counter == 0 {
		s.SendBroadcast("REVEAL_VOTINGS", command.SessionId, nil)
	}
}

func (s *StompService) SaveUser(command model.Command) {
	err := s.DbService.AddUserToSession(command.SessionId, command.User)
	if err != nil {
		s.Logger.Error("could not append user", zap.Error(err))
	}
}

func (s *StompService) RemoveUser(command model.Command) {
	err := s.DbService.RemoveUserFromSession(command.SessionId, command.User)
	if err != nil {
		s.Logger.Error("could not remote user", zap.Error(err))
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
		s.Logger.Error("could not send broadcast", zap.String("type", typ), zap.Error(err))
	}
	s.Logger.Info("<<< sent broadcast", zap.String("type", broadcast.Type), zap.String("topic", _topicBroadcast))
}
