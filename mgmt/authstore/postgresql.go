package authstore

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"time"

	"git.rootprojects.org/root/telebit/files"

	"github.com/jmoiron/sqlx"
	// pq injects itself into sql as 'postgres'
	_ "github.com/lib/pq"
)

func NewStore(pgURL, initSQL string) (Store, error) {
	// https://godoc.org/github.com/lib/pq

	f, err := files.Open(initSQL)
	if nil != err {
		return nil, err
	}

	dbtype := "postgres"
	sqlBytes, err := ioutil.ReadAll(f)
	if nil != err {
		return nil, err
	}

	ctx, done := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer done()
	db, err := sql.Open(dbtype, pgURL)
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

func (s *PGStore) SetMaster(secret string) error {
	ctx, done := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer done()

	pub := ToPublicKeyString(secret)
	auth := &Authorization{
		Slug:        "*",
		SharedKey:   secret,
		MachinePPID: secret,
		PublicKey:   pub,
	}
	err := s.Add(auth)

	query := `
		UPDATE authorizations SET
			machine_ppid=$1,
			shared_key=$1,
			public_key=$2,
      deleted_at='1970-01-01 00:00:00'
		WHERE slug = '*'
	`
	_, err = s.dbx.ExecContext(ctx, query, auth.MachinePPID, auth.PublicKey)
	return err
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
	now := time.Now()
	res, err := tx.ExecContext(ctx, query2, auth.Slug, auth.SharedKey, auth.PublicKey)
	if nil != err {
		return err
	}

	// PostgreSQL does support RowsAffected(), but not LastInsertId()
	if count, _ := res.RowsAffected(); count != 1 {
		// TODO be more sure?
		return ErrExists // fmt.Errorf("record not added (probably exists)")
	}

	if err := tx.Commit(); nil != err {
		return err
	}

	auth.CreatedAt = now
	auth.UpdatedAt = now
	return nil
}

func (s *PGStore) Set(auth *Authorization) error {
	ctx, done := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer done()
	query := `
		UPDATE authorizations SET
			machine_ppid = $1,
			shared_key = $2,
			public_key = $3,
			updated_at = 'now'
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

func (s *PGStore) Touch(pub string) error {
	ctx, done := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer done()
	query := `
		UPDATE authorizations SET
			updated_at = 'now'
		WHERE deleted_at = '1970-01-01 00:00:00'
			AND (public_key = $1 OR slug = $1)
	`
	row, err := s.dbx.ExecContext(ctx, query, pub)
	if nil != err {
		return err
	}
	// PostgreSQL is one of the databases for which RowsAffected() IS supported
	if count, _ := row.RowsAffected(); count != 1 {
		return ErrNotFound
	}
	return nil
}

func (s *PGStore) Active() ([]Authorization, error) {
	ctx, done := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer done()

	auths := []Authorization{}
	query := `
		SELECT * FROM authorizations
		WHERE deleted_at = '1970-01-01 00:00:00'
		  AND updated_at > $1
	`
	ago15Min := time.Now().Add(-15 * time.Minute)
	err := s.dbx.SelectContext(ctx, &auths, query, ago15Min)
	if nil != err {
		return nil, err
	}
	return auths, nil
}

func (s *PGStore) Inactive() ([]Authorization, error) {
	ctx, done := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer done()

	auths := []Authorization{}
	query := `
		SELECT * FROM authorizations
		WHERE deleted_at = '1970-01-01 00:00:00'
		  AND updated_at <= $1
		  AND slug != '*'
	`
	ago15Min := time.Now().Add(-15 * time.Minute)
	err := s.dbx.SelectContext(ctx, &auths, query, ago15Min)
	if nil != err {
		return nil, err
	}
	return auths, nil
}

func (s *PGStore) Get(id string) (*Authorization, error) {
	ctx, done := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer done()
	query := `
		SELECT * FROM authorizations
		WHERE deleted_at = '1970-01-01 00:00:00'
		  AND (slug = $1 OR public_key = $1 OR public_key = $2)
	`
	// if the id is actually the secret, we want the public form
	// (we do this to protect against a timing attack)
	pubby := ToPublicKeyString(id)
	if len(id) > 24 {
		id = id[:24]
	}
	row := s.dbx.QueryRowxContext(ctx, query, id, pubby)
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
		return fmt.Errorf("record does not exist")
	}
	return nil
}

func (s *PGStore) Close() error {
	return s.dbx.DB.Close()
}
