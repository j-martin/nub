package core

import (
	"fmt"
	"github.com/imdario/mergo"
	"github.com/j-martin/nub/utils"
	"github.com/manifoldco/promptui"
	"github.com/tmc/keyring"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
	"strings"
)

const (
	ConfigUserFile   = "config.yml"
	ConfigSharedFile = "shared.yml"
)

type Environment struct {
	Prefix, Region, Domain string
	JumpHost               string `yaml:"jumphost"`
}

type User struct {
	Name, Slack, Email string
	GitHub             string `yaml:"github"`
}

type Configuration struct {
	Git struct {
		NoVerify bool `yaml:"noVerify"`
	}
	GitHub struct {
		Organization, Username, Token string
		Reviewers                     []string
	}
	Users []User
	JIRA  struct {
		Server, Username, Password string
		Project, Board             string
		Transitions                []JIRATransition
		Enabled                    bool
	}
	Jenkins    ServiceConfiguration
	Confluence ServiceConfiguration
	Vault      struct {
		AuthMethod, Server, Username, Password, Path string
	}
	Ssh struct {
		ConnectTimeout uint `yaml:"connectTimeout"`
	}
	ResetCredentials bool
}

type JIRATransition struct {
	Name, Alias string
}

type ServiceConfiguration struct {
	Server, Username, Password string
}

var config = `---
# use 'nub config --shared' to edit the shared config.
github:
	organization: benchlabs
	reviewers:
		# - reviewers (GitHub username) that will be applied to the PRs by default.

jenkins:
	server: "https://jenkins.example..com"

vault:
	server: "https://vault.example..com"
	path: "/secret/tool/nub"

confluence:
	server: "https://example.atlassian.net/wiki"

jira:
	server: "https://example.atlassian.net"
	project: # default project to use when creating issues.
	board: id of the board when creating issues in the current sprint.

ssh:
	connectTimeout: 3
`

func GetConfigString() string {
	return strings.Replace(config, "\t", "  ", -1)
}

func LoadConfiguration() (*Configuration, error) {
	baseCfg, err := loadConfiguration(ConfigSharedFile)
	if err != nil && err != utils.FileDoesNotExist {
		return nil, err
	}
	cfg, err := loadConfiguration(ConfigUserFile)
	if err != nil {
		return nil, err
	}
	err = mergo.Merge(baseCfg, *cfg)
	if err != nil {
		return nil, err
	}
	resetCredentials := os.Getenv("BUB_UPDATE_CREDENTIALS")
	if resetCredentials != "" {
		baseCfg.ResetCredentials = true
	}
	return baseCfg, nil
}

func getConfigPath(configFile string) (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", nil
	}

	configPath := path.Join(usr.HomeDir, ".config", "nub", configFile)
	return configPath, nil
}

func loadConfiguration(configFile string) (*Configuration, error) {
	cfg := &Configuration{}
	configPath, err := getConfigPath(configFile)
	if err != nil {
		return cfg, err
	}
	fileExists, _ := utils.PathExists(configPath)
	if !fileExists {
		return cfg, utils.FileDoesNotExist
	}

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Print("No nub configuration found. Please run `nub setup`")
		return cfg, err
	}

	err = yaml.Unmarshal(data, &cfg)
	return cfg, err
}

func EditConfiguration(configFile string) error {
	return utils.CreateAndEdit(GetConfigPath(configFile), GetConfigString())
}

func GetConfigPath(configFile string) string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return path.Join(usr.HomeDir, ".config", "nub", configFile)
}

func MustSetupConfig() {
	utils.Prompt("Setting up the base config. Just save and exit. Continue?")
	err := utils.CreateAndEdit(GetConfigPath(ConfigUserFile), GetConfigString())
	if err != nil {
		log.Fatal(err)
	}
}

func ShowConfig(cfg *Configuration) error {
	yml, _ := yaml.Marshal(cfg)
	fmt.Println(string(yml))
	return nil
}

func CheckServerConfig(server string) {
	if server == "" {
		log.Fatal("Server cannot be empty, make sure the config file is properly configured. run 'nub config'.")
	}
}

func LoadCredentials(item string, username, password *string, resetCredentials bool) (err error) {
	if err = LoadCredentialItem(item+" Username", username, resetCredentials); err != nil {
		return err
	}
	if err = LoadCredentialItem(item+" Password", password, resetCredentials); err != nil {
		return err
	}
	return nil
}

func LoadCredentialItem(item string, ptr *string, resetCredentials bool) (err error) {
	if resetCredentials {
		return setKeyringItem(item, ptr)
	}
	// e.g. "Confluence Username" -> "CONFLUENCE_USERNAME"
	envVar := os.Getenv(strings.Replace(strings.ToUpper(item), " ", "_", -1))
	if envVar != "" {
		*ptr = envVar
		return
	}

	if *ptr != "" && !strings.HasPrefix(*ptr, "<optional-") {
		return nil
	}

	return LoadKeyringItem(item, ptr)
}

func LoadKeyringItem(item string, ptr *string) (err error) {
	service := "nub"
	if pw, err := keyring.Get(service, item); err == nil {
		*ptr = pw
		return nil
	} else if err == keyring.ErrNotFound {
		return setKeyringItem(item, ptr)
	} else {
		return err
	}
}

func setKeyringItem(item string, ptr *string) (err error) {
	service := "nub"
	prompt := promptui.Prompt{
		Label: "Enter " + item,
	}
	if strings.HasSuffix(strings.ToLower(item), "password") {
		prompt.Mask = '*'
	}
	result, err := prompt.Run()
	if err != nil {
		return err
	}
	err = keyring.Set(service, item, string(result))
	if err != nil {
		return err
	}
	return LoadKeyringItem(item, ptr)
}

func equalAndNotEmpty(a, b string) bool {
	return a != "" && a == b
}

func (cfg *Configuration) PopulateUser(u *User) error {
	for _, userCfg := range cfg.Users {
		if equalAndNotEmpty(u.GitHub, userCfg.GitHub) ||
			equalAndNotEmpty(u.Name, userCfg.Name) ||
			equalAndNotEmpty(u.Slack, userCfg.Slack) {
			return mergo.Merge(u, userCfg)
		}
	}
	return nil
}
