package event

import (
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type ELB struct {
	RequestContext        struct{}          `json:"requestContext"`
	HttpMethod            string            `json:"httpMethod"`
	Path                  string            `json:"path"`
	QueryStringParameters struct{}          `json:"queryStringParameters"`
	Headers               map[string]string `json:"headers"`
	Body                  string            `json:"body"`
	IsBase64Encoded       bool              `json:"isBase64Encoded"`
}

// Structure the data comming from the ELB.
func NewELB(r *http.Request) ELB {
	//read message body, datatype of body []byte
	bod, err := ioutil.ReadAll(r.Body)
	body := string(bod)
	defer func() {
		err = r.Body.Close()
		if err != nil {
			log.Printf("unable to close body")
		}
	}()
	if err != nil {
		return ELB{}
	}
	elbEvent := ELB{}
	elbEvent.HttpMethod = r.Method
	elbEvent.Path = r.URL.Path
	elbEvent.Body = body
	elbEvent.IsBase64Encoded = true

	elbEvent.Headers = make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			elbEvent.Headers[strings.ToLower(k)] = v[0]
		}
	}
	return elbEvent
}
