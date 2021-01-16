package main

import (
	"encoding/json"
	"net/http"
	"time"
)

type command struct {
	ID      string
	Command string
	Agent   string
	Status  string
}

type Foo struct {
	Bar string
}

var myClient = &http.Client{Timeout: 10 * time.Second}

func main() {

	foo1 := new(Foo) // or &Foo{}
	getJson("http://localhost:8005/api/cmds/test", foo1)
	println(foo1.Bar)
}

func getJson(url string, target interface{}) error {
	r, err := myClient.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(target)
}
