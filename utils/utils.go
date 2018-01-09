package utils

import (
	"errors"
	"fmt"
	"github.com/manifoldco/promptui"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"time"
)

var (
	FileDoesNotExist = errors.New("file or directory does not exists")
)

func CurrentTimeForFilename() string {
	return time.Now().Format("2006-01-02T15-04-05Z")
}

func GetEnvWithDefault(key string, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		val = defaultValue
	}
	return val
}

func PathExists(filePath ...string) (bool, error) {
	_, err := os.Stat(path.Join(filePath...))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func EditFile(filePath string) {
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

func CreateAndEdit(filePath string, content string) {
	directory := path.Dir(filePath)
	dirExists, err := PathExists(directory)
	if err != nil {
		log.Fatal(err)
	}

	if !dirExists {
		os.MkdirAll(directory, 0700)
	}

	fileExists, err := PathExists(filePath)
	if err != nil {
		log.Fatal(err)
	}

	if !fileExists {
		log.Printf("Creating %s file.\n", filePath)
		ioutil.WriteFile(filePath, []byte(content), 0700)
	}

	fmt.Printf("Editing %s.", filePath)
	EditFile(filePath)
}

func JoinStringPointers(ptrs []*string, joinStr string) string {
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

func PickItem(label string, items []string) (string, error) {
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}:",
		Active:   "▶ {{ . }}",
		Inactive: "  {{ . }}",
		Selected: "▶ {{ . }}",
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

func Random(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}

func HasNonEmptyLines(lines []string) bool {
	for _, s := range lines {
		if s != "" {
			return true
		}
	}
	return false
}

func ConditionalOp(message string, noop bool, fn func() error) error {
	if noop {
		log.Printf("%v (noop)", message)
		return nil
	}
	log.Printf(message)
	return fn()
}

func InRepository() bool {
	result, err := PathExists(".git")
	if err != nil {
		return false
	}
	return result
}

func IsRepository(repoDir string) bool {
	result, err := PathExists(repoDir, ".git")
	if err != nil {
		return false
	}
	return result
}

func OpenURI(uriSegments ...string) error {
	uri := strings.Join(uriSegments, "/")
	log.Printf("Opening: %v", uri)
	if runtime.GOOS == "darwin" {
		return exec.Command("open", uri).Run()
	} else if runtime.GOOS == "linux" {
		return exec.Command("xdg-open", uri).Run()
	}
	return errors.New("could not open the link automatically")
}
