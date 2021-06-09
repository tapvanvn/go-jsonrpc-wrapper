package worker

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/tapvanvn/go-jsonrpc-wrapper/command"
)

var __id int = 0

type Tool struct {
	id             int
	proc           *exec.Cmd
	stdin          io.WriteCloser
	stdout         io.ReadCloser
	ready          bool
	openIn         bool
	openOut        bool
	commands       chan command.Command
	waitingCommand command.Command
	response       chan string
	firstCaret     bool
}

func (tool *Tool) scan() {
	s := bufio.NewScanner(tool.stdout)
	for s.Scan() {
		text := s.Text()
		tool.response <- text
	}
}

func (tool *Tool) processResponse() {

	go tool.scan()

	ticker := time.NewTicker(time.Microsecond * 10)
	defer func() {
		ticker.Stop()
	}()
	var cache = ""
	for {
		select {
		case text := <-tool.response:
			pos := strings.Index(text, ">")
			if pos > -1 {
				if tool.firstCaret {
					cache += text[pos+1:]
					if tool.waitingCommand != nil {
						if len(strings.TrimSpace(cache)) > 0 {

							re := regexp.MustCompile(`(\w*):`)
							cache = re.ReplaceAllString(cache, "\"$1\":")
							//fmt.Println("end by returned:", time.Now().Nanosecond(), tool.waitingCommand.GetID(), cache)
							inter := tool.waitingCommand.GetResponseInterface()
							err := json.Unmarshal([]byte(cache), inter)
							if err != nil {
								fmt.Println(err)
							}
							tool.waitingCommand.Debug()
							tool.waitingCommand = nil
							cache = ""
						}
					}
				} else {
					tool.firstCaret = true
					cache = text[pos+1:]
				}
			} else {

				cache += text

			}
			ticker.Reset(time.Second * 1)
		case <-ticker.C:
			if tool.waitingCommand != nil {

				if tool.firstCaret {
					//fmt.Println("end by timeout:", time.Now().Nanosecond(), tool.waitingCommand.GetID(), cache)
					inter := tool.waitingCommand.GetResponseInterface()
					re := regexp.MustCompile(`(\w*):`)
					cache = re.ReplaceAllString(cache, "\"$1\":")
					//fmt.Println("end by timeout:", time.Now().Nanosecond(), tool.waitingCommand.GetID(), cache)
					err := json.Unmarshal([]byte(cache), inter)

					if err != nil {
						fmt.Println(err)
					}
					tool.waitingCommand.Debug()
					tool.waitingCommand = nil
				}
			}
			cache = ""
		}
	}
}

func (tool *Tool) addCommand(cmd command.Command) {

	tool.commands <- cmd
}

func (tool *Tool) AddCommand(cmd command.Command) {
	tool.commands <- cmd
}

func (tool *Tool) processCommand() {

	for {

		if tool.waitingCommand != nil {
			time.Sleep(time.Microsecond * 20)
			continue
		}
		cmd, ok := <-tool.commands

		if !ok {
			break
		}

		tool.waitingCommand = cmd
		tool.stdin.Write([]byte(cmd.GetCommand() + "\n"))
	}
}

func (tool *Tool) Close() {
	if tool.openIn {
		tool.stdin.Close()
	}
	if tool.openOut {
		tool.stdout.Close()
	}
}

func NewTool(chain string) (*Tool, error) {

	__id += 1
	tool := &Tool{id: __id,
		ready:          false,
		openIn:         false,
		openOut:        false,
		commands:       make(chan command.Command),
		waitingCommand: nil,
		response:       make(chan string),
		firstCaret:     false,
	}

	var command string = ""

	if chain == "bsc" {
		command = "./bsc/geth"
	}

	if command == "" {
		return nil, errors.New("unknown chain")
	}

	tool.proc = exec.Command(command, "attach", "https://bsc-dataseed1.binance.org")

	stdin, _ := tool.proc.StdinPipe()

	tool.stdin = stdin
	tool.openIn = true

	stdout, _ := tool.proc.StdoutPipe()

	tool.stdout = stdout
	tool.openOut = true

	go tool.processResponse()

	err := tool.proc.Start()

	if err != nil {
		fmt.Println(err)
		tool.Close()
		return nil, err
	}

	go tool.processCommand()

	tool.ready = true

	return tool, nil
}
