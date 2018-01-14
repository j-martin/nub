package github

import (
	"github.com/benchlabs/bub/core"
	"github.com/benchlabs/bub/utils"
	"io/ioutil"
	"path"
	"regexp"
	"strings"
)

type OwnerMap map[string][]string

func (gh *GitHub) PopulateOwners(m *core.Manifest) error {
	owners, err := gh.GetCodeOwners()
	if err != nil {
		return err
	}

	m.Owners = make(map[string][]core.User)
	for fPath, o := range owners {
		var ownerList []core.User
		for _, user := range o {
			u := core.User{}
			if strings.HasPrefix(user, "@") {
				u.GitHub = strings.TrimLeft(user, "@")
			} else {
				u.Email = user
			}
			gh.cfg.PopulateUser(&u)
			ownerList = append(ownerList, u)
		}
		m.Owners[fPath] = ownerList
	}
	return nil
}

func (gh *GitHub) GetCodeOwners() (owners OwnerMap, err error) {
	repo, err := core.MustInitGit("").GetRepositoryRootPath()
	if err != nil {
		return owners, err
	}
	for _, i := range []string{"", ".github", "docs"} {
		owners, err = readCodeOwner(repo, i)
		if err != nil && err != utils.FileDoesNotExist {
			return nil, err
		}
		if len(owners) > 0 {
			return owners, nil
		}
	}
	return owners, nil
}

func readCodeOwner(repo, dir string) (owners OwnerMap, err error) {
	filePath := path.Join(repo, dir, "CODEOWNERS")
	exists, err := utils.PathExists(filePath)
	if !exists {
		return owners, utils.FileDoesNotExist
	}
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return owners, err
	}
	return parseCodeOwnerContent(string(data))
}

func parseCodeOwnerContent(body string) (owners OwnerMap, err error) {
	owners = make(OwnerMap)
	re := regexp.MustCompile(" {2,}")
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		line := re.ReplaceAllString(strings.Trim(line, " "), " ")
		items := strings.Split(line, " ")
		ownershipPath := items[0]
		owners[ownershipPath] = items[1:]
	}
	return owners, err
}
