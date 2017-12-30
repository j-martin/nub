package integrations

import (
	"github.com/benchlabs/bub/core"
	"github.com/benchlabs/bub/utils"
)

func OpenSplunk(cfg *core.Configuration, m *core.Manifest, isStaging bool) error {
	base := cfg.Splunk.Server +
		"/en-US/app/search/search/?dispatch.sample_ratio=1&earliest=rt-1h&latest=rtnow&q=search%20sourcetype%3D"
	var sourceType string
	if isStaging {
		sourceType = "staging"
	} else {
		sourceType = "pro"
	}
	sourceType = sourceType + "-" + m.Name + "*"
	return utils.OpenURI(base + sourceType)
}

