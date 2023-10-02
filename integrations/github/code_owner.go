package github

import (
	"github.com/j-martin/nub/core"
	"github.com/j-martin/nub/utils"
	"io/ioutil"
	"path"
	"regexp"
	"strings"
)

type OwnerMap map[string][]string

func (gh *GitHub) PopulateOwners(m *core.Manifest) error {
	owners, err := gh.ListCodeOwners()
	if err != nil {
		return err
	}
	m.Owners = owners
	return nil
}

func (gh *GitHub) ListCodeOwners() (core.Ownership, error) {
	owners, err := gh.GetCodeOwners()
	if err != nil {
		return nil, err
	}

	ownerMap := make(core.Ownership)
	for fPath, pathOwners := range owners {
		var ownerList []core.User
		for _, user := range pathOwners {
			u := core.User{}
			if strings.HasPrefix(user, "@") {
				u.GitHub = strings.TrimLeft(user, "@")
			} else {
				u.Email = user
			}
			gh.cfg.PopulateUser(&u)
			ownerList = append(ownerList, u)
		}
		ownerMap[fPath] = ownerList
	}
	return ownerMap, nil
}

type Reviewers []string

func (gh *GitHub) ListReviewers() (reviewers Reviewers, err error) {
	reviewers = gh.cfg.GitHub.Reviewers
	owners, err := gh.ListCodeOwners()
	if err != nil {
		return nil, err
	}

	for _, filename := range core.MustInitGit("").ListFileChanged() {
		for rule, o := range owners {
			if matchesCodeOwnerRules(rule, filename) {
				for _, owner := range o {
					if owner.GitHub == gh.cfg.GitHub.Username {
						continue
					}
					reviewers = append(reviewers, owner.GitHub)
				}
			}
		}
	}
	return utils.RemoveDuplicatesUnordered(reviewers), nil
}

func matchesCodeOwnerRules(rule, filename string) bool {
	if rule == "*" {
		return true
	}
	if strings.HasPrefix(rule, "*") {
		return strings.HasSuffix(filename, rule[1:])
	}
	if strings.HasSuffix(rule, "*") && strings.HasPrefix(filename, rule[0:len(rule)-1]) {
		// If 'something/*', only the file on that level are included.
		return !strings.Contains(strings.TrimPrefix(filename, rule), "/")
	}
	return strings.HasPrefix(filename, rule)
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
