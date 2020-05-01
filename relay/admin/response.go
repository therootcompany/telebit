package admin

import (
	"bytes"
	"encoding/json"
	"io"
	"time"
)

//Response -- Standard response structure
type Response struct {
	TransactionType      string      `json:"type"`
	Schema               string      `json:"schema"`
	TransactionTimeStamp int64       `json:"txts"`
	TransactionID        int64       `json:"txid"`
	Error                string      `json:"error"`
	ErrorDescription     string      `json:"error_description"`
	ErrorURI             string      `json:"error_uri"`
	Result               interface{} `json:"result"`
}

//NewResponse -- Constructor
func NewResponse(transactionType string) (p *Response) {
	// TODO BUG use atomic
	transactionID++

	p = &Response{}
	p.TransactionType = transactionType
	p.TransactionID = transactionID
	p.TransactionTimeStamp = time.Now().Unix()
	p.Error = "ok"

	return
}

//Generate -- encode into JSON and return string
func (e *Response) Generate() string {
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(e)
	return buf.String()
}

//GenerateWriter --
func (e *Response) GenerateWriter(w io.Writer) {
	json.NewEncoder(w).Encode(e)
}
