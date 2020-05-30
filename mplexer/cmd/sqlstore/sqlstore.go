package main

import (
	"fmt"
	"log"

	"git.coolaj86.com/coolaj86/go-telebitd/mplexer/mgmt/authstore"
)

func main() {
	store, err := authstore.NewStore(nil)
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
