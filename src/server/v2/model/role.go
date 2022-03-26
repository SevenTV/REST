package model

import "github.com/SevenTV/Common/structures/v3"

type Role struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Position int32  `json:"position"`
	Color    int32  `json:"color"`
	Allowed  int64  `json:"allowed"`
	Denied   int64  `json:"denied"`
}

func NewRole(s *structures.Role) *Role {
	return &Role{
		ID:       s.ID.Hex(),
		Name:     s.Name,
		Position: s.Position,
		Color:    s.Color,
		Allowed:  int64(s.Allowed),
		Denied:   int64(s.Denied),
	}
}
