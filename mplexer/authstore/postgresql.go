package authstore

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Authorization struct {
	ID          string    `db:"id,omitempty"`
	Slug        string    `db:"slug,omitempty"`
	MachinePPID string    `db:"machine_ppid,omitempty"`
	PublicKey   string    `db:"public_key,omitempty"`
	SharedKey   string    `db:"shared_key,omitempty"`
	CreatedAt   time.Time `db:"created_at,omitempty"`
	UpdatedAt   time.Time `db:"updated_at,omitempty"`
	DeletedAt   time.Time `db:"deleted_at,omitempty"`
}

type Store interface {
	Add(auth *Authorization) error
	Set(auth *Authorization) error
	Get(id string) (*Authorization, error)
	GetBySlug(id string) (*Authorization, error)
	GetByPub(id string) (*Authorization, error)
	Delete(auth *Authorization) error
	Close() error
}

type StoreConfig interface {
	Type() string
	URL() string
}

func NewStore(c StoreConfig) (Store, error) {
	// https://godoc.org/github.com/lib/pq

	connStr := "postgres://postgres:postgres@localhost/postgres"
	if strings.Contains(connStr, "@localhost/") {
		connStr += "?sslmode=disable"
	} else {
		connStr += "?sslmode=required"
	}
	// TODO url.Parse
	dbtype := "postgres"
	sqlBytes, err := ioutil.ReadFile("./init.sql")
	if nil != err {
		return nil, err
	}

	ctx, done := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer done()
	db, err := sql.Open(dbtype, connStr)
	if err := db.PingContext(ctx); nil != err {
		return nil, err
	}
	if _, err := db.ExecContext(ctx, string(sqlBytes)); nil != err {
		return nil, err
	}

	dbx := sqlx.NewDb(db, dbtype)

	return &PGStore{
		dbx: dbx,
	}, nil
}

type PGStore struct {
	dbx *sqlx.DB
}

func (s *PGStore) Add(auth *Authorization) error {
	ctx, done := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer done()

	tx, err := s.dbx.DB.BeginTx(ctx, &sql.TxOptions{})
	if nil != err {
		return err
	}
	query1 := `LOCK TABLE authorizations IN SHARE ROW EXCLUSIVE MODE`
	_, err = tx.ExecContext(ctx, query1)
	if nil != err {
		return err
	}
	query2 := `
		INSERT INTO authorizations (slug, shared_key, public_key)
			SELECT $1, $2, $3
		WHERE NOT EXISTS (
			SELECT slug FROM authorizations WHERE deleted_at = '1970-01-01 00:00:00' AND slug = $1
		)
	`
	res, err := tx.ExecContext(ctx, query2, auth.Slug, auth.SharedKey, auth.PublicKey)
	if nil != err {
		return err
	}

	// PostgreSQL does support RowsAffected(), but not LastInsertId()
	if count, _ := res.RowsAffected(); count != 1 {
		return fmt.Errorf("record not added (probably exists)")
	}

	if err := tx.Commit(); nil != err {
		return err
	}

	return nil
}

func (s *PGStore) Set(auth *Authorization) error {
	ctx, done := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer done()
	query := `
		UPDATE authorizations SET
			machine_ppid=$1,
			shared_key=$2,
			public_key=$3
		WHERE
			deleted_at = '1970-01-01 00:00:00'
				AND shared_key = $2
				AND machine_ppid= ''
	`
	row, err := s.dbx.ExecContext(ctx, query, auth.MachinePPID, auth.SharedKey, auth.PublicKey)
	if nil != err {
		return err
	}
	// PostgreSQL does support RowsAffected()
	if count, _ := row.RowsAffected(); count != 1 {
		return fmt.Errorf("record exists")
	}
	return nil
}

func (s *PGStore) Get(id string) (*Authorization, error) {
	ctx, done := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer done()
	query := `SELECT * FROM authorizations WHERE deleted_at = '1970-01-01 00:00:00' AND (slug = $1 OR public_key = $1)`
	row := s.dbx.QueryRowxContext(ctx, query, id)
	if nil != row {
		auth := &Authorization{}
		if err := row.StructScan(auth); nil != err {
			fmt.Println("what's wrong here", err)
			return nil, err
		}
		return auth, nil
	}
	return nil, nil
}

func (s *PGStore) GetBySlug(id string) (*Authorization, error) {
	ctx, done := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer done()
	query := `SELECT * FROM authorizations WHERE deleted_at = '1970-01-01 00:00:00' AND slug = $1`
	row := s.dbx.QueryRowxContext(ctx, query, id)
	if nil != row {
		auth := &Authorization{}
		if err := row.StructScan(auth); nil != err {
			return nil, err
		}
		return auth, nil
	}
	return nil, nil
}

func (s *PGStore) GetByPub(id string) (*Authorization, error) {
	ctx, done := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer done()
	query := `SELECT * FROM authorizations WHERE deleted_at = '1970-01-01 00:00:00' AND public_key = $1`
	row := s.dbx.QueryRowxContext(ctx, query, id)
	if nil != row {
		auth := &Authorization{}
		if err := row.StructScan(auth); nil != err {
			return nil, err
		}
		return auth, nil
	}
	return nil, nil
}

func (s *PGStore) Delete(auth *Authorization) error {
	ctx, done := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer done()
	query := `
		UPDATE authorizations SET deleted_at = 'now'
		WHERE deleted_at = '1970-01-01 00:00:00' AND slug = $1
	`
	row, err := s.dbx.ExecContext(ctx, query, auth.Slug)
	if nil != err {
		return err
	}
	// PostgreSQL does support RowsAffected()
	if count, _ := row.RowsAffected(); count != 1 {
		return fmt.Errorf("record exists")
	}
	return nil
}

func (s *PGStore) Close() error {
	return s.dbx.DB.Close()
}
