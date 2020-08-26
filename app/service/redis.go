package service

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/segmentio/encoding/json"
	"marcel.works/stop-go/app/model"
	"os"
)

type RedisService struct {
	Client *redis.Client
	Ctx    context.Context
}

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
		Id:    sessionId,
		Users: nil,
	}
	payload, _ := json.Marshal(session)
	return s.Client.Set(s.Ctx, sessionId, payload, 0).Err()
}

func (s *RedisService) InsertUser(user model.User) error {
	payload, _ := json.Marshal(user)
	return s.Client.Set(s.Ctx, user.Uuid, payload, 0).Err()
}

func (s *RedisService) AddUserToSession(sessionId string, user model.User) error {
	inbound, err := s.Client.Get(s.Ctx, sessionId).Result()
	if err != nil {
		return err
	}
	var session model.Session
	_ = json.Unmarshal([]byte(inbound), &session)
	session.Users = append(session.Users, user)
	outbound, _ := json.Marshal(session)
	return s.Client.Set(s.Ctx, sessionId, outbound, 0).Err()
}

func (s *RedisService) GetUsers(sessionId string) ([]model.User, error) {
	payload, err := s.Client.Get(s.Ctx, sessionId).Result()
	if err != nil {
		return nil, err
	}
	var session model.Session
	_ = json.Unmarshal([]byte(payload), &session)
	return session.Users, nil
}

func (s *RedisService) UpdateUser(sessionId string, user model.User) error {
	payload, err := s.Client.Get(s.Ctx, sessionId).Result()
	if err != nil {
		return err
	}
	var session model.Session
	err = json.Unmarshal([]byte(payload), &session)
	if err != nil {
		return err
	}
	for index, u := range session.Users {
		if user.Uuid == u.Uuid {
			session.Users[index] = user
		}
	}
	return nil
}

func (s *RedisService) ResetVotings(sessionId string) error {
	payload, err := s.Client.Get(s.Ctx, sessionId).Result()
	if err != nil {
		return err
	}
	var session model.Session
	err = json.Unmarshal([]byte(payload), &session)
	if err != nil {
		return err
	}
	for index, _ := range session.Users {
		session.Users[index].Voting = 0
	}
	return nil
}

func (s *RedisService) CountVotings(sessionId string) (int, error) {
	counter := 0
	payload, err := s.Client.Get(s.Ctx, sessionId).Result()
	if err != nil {
		return -1, err
	}
	var session model.Session
	err = json.Unmarshal([]byte(payload), &session)
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
