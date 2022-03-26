package model

import (
	"github.com/SevenTV/Common/structures/v3"
	"github.com/SevenTV/Common/utils"
)

type User struct {
	ID          string `json:"id"`
	TwitchID    string `json:"twitch_id"`
	Login       string `json:"login"`
	DisplayName string `json:"display_name"`
	Role        *Role  `json:"role"`
}

func NewUser(s *structures.User) *User {
	u := &User{
		ID:          s.ID.Hex(),
		Login:       s.Username,
		DisplayName: utils.Ternary(s.DisplayName != "", s.DisplayName, s.Username).(string),
		Role:        NewRole(s.GetHighestRole()),
	}
	tw, _ := s.Connections.Twitch()
	if tw != nil {
		u.TwitchID = tw.ID
	}

	return u
}
