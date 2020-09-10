package service

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/segmentio/encoding/json"
	"go.uber.org/zap"
	"marcel.works/stop-go/app/model"
	"os"
	"time"
)

type RedisService struct {
	Logger *zap.Logger
	Client *redis.Client
	Ctx    context.Context
}

const TTL = 4 * time.Hour

func (s *RedisService) Connect() error {
	auth := os.Getenv("ANNAPOKER_DB_AUTH")
	host := os.Getenv("ANNAPOKER_DB_HOST")
	if host == "" {
		host = "localhost:6379"
	}
	s.Client = redis.NewClient(&redis.Options{
		Addr:     host,
		Password: auth,
		DB:       0,
	})
	s.Ctx = context.Background()
	return s.Client.Ping(s.Ctx).Err()
}

func (s *RedisService) InsertSession(sessionId string) error {
	session := model.Session{
		Id:        sessionId,
		Users:     nil,
		Timestamp: time.Now(),
	}
	payload, _ := json.Marshal(session)
	return s.Client.Set(s.Ctx, sessionId, payload, TTL).Err()
}

func (s *RedisService) AddUserToSession(sessionId string, user model.User) error {
	session, err := s.getSession(sessionId)
	if err != nil {
		return err
	}
	session.Users = append(session.Users, user)
	outbound, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return s.Client.Set(s.Ctx, sessionId, outbound, TTL).Err()
}

func (s *RedisService) RemoveUserFromSession(sessionId string, user model.User) error {
	session, err := s.getSession(sessionId)
	if err != nil {
		return err
	}
	for index, u := range session.Users {
		if user.Uuid == u.Uuid {
			session.Users[index] = session.Users[len(session.Users)-1]
			session.Users = session.Users[:len(session.Users)-1]
		}
	}
	payload, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return s.Client.Set(s.Ctx, sessionId, payload, TTL).Err()
}

func (s *RedisService) GetUsers(sessionId string) ([]model.User, error) {
	session, err := s.getSession(sessionId)
	if err != nil {
		return nil, err
	}
	return session.Users, nil
}

func (s *RedisService) UpdateUser(sessionId string, user model.User) error {
	session, err := s.getSession(sessionId)
	if err != nil {
		return err
	}
	for index, u := range session.Users {
		if u.Uuid == user.Uuid {
			session.Users[index].Voting = user.Voting
		}
	}
	payload, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return s.Client.Set(s.Ctx, sessionId, payload, TTL).Err()
}

func (s *RedisService) ResetVotings(sessionId string) error {
	session, err := s.getSession(sessionId)
	if err != nil {
		return err
	}
	for index, _ := range session.Users {
		session.Users[index].Voting = 0
	}
	payload, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return s.Client.Set(s.Ctx, sessionId, payload, TTL).Err()
}

func (s *RedisService) CountVotings(sessionId string) (int, error) {
	counter := 0
	session, err := s.getSession(sessionId)
	if err != nil {
		return -1, err
	}
	for index, _ := range session.Users {
		if session.Users[index].Voting == 0 {
			counter++
		}
	}
	return counter, nil
}

func (s *RedisService) getSession(sessionId string) (*model.Session, error) {
	payload, err := s.Client.Get(s.Ctx, sessionId).Result()
	if err != nil {
		return nil, err
	}
	var session model.Session
	err = json.Unmarshal([]byte(payload), &session)
	if err != nil {
		return nil, err
	}
	return &session, nil
}
