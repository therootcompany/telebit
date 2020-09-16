package main

import (
	"fmt"
	"log"
	"strings"

	"git.rootprojects.org/root/telebit/mgmt/authstore"
)

func main() {
	connStr := "postgres://postgres:postgres@localhost:5432/postgres"
	if strings.Contains(connStr, "@localhost/") || strings.Contains(connStr, "@localhost:") {
		connStr += "?sslmode=disable"
	} else {
		connStr += "?sslmode=required"
	}

	store, err := authstore.NewStore(connStr, initSQL)
	if nil != err {
		log.Fatal("connection error", err)
		return
	}

	num := "8"
	slug := num + "-xxx-client"
	pubkey := num + "-somehash"
	auth1 := &authstore.Authorization{
		Slug:      slug,
		SharedKey: "3-xxxx-zzzz-yyyy",
		PublicKey: pubkey,
	}
	err = store.Add(auth1)
	if nil != err {
		log.Fatal("add error", err)
		return
	}

	auth, err := store.Get(slug)
	if nil != err {
		log.Fatal("get by slug error", err)
		return
	}

	auth, err = store.Get(pubkey)
	if nil != err {
		log.Fatal("get by pub error", err)
		return
	}

	auth1.MachinePPID = "a-secretish-id"
	err = store.Set(auth1)
	if nil != err {
		log.Fatal("set machine id", err)
		return
	}

	err = store.Delete(auth1)
	if nil != err {
		log.Fatal("set machine id", err)
		return
	}

	store.Close()

	fmt.Printf("%#v\n", auth)
}
