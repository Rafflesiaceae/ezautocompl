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

	"github.com/atotto/clipboard"
	yaml "gopkg.in/yaml.v2"
)

func copyToClipboard(val string) {
	// dont copy empty strings
	if strings.TrimSpace(val) == "" {
		return
	}

	err := clipboard.WriteAll(val)
	if err != nil {
		panic(err)
	}
}

func main() {
	var err error
	var ( // flags
		url   string
		path  string
		bench bool
	)

	flag.StringVar(&url, "url", "", "url to yaml")
	flag.StringVar(&path, "path", "", "path to yaml")
	flag.BoolVar(&bench, "bench", false, "print time spent to stderr")
	flag.Parse()

	var rawContents []byte
	{ // read rawContents
		if path != "" { // read from file
			rawContents, err = ioutil.ReadFile(path)
			if err != nil {
				panic(err)
			}
		} else if url != "" { // fetch from http
			resp, err := http.Get(url)
			if err != nil {
				panic(err)
			}
			defer resp.Body.Close()

			rawContents, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				panic(err)
			}
		} else {
			rawContents, err = ioutil.ReadAll(os.Stdin)
			if err != nil {
				panic(err)
			}

		}
	}

	contents := make(map[string]interface{})
	// parse yaml
	if err := yaml.Unmarshal(rawContents, &contents); err != nil {
		panic(err)
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

	aval := contents[key]
	switch t := aval.(type) {
	case map[interface{}]interface{}:
		{ // run cmd
			val := make(map[string]interface{})
			{ // cast into val
				ival := aval.(map[interface{}]interface{})
				for k, v := range ival {
					k2 := k.(string)
					val[k2] = v
				}
			}

			// load yaml contents
			var cmds []string
			if s, ok := val["cmd"].(string); ok {
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

			var dontcopy bool
			if val, ok := val["copy-to-clipboard"]; ok {
				dontcopy = val == false
			}

			{ // run command
				cmd := exec.Command(cmds[0], cmds[1:]...)

				if dontcopy {
					cmd.Stdout = nil
					cmd.Stderr = nil

					err = cmd.Start()
					if err != nil {
						panic(err)
					}

					err = cmd.Process.Release()
					if err != nil {
						panic(err)
					}
				} else {
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
		}
	case string:
		copyToClipboard(aval.(string))
	default:
		fmt.Printf("unexpected type %T\n", t)
	}
}
