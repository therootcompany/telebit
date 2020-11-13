package authstore

import (
	"fmt"
	"testing"
)

func TestStore(t *testing.T) {
	// Note: test output is cached (running twice will not result in two records)

	store, err := NewStore(connStr, initSQL)
	if nil != err {
		t.Fatal("connection error", err)
		return
	}

	num := "8"
	slug := num + "-xxx-client"
	secret := "3-xxxx-zzzz-yyyy"
	pubkey := ToPublicKeyString(secret)
	auth1 := &Authorization{
		Slug:      slug,
		SharedKey: secret,
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

	auth, err = store.Get(secret)
	if nil != err {
		t.Fatal("get by key error", err)
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

	auth, err = store.Get(slug)
	if nil == err {
		t.Fatal("should get nothing back")
		return
	}

	store.Close()

	fmt.Printf("%#v\n", auth)
}
