package cat

import (
	"fmt"
	"io/ioutil"
)

func Cat(filename string) string {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println("File reading error", err)
		return "Error reading file " + filename
	}
	fmt.Println("Contents of file:", string(data))
	return string(data)
}
