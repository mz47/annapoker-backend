package service

import (
	"gopkg.in/rethinkdb/rethinkdb-go.v6"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
	"log"
	"marcel.works/stop-go/app/model"
	"os"
	"strings"
)

var (
	db            = "annapoker"
	tableSessions = "sessions"
	fieldUsers    = "users"
	fieldUuid     = "id"
)

type RethinkService struct {
	Session *rethinkdb.Session
}

func (s *RethinkService) Connect() error {
	dbHostEnv := os.Getenv("ANNAPOKER_DB_HOSTS")
	if dbHostEnv == "" {
		dbHostEnv = "localhost:28015"
	}
	hosts := strings.Split(dbHostEnv, ",")
	session, err := r.Connect(r.ConnectOpts{
		Addresses: hosts,
	})
	if err != nil {
		log.Fatalln("could not connect to database:", err.Error())
		return err
	}
	s.Session = session
	return nil
}

func (s *RethinkService) AddUserToSession(sessionId string, user model.User) error {
	_, err := r.DB(db).Table(tableSessions).
		Filter(r.Row.Field(fieldUuid).Eq(sessionId)).
		Update(
			map[string]r.Term{fieldUsers: r.Row.Field(fieldUsers).Append(user)},
		).
		RunWrite(s.Session)
	if err != nil {
		return err
	}
	return nil
}

func (s *RethinkService) GetUsers(sessionId string) ([]model.User, error) {
	var arr [][]model.User
	result, err := r.DB(db).Table(tableSessions).
		Filter(r.Row.Field(fieldUuid).Eq(sessionId)).
		Field(fieldUsers).
		Run(s.Session)
	if err != nil {
		return nil, err
	}
	defer result.Close()

	err = result.All(&arr)
	if err != nil {
		return nil, err
	}
	return arr[0], nil
}

func (s *RethinkService) UpdateUser(sessionId string, user model.User) error {
	_, err := r.DB(db).Table(tableSessions).
		Get(sessionId).
		Update(
			map[string]r.Term{"users": r.Row.Field("users").Map(func(u r.Term) r.Term {
				return r.Branch(
					u.Field("Uuid").Eq(user.Uuid),
					u.Merge(map[string]int{"Voting": user.Voting}),
					u,
				)
			})},
		).
		RunWrite(s.Session)
	if err != nil {
		return err
	}
	return nil
}

func (s *RethinkService) ResetVotings(sessionId string) error {
	_, err := r.DB(db).Table(tableSessions).Get(sessionId).
		Update(
			map[string]r.Term{"users": r.Row.Field("users").Map(func(u r.Term) r.Term {
				return r.Branch(
					u.Field("Voting"),
					u.Merge(map[string]int{"Voting": 0}),
					u,
				)
			})},
		).RunWrite(s.Session)
	if err != nil {
		return err
	}
	return nil
}

func (s *RethinkService) CountVotings(sessionId string) (int, error) {
	var arr []interface{}
	result, err := r.DB(db).Table(tableSessions).
		Get(sessionId).
		Field("users").
		Field("Voting").
		Count(func(v r.Term) r.Term {
			return v.Eq(0)
		}).
		Run(s.Session)
	if err != nil {
		return -1, err
	}
	defer result.Close()

	err = result.All(&arr)
	if len(arr) > 0 {
		log.Println("#### count votings result:", arr[0])
		raw := arr[0].(float64)
		return int(raw), nil
	}
	return -1, nil
}

func (s *RethinkService) InsertSession(sessionId string) error {
	session := model.Session{
		Id:    sessionId,
		Users: nil,
	}
	_, err := r.DB(db).Table(tableSessions).Insert(session).Run(s.Session)
	if err != nil {
		return err
	}
	return nil
}

func (s *RethinkService) InsertUser(user model.User) error {
	_, err := r.DB(db).Table("users").Insert(user).Run(s.Session)
	if err != nil {
		return err
	}
	return nil
}

func (s *RethinkService) GetAllSessions() error {
	result := r.DB(db).Table(tableSessions).String()
	log.Println("received", result)
	return nil
}
