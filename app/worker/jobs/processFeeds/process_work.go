package processFeeds

import (
	"encoding/json"
	"github.com/Crossbell-Box/OperatorSync/app/worker/config"
	"github.com/Crossbell-Box/OperatorSync/app/worker/global"
	"github.com/Crossbell-Box/OperatorSync/app/worker/types"
	commonConsts "github.com/Crossbell-Box/OperatorSync/common/consts"
	commonTypes "github.com/Crossbell-Box/OperatorSync/common/types"
	"github.com/nats-io/nats.go"
	"strings"
	"time"
)

func ProcessFeeds(platformID string, cccs *types.ConcurrencyChannels) func(m *nats.Msg) {
	return func(m *nats.Msg) {

		global.Logger.Debug("New work received: ", string(m.Data))

		var workDispatched commonTypes.WorkDispatched

		if err := json.Unmarshal(m.Data, &workDispatched); err != nil {
			global.Logger.Error("Failed to parse dispatched work data.")
			// Even unable to report to failed works
			return
		}

		acceptTime := time.Now()

		collectLink := commonConsts.SUPPORTED_PLATFORM[workDispatched.Platform].FeedLink

		collectLink =
			strings.ReplaceAll(
				strings.ReplaceAll(
					collectLink,
					"{{rsshub_stateful}}",
					config.Config.RSSHubEndpointStateful,
				),
				"{{rsshub_stateless}}",
				config.Config.RSSHubEndpointStateless,
			)

		switch platformID {
		case "medium":
			feedsMedium(cccs, &workDispatched, acceptTime, collectLink)
		case "github":
			feedsGitHub(cccs, &workDispatched, acceptTime, collectLink)
		case "tiktok":
			feedsTikTok(cccs, &workDispatched, acceptTime, collectLink)
		default:
			// Unable to handle
			handleFailed(&workDispatched, acceptTime, commonConsts.ERROR_CODE_UNSUPPORTED_PLATFORM, "Unsupported platform")
		}
	}
}
