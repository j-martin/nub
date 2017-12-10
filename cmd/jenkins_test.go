package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetJobName(t *testing.T) {
	t.Parallel()
	cfg := Configuration{}
	cfg.GitHub.Organization = "BenchLabs"
	j := Jenkins{cfg: &cfg}
	manifest := Manifest{Repository: "test", Branch: "master"}
	assert.Equal(t, "BenchLabs/job/test/job/master", j.getJobName(manifest))
}
