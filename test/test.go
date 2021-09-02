package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/djherbis/atime"
	"github.com/olekukonko/tablewriter"
)

type FileBrowser struct {
	Files        []FileData     `json:"files"`
	IsFile       bool           `json:"is_file"`
	Permissions  PermissionJSON `json:"permissions"`
	Filename     string         `json:"name"`
	ParentPath   string         `json:"parent_path"`
	Success      bool           `json:"success"`
	FileSize     int64          `json:"size"`
	LastModified string         `json:"modify_time"`
	LastAccess   string         `json:"access_time"`
}

type PermissionJSON struct {
	Permissions FilePermission `json:"permissions"`
}

type FileData struct {
	IsFile       bool           `json:"is_file"`
	Permissions  PermissionJSON `json:"permissions"`
	Name         string         `json:"name"`
	FullName     string         `json:"full_name"`
	FileSize     int64          `json:"size"`
	LastModified string         `json:"modify_time"`
	LastAccess   string         `json:"access_time"`
}

type FilePermission struct {
	UID         int    `json:"uid"`
	GID         int    `json:"gid"`
	Permissions string `json:"permissions"`
	User        string `json:"user,omitempty"`
	Group       string `json:"group,omitempty"`
}

const (
	layoutStr = "01/02/2006 15:04:05"
)

func main() {
	//env test
	envs := env()
	fmt.Println(strings.Join(envs[:], "\n"))

	//ls test
	ls("/etc")
}

func env() []string {
	var env []string
	for _, e := range os.Environ() {
		env = append(env, e)
	}
	return env
}

func GetPermission(finfo os.FileInfo) FilePermission {
	perms := FilePermission{}
	perms.Permissions = finfo.Mode().Perm().String()
	systat := finfo.Sys().(*syscall.Stat_t)
	if systat != nil {
		perms.UID = int(systat.Uid)
		perms.GID = int(systat.Gid)
		tmpUser, err := user.LookupId(strconv.Itoa(perms.UID))
		if err == nil {
			perms.User = tmpUser.Username
		}
		tmpGroup, err := user.LookupGroupId(strconv.Itoa(perms.GID))
		if err == nil {
			perms.Group = tmpGroup.Name
		}
	}
	return perms
}

func ls(path string) []FileData {
	data := [][]string{}
	//var users []string

	var e FileBrowser
	abspath, _ := filepath.Abs(path)
	dirInfo, err := os.Stat(abspath)
	if err != nil {
		fmt.Println("Error")
	}
	e.IsFile = !dirInfo.IsDir()

	//p := FilePermission{}
	e.Permissions.Permissions = GetPermission(dirInfo)
	e.Filename = dirInfo.Name()
	e.ParentPath = filepath.Dir(abspath)
	if strings.Compare(e.ParentPath, e.Filename) == 0 {
		e.ParentPath = ""
	}
	e.FileSize = dirInfo.Size()
	e.LastModified = dirInfo.ModTime().Format(layoutStr)
	at, err := atime.Stat(abspath)
	if err != nil {
		e.LastAccess = ""
	} else {
		e.LastAccess = at.Format(layoutStr)
	}
	e.Success = true

	if dirInfo.IsDir() {
		files, err := ioutil.ReadDir(abspath)
		if err != nil {
			fmt.Println("Error")
		}

		fileEntries := make([]FileData, len(files))
		for i := 0; i < len(files); i++ {
			fileEntries[i].IsFile = !files[i].IsDir()
			fileEntries[i].Permissions.Permissions = GetPermission(files[i])
			fileEntries[i].Name = files[i].Name()
			fileEntries[i].FullName = filepath.Join(abspath, files[i].Name())
			fileEntries[i].FileSize = files[i].Size()
			fileEntries[i].LastModified = files[i].ModTime().Format(layoutStr)
			at, err := atime.Stat(abspath)
			if err != nil {
				fileEntries[i].LastAccess = ""
			} else {
				fileEntries[i].LastAccess = at.Format(layoutStr)
			}
		}
		e.Files = fileEntries
	}
	for _, f := range e.Files {
		row := []string{f.FullName, f.LastAccess, f.LastModified, f.Permissions.Permissions.User, f.Permissions.Permissions.Group, f.Permissions.Permissions.Permissions}
		data = append(data, row)
	}
	header := []string{"File", "LastAccess", "LastModified", "User", "Group", "Permissions"}
	tableData(data, header)
	return e.Files
}

func tableData(data [][]string, header []string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(header)

	for _, v := range data {
		table.Append(v)
	}
	table.Render() // Send output
}

func cp(src, dst string) (int64, error) {
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
