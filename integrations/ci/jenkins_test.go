package ci

import (
	"github.com/j-martin/bub/core"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetJobName(t *testing.T) {
	t.Parallel()
	cfg := core.Configuration{}
	cfg.Jenkins.Server = "https://something.com"
	cfg.GitHub.Organization = "BenchLabs"
	manifest := core.Manifest{Repository: "test", Branch: "master"}
	j := Jenkins{cfg: &cfg, manifest: &manifest}
	assert.Equal(t, "BenchLabs/job/test/job/master", j.getJobName())
}
