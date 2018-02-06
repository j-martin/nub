package core

import (
	"fmt"
	"github.com/benchlabs/bub/utils"
	"github.com/imdario/mergo"
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

type RDSConfiguration struct {
	Prefix, Database, User, Password string
}

type Environment struct {
	Prefix, Region, Domain string
	JumpHost               string `yaml:"jumphost"`
}

type User struct {
	Name, Slack, Email string
	GitHub             string `yaml:"github"`
}

type Configuration struct {
	AWS struct {
		Regions      []string
		RDS          []RDSConfiguration
		Environments []Environment
	}
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
	}
	Jenkins ServiceConfiguration
	Splunk  struct {
		Server string
	}
	Confluence ServiceConfiguration
	Circle     struct {
		Token string
	}
	Updates struct {
		Region, Bucket, Prefix string
	}
	Vault struct {
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
# use 'bub config --shared' to edit the shared config.

aws:
	regions:
		- us-east-1
		- us-west-2

	rds:
		# The first prefix match will be used.
		# The database name, unless specified, will be inferred from the host name.
		- prefix: staging
			database: <optional>
			user: <optional>
			password: <optional>

	environments:
		- prefix: staging2
			jumphost: jump.staging2.example.com
			domain: staging2.internal.example.com
			region: us-west-2
		- prefix: staging
			jumphost: jump.example.com
			region: us-west-2
			domain: staging.internal.example.com
		# if there is no prefix the last entry act as a catch all.
		- jumphost: jump.example.com
			region: us-east-1
			domain: production.internal.example.com

github:
	organization: benchlabs
	reviewers:
		# - reviewers (GitHub username) that will be applied to the PRs by default.

jenkins:
	server: "https://jenkins.example..com"

vault:
	server: "https://vault.example..com"
	path: "/secret/tool/bub"

confluence:
	server: "https://example.atlassian.net/wiki"

jira:
	server: "https://example.atlassian.net"
	project: # default project to use when creating issues.
	board: id of the board when creating issues in the current sprint.

splunk:
	server: "https://splunk.example.com"

circle:
	token: <optional-change-me>

updates:
	region: us-east-1
	bucket: s3bucket
	prefix: contrib/bub

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
	if len(cfg.AWS.Regions) == 0 {
		cfg.AWS.Regions = []string{"us-east-1", "us-west-2"}
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

	configPath := path.Join(usr.HomeDir, ".config", "bub", configFile)
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
		log.Print("No bub configuration found. Please run `bub setup`")
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
	return path.Join(usr.HomeDir, ".config", "bub", configFile)
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
		log.Fatal("Server cannot be empty, make sure the config file is properly configured. run 'bub config'.")
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
	service := "bub"
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
	service := "bub"
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
