package mgmt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"

	telebit "git.coolaj86.com/coolaj86/go-telebitd/mplexer"
	"git.coolaj86.com/coolaj86/go-telebitd/mplexer/mgmt/authstore"
)

type SuccessResponse struct {
	Success bool `json:"success"`
}

func Ping(authURL, token string) error {
	msg, err := telebit.Request("POST", authURL+"/ping", token, nil)
	if nil != err {
		return err
	}
	if nil == msg {
		return fmt.Errorf("invalid response")
	}
	resp := SuccessResponse{}
	err = json.NewDecoder(msg).Decode(&resp)
	if err != nil {
		return err
	}
	if true != resp.Success {
		return fmt.Errorf("expected successful response")
	}
	return nil
}

func Register(authURL, secret, ppid string) (kid string, err error) {
	pub := authstore.ToPublicKeyString(ppid)
	jsonb := bytes.NewBuffer([]byte(
		fmt.Sprintf(`{ "machine_ppid": "%s", "public_key": "%s" }`, ppid, pub),
	))
	msg, err := telebit.Request("POST", authURL+"/register-device/"+secret, "", jsonb)
	if nil != err {
		return "", err
	}
	if nil == msg {
		return "", fmt.Errorf("invalid response")
	}

	auth := &authstore.Authorization{}
	err = json.NewDecoder(msg).Decode(auth)
	if err != nil {
		return "", err
	}
	msgBytes, _ := ioutil.ReadAll(msg)
	if "" == auth.PublicKey {
		return "", fmt.Errorf("unexpected server response: no public key: %s", string(msgBytes))
	}
	if pub != auth.PublicKey {
		return "", fmt.Errorf("server disagrees about public key id: %s vs %s", kid, auth.PublicKey)
	}
	return auth.PublicKey, nil
}
