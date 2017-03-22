package envelope

import (
	"bytes"
	"encoding/json"
	"io"
	"time"
)

//Envelope -- Standard daplie response structure
type Envelope struct {
	TransactionType      string      `json:"type"`
	Schema               string      `json:"schema"`
	TransactionTimeStamp int64       `json:"txts"`
	TransactionID        int64       `json:"txid"`
	Error                string      `json:"error"`
	ErrorDescription     string      `json:"error_description"`
	ErrorURI             string      `json:"error_uri"`
	Result               interface{} `json:"result"`
}

//NewEnvelope -- Constructor
func NewEnvelope(transactionType string) (p *Envelope) {
	transactionID++

	p = new(Envelope)
	p.TransactionType = transactionType
	p.TransactionID = transactionID
	p.TransactionTimeStamp = time.Now().Unix()
	p.Error = "ok"

	return
}

//Generate -- encode into JSON and return string
func (e *Envelope) Generate() string {
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(e)
	return buf.String()
}

//GenerateWriter --
func (e *Envelope) GenerateWriter(w io.Writer) {
	json.NewEncoder(w).Encode(e)
}
