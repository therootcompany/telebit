package mgmt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"git.rootprojects.org/root/telebit/internal/dbg"
	"git.rootprojects.org/root/telebit/internal/mgmt/authstore"
	"git.rootprojects.org/root/telebit/internal/telebit"
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
	jsons := fmt.Sprintf(`{ "machine_ppid": "%s", "public_key": "%s" }`, ppid, pub)
	jsonb := bytes.NewBuffer([]byte(jsons))
	fullURL := authURL + "/register-device/" + secret
	if dbg.Debug {
		fmt.Fprintf(os.Stderr, "[debug] authURL=%s, secret=%s, ppid=%s\n", fullURL, secret, jsons)
	}
	msg, err := telebit.Request("POST", fullURL, "", jsonb)
	if nil != err {
		return "", err
	}
	if nil == msg {
		return "", fmt.Errorf("invalid response")
	}

	auth := &authstore.Authorization{}
	msgBytes, _ := ioutil.ReadAll(msg)
	//err = json.NewDecoder(msg).Decode(auth)
	err = json.Unmarshal(msgBytes, auth)
	if err != nil {
		return "", err
	}
	//msgBytes, _ := ioutil.ReadAll(msg)
	if "" == auth.PublicKey {
		return "", fmt.Errorf("unexpected server response: no public key: %s", string(msgBytes))
	}
	if pub != auth.PublicKey {
		return "", fmt.Errorf("server disagrees about public key id: %s vs %s", kid, auth.PublicKey)
	}
	return auth.PublicKey, nil
}
