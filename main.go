package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	pathlib "path"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

func copyToClipboard(val string) {
	cmd := exec.Command("xclip", "-selection", "clipboard")

	cmd.Stdin = strings.NewReader(val)

	if err := cmd.Run(); err != nil {
		panic(err)
	}
}

func main() {
	var ( // flags
		url   string
		path  string
		bench bool
	)

	flag.StringVar(&url, "url", "", "url to yaml")
	flag.StringVar(&path, "path", "", "path to yaml")
	flag.BoolVar(&bench, "bench", false, "print time spent to stderr")
	flag.Parse()

	// var start Time

	// if bench {
	// 	start = time.Now()
	// }

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

	contents := make(map[string]interface{})
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

	// if bench {
	// 	since
	// }

	var key string
	{ // fzf - prompt for key
		var homeDir string
		{ // get homeDir
			user, err := user.Current()
			if err != nil {
				panic(err)
			}
			homeDir = user.HomeDir
		}
		cmd := exec.Command("fzf",
			"--no-info",
			fmt.Sprintf("--history=%s",
				pathlib.Join(homeDir, ".history_ezautocompl")),
			// "--expect=tab",
		)

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

	val := contents[key]
	switch t := val.(type) {
	case map[interface{}]interface{}:
		{ // run cmd
			ival := val.(map[interface{}]interface{})
			val := make(map[string]interface{})
			for k, v := range ival {
				k2 := k.(string)
				val[k2] = v
			}
			// fmt.Printf("LOL%v\n", val)

			var cmds []string
			if s, ok := val["cmd"].(string); ok {
				// cmds = make([]string, 1)
				cmds = append(cmds, s)
			} else if rs, ok := val["cmd"].([]interface{}); ok {
				for _, v := range rs {
					cmds = append(cmds, v.(string))
				}
			} else {
				log.Fatalf("cmds: must either be of type string or []string, is %T\n", val["cmd"])
			}

			var stdin string
			if val, ok := val["stdin"]; ok {
				stdin = val.(string)
			}

			{ // run command
				cmd := exec.Command(cmds[0], cmds[1:]...)

				var stdoutBuf, stderrBuf bytes.Buffer
				cmd.Stdout = &stdoutBuf
				cmd.Stderr = &stderrBuf

				if stdin != "" {
					cmd.Stdin = strings.NewReader(stdin)
				}

				err := cmd.Run()
				if err != nil {
					if exitError, ok := err.(*exec.ExitError); ok {
						log.Fatalf("cmd %v returned %d with error:\nstdout: %s\nstderr: %s",
							cmds,
							exitError.ExitCode(),
							string(stdoutBuf.Bytes()),
							string(stderrBuf.Bytes()))
					}
				}

				outStr, _ := string(stdoutBuf.Bytes()), string(stderrBuf.Bytes())
				copyToClipboard(outStr)
			}
		}
	case string:
		copyToClipboard(val.(string))
	default:
		fmt.Printf("unexpected type %T\n", t)
	}

}
