package utils

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/mitchellh/go-wordwrap"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func GetTerminalSize() (uint, uint, error) {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}
	xy := strings.Split(strings.Trim(string(output), "\n"), " ")
	y, err := strconv.ParseUint(xy[0], 10, 32)
	if err != nil {
		return 0, 0, err
	}
	x, err := strconv.ParseUint(xy[1], 10, 32)
	if err != nil {
		return 0, 0, err
	}
	return uint(x), uint(y), err
}

func WordWrap(text string) string {
	x, _, err := GetTerminalSize()
	if err != nil {
		log.Printf("%v", err)
		return text
	}
	return wordwrap.WrapString(text, x-10)
}

func RunCmd(cmd string, args ...string) error {
	command := exec.Command(cmd, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

func RunCmdWithStdout(cmd string, args ...string) (string, error) {
	command := exec.Command(cmd, args...)
	command.Stderr = os.Stderr
	output, err := command.Output()
	return strings.Trim(string(output), "\n"), err
}

func RunCmdWithFullOutput(cmd string, args ...string) (string, error) {
	command := exec.Command(cmd, args...)
	var buf bytes.Buffer
	command.Stderr = &buf
	command.Stdout = &buf
	err := command.Run()
	return strings.Join(args, " ") + "\n" + strings.Trim(string(buf.String()), "\n"), err
}

func Prompt(message string) {
	fmt.Println("\n" + message)
	fmt.Print("Press 'Enter' to continue...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func AskForConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s [y/n]: ", s)
		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		response = strings.ToLower(strings.TrimSpace(response))
		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}
	}
}

func IsITerm() bool {
	return strings.HasPrefix(os.Getenv("TERM_PROGRAM"), "iTerm")
}

func SetBadge(name string) {
	if !IsITerm() {
		return
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(name))
	// https://www.iterm2.com/documentation-badges.html
	sequence := fmt.Sprintf("\033]1337;SetBadgeFormat=%v\033\\", encoded)
	print(sequence)
}

func ConfigureITerm(hostname string) {
	if !IsITerm() {
		return
	}
	// Set escape codes for iTerm2
	// https://www.iterm2.com/documentation-escape-codes.html
	SetBadge(hostname)
	if strings.HasPrefix(hostname, "pro") {
		// red for production
		print("\033]Ph501010\033\\")
	} else {
		// yellow for staging
		print("\033]Ph403010\033\\")
	}
}
func ResetITerm() {
	if !IsITerm() {
		return
	}
	print("\033]1337;SetProfile=Default\033\\")
	hostname, _ := os.Hostname()
	SetBadge(hostname)
}
