package authstore

import (
	"time"
)

type Authorization struct {
	ID string `db:"id,omitempty" json:"-"`

	MachinePPID string `db:"machine_ppid,omitempty" json:"machine_ppid,omitempty"`
	PublicKey   string `db:"public_key,omitempty" json:"public_key,omitempty"`
	SharedKey   string `db:"shared_key,omitempty" json:"shared_key"`
	Slug        string `db:"slug,omitempty" json:"slug"`

	CreatedAt time.Time `db:"created_at,omitempty" json:"created_at,omitempty"`
	UpdatedAt time.Time `db:"updated_at,omitempty" json:"updated_at,omitempty"`
	DeletedAt time.Time `db:"deleted_at,omitempty" json:"-"`
}

type Store interface {
	SetMaster(secret string) error
	Add(auth *Authorization) error
	Set(auth *Authorization) error
	Get(id string) (*Authorization, error)
	GetBySlug(id string) (*Authorization, error)
	GetByPub(id string) (*Authorization, error)
	Delete(auth *Authorization) error
	Close() error
}
