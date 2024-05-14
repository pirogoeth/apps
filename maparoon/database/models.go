// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0

package database

import (
	"database/sql"
)

type Host struct {
	ID         int64          `json:"id"`
	NetworkID  int64          `json:"network_id"`
	Address    string         `json:"address"`
	Comments   sql.NullString `json:"comments"`
	Attributes sql.NullString `json:"attributes"`
}

type HostPort struct {
	Address    string         `json:"address"`
	Port       int64          `json:"port"`
	Protocol   string         `json:"protocol"`
	Comments   sql.NullString `json:"comments"`
	Attributes sql.NullString `json:"attributes"`
}

type Network struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Address    string `json:"address"`
	Cidr       int64  `json:"cidr"`
	Comments   string `json:"comments"`
	Attributes string `json:"attributes"`
}