package main

import (
	"bufio"
	"fmt"
	"github.com/manifoldco/promptui"
	"log"
	"os"
	"os/exec"
	"strings"
)

func askForConfirmation(s string) bool {
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

func getEnvWithDefault(key string, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		val = defaultValue
	}
	return val
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func editFile(filePath string) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	cmd := exec.Command(editor, filePath)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func joinStringPointers(ptrs []*string, joinStr string) string {
	var arr []string
	for _, ref := range ptrs {
		if ref == nil {
			arr = append(arr, "")
		} else {
			arr = append(arr, *ref)
		}
	}
	return strings.Join(arr, joinStr)
}

func pickItem(label string, items []string) (string, error) {
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}:",
		Active:   "▶ {{ .}}",
		Inactive: "  {{ .}}",
		Selected: "▶ {{ .}}",
	}

	searcher := func(input string, index int) bool {
		i := items[index]
		name := strings.Replace(strings.ToLower(i), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)
		return strings.Contains(name, input)
	}

	prompt := promptui.Select{
		Size:      20,
		Label:     label,
		Items:     items,
		Templates: templates,
		Searcher:  searcher,
	}
	i, _, err := prompt.Run()
	return items[i], err
}
