package authstore

import (
	"fmt"
	"strings"
	"testing"
)

func TestStore(t *testing.T) {
	// Note: test output is cached (running twice will not result in two records)

	connStr := "postgres://postgres:postgres@localhost/postgres"
	if strings.Contains(connStr, "@localhost/") || strings.Contains(connStr, "@localhost:") {
		connStr += "?sslmode=disable"
	} else {
		connStr += "?sslmode=required"
	}
	initSQL := "./init.sql"

	// TODO url.Parse

	store, err := NewStore(connStr, initSQL)
	if nil != err {
		t.Fatal("connection error", err)
		return
	}

	num := "8"
	slug := num + "-xxx-client"
	pubkey := num + "-somehash"
	auth1 := &Authorization{
		Slug:      slug,
		SharedKey: "3-xxxx-zzzz-yyyy",
		PublicKey: pubkey,
	}
	err = store.Add(auth1)
	if nil != err {
		t.Fatal("add error", err)
		return
	}

	auth, err := store.Get(slug)
	if nil != err {
		t.Fatal("get by slug error", err)
		return
	}

	auth, err = store.Get(pubkey)
	if nil != err {
		t.Fatal("get by pub error", err)
		return
	}

	auth1.MachinePPID = "a-secretish-id"
	err = store.Set(auth1)
	if nil != err {
		t.Fatal("set machine id", err)
		return
	}

	err = store.Delete(auth1)
	if nil != err {
		t.Fatal("set machine id", err)
		return
	}

	auth, err := store.Get(slug)
	if nil == err {
		t.Fatal("should get nothing back")
		return
	}

	store.Close()

	fmt.Printf("%#v\n", auth)
}
