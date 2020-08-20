package model

import "time"

type Command struct {
	Cmd       string `json:"cmd"`
	Username  string `json:"user"`
	User      User   `json:"data"`
	SessionId string `json:"sessionId"`
}

type Broadcast struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

type User struct {
	Uuid     string `json:"uuid"`
	Username string `json:"username"`
	Voting   int    `json:"voting"`
}

type Session struct {
	Id    string `rethinkdb:"id,omitempty"`
	Users []User `rethinkdb:"users"`
}
