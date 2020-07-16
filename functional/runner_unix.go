package functional

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Netflix/go-expect"
)

func (scenario *Scenario) runStdinForUnix() (bytes.Buffer, error) {
	echo := strings.Fields(scenario.Steps[0].Value)
	rit := strings.Fields(scenario.Steps[1].Value)

	commandEcho := exec.Command("echo", echo...)
	commandRit := exec.Command("rit", rit...)

	pipeReader, pipeWriter := io.Pipe()
	commandEcho.Stdout = pipeWriter
	commandRit.Stdin = pipeReader

	var b2 bytes.Buffer
	commandRit.Stdout = &b2

	errorEcho := commandEcho.Start()
	if errorEcho != nil {
		log.Printf("Error while running: %q", errorEcho)
	}
	var stderr bytes.Buffer
	commandRit.Stderr = &stderr

	errorRit := commandRit.Start()
	if errorRit != nil {
		log.Printf("Error while running: %q", errorRit)
	}

	errorEcho = commandEcho.Wait()
	if errorEcho != nil {
		log.Printf("Error while running: %q", errorEcho)
	}

	pipeWriter.Close()

	errorRit = commandRit.Wait()
	if errorRit != nil {
		log.Printf("Error while running: %q", errorRit)
		b2 = stderr
	}

	fmt.Println(&b2)
	fmt.Println("--------")
	return b2, errorRit
}

func setUpRitSingleUnix() {
	fmt.Println("Running Setup for Unix..")

	fmt.Println("Running INIT")
	initStepEcho := Step{Key: "", Value: "{\"passphrase\":\"12345\"}", Action: "echo"}
	initStepRit := Step{Key: "", Value: "init --stdin", Action: "rit"}
	init := Scenario{Entry: "Running Init", Result: "", Steps: []Step{initStepEcho, initStepRit}}

	_, err := init.runStdinForUnix()
	if err != nil {
		log.Printf("Error when do init: %q", err)
	}

}

func setUpRitTeamUnix(){
	fmt.Println("Running Setup for Unix Team..")

	fmt.Println("Running INIT")
	initStepEcho := Step{Key: "", Value: "{\"organization\":\"zup\", \"url\":\"https://ritchie-server.itiaws.dev\"}", Action: "echo"}
	initStepRit := Step{Key: "", Value: "init --stdin", Action: "rit"}
	init := Scenario{Entry: "Running Init", Result: "", Steps: []Step{initStepEcho, initStepRit}}

	out, err := init.runStdinForUnix()
	if err != nil {
		log.Printf("Error when do init: %q", err)
	}
	fmt.Println(out)

	fmt.Println("Running Login")
	loginStepEcho := Step{Key: "", Value: "{\"username\":\"admin.ritchie\", \"password\":\"C@m@r0@m@r3l0\"}", Action: "echo"}
	loginStepRit := Step{Key: "", Value: "login --stdin", Action: "rit"}
	login := Scenario{Entry: "Running Init", Result: "", Steps: []Step{loginStepEcho, loginStepRit}}

	out, err = login.runStdinForUnix()
	if err != nil {
		log.Printf("Error when do Login: %q", err)
	}
	fmt.Println(out)
}

func setUpClearSetupUnix() {
	fmt.Println("Running Clear for Unix..")
	myPath := "/.rit/"
	usr, _ := user.Current()
	dir := usr.HomeDir + myPath

	d, err := os.Open(dir)
	if err != nil {
		log.Printf("Error Open dir: %q", err)
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		log.Printf("Error Readdirnames: %q", err)
	}
	for _, name := range names {
		err := os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			log.Printf("Error cleaning repo rit: %q", err)
		}
	}
}

func (scenario *Scenario) runStepsForUnix() (error, string) {
	args := strings.Fields(scenario.Steps[0].Value)
	cmd, c, out, err := execRit(args)
	if err != nil {
		panic(err)
	}

	defer c.Close()


	go func() {
		c.ExpectEOF()
	}()

	go func() {
		for _, step := range scenario.Steps {
			if step.Action == "sendkey" {
				sendKeys(step, out, c)
			}
			if step.Action == "newtype" {
				selectNewType(step, out, c)
			}
		}
	}()
	err = cmd.Start()
	if err != nil {
		panic(err)
	}

	err = cmd.Wait()
	if err != nil {
		panic(err)
	}

	fmt.Println(out.String())
	return err, out.String()
}


func execRit(args []string) (*exec.Cmd, *expect.Console, *bytes.Buffer, error) {

	cmd := exec.Command(rit, args...)
	out := new(bytes.Buffer)
	c, err := expect.NewConsole(expect.WithStdout(out))
	if err != nil {
		fmt.Println(err)
	}


	cmd.Stdin = c.Tty()
	cmd.Stdout = c.Tty()

	return cmd, c, out, err
}

func sendKeys(step Step, out *bytes.Buffer, c *expect.Console) {
	for {
		matched, _ := regexp.MatchString(step.Key, out.String())
		if matched {
			_, err := c.SendLine(step.Value + "\n")
			if err != nil {
				panic(err)
			}
			break
		}
		time.Sleep(1 * time.Second)
	}
}

func selectNewType(step Step, out *bytes.Buffer, c *expect.Console) {
	control := 0
	done := false
	for !done {
		matched, _ := regexp.MatchString(step.Key, out.String())
		if matched {
			resp := strings.Split(out.String(), "\r")
			for i, s := range resp {
				if (strings.Contains(s, "▸")) && i >= control {
					control = len(resp)
					selection, _ := regexp.MatchString("▸(.*)?"+step.Value+"(.*)+", s)
					if selection {
						_, err := c.SendLine("\n")
						if err != nil {
							panic(err)
						}
						done = true
						break
					} else {
						_, err := c.SendLine("j")
						if err != nil {
							panic(err)
						}
					}
				}
			}
		}
		time.Sleep(1 * time.Second)
	}
}
