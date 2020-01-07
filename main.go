package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

func main() {
	var (
		url  string
		path string
	)

	flag.StringVar(&url, "url", "", "url to yaml")
	flag.StringVar(&path, "path", "", "path to yaml")
	flag.Parse()

	var rawContents []byte
	if path != "" { // read from file
		var err error

		rawContents, err = ioutil.ReadFile(path)
		if err != nil {
			panic(err)
		}
	} else { // fetch from http
		var err error

		resp, err := http.Get(url)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		rawContents, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}

	}

	contents := make(map[string]string)
	{ // parse yaml
		if err := yaml.Unmarshal(rawContents, &contents); err != nil {
			panic(err)
		}
	}

	var contentsKeys []string
	{ // compose contentsKeys
		contentsKeys = make([]string, len(contents))

		i := 0
		for k := range contents {
			contentsKeys[i] = k
			i++
		}
	}

	var key string
	{ // fzf - prompt for key
		cmd := exec.Command("fzf")

		var stdoutBuf bytes.Buffer

		cmd.Stdout = &stdoutBuf
		cmd.Stderr = os.Stderr
		cmd.Stdin = strings.NewReader(strings.Join(contentsKeys, "\n"))

		if err := cmd.Run(); err != nil {
			panic(err)
		}

		key = string(stdoutBuf.Bytes())
		key = strings.TrimSpace(key)
	}

	{ // copy to clipboard
		cmd := exec.Command("xclip", "-selection", "clipboard")

		cmd.Stdin = strings.NewReader(contents[key])

		if err := cmd.Run(); err != nil {
			panic(err)
		}
	}
}
