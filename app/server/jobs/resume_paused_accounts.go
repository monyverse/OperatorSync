package jobs

import (
	"errors"
	"github.com/Crossbell-Box/OperatorSync/app/server/global"
	"github.com/Crossbell-Box/OperatorSync/app/server/models"
	"github.com/Crossbell-Box/OperatorSync/app/server/types"
	"github.com/Crossbell-Box/OperatorSync/app/server/utils"
	"gorm.io/gorm"
	"sort"
	"time"
)

func ResumePausedAccounts() {
	global.Logger.Debug("Paused account resume work start dispatching...")
	go func() {
		t := time.NewTicker(10 * time.Minute)
		for {
			select {
			case <-t.C:
				tryToResumeAllPausedAccounts()
			}
		}
	}()
}

var (
	_isResumeWorkProcessing bool
)

func init() {
	_isResumeWorkProcessing = false
}

func tryToResumeAllPausedAccounts() {
	if _isResumeWorkProcessing {
		// No need to start another one, skip
		return
	}

	// Set busy flag
	_isResumeWorkProcessing = true

	global.Logger.Debug("Start trying to resume all paused accounts...")

	var pausedAccounts []models.Account

	global.DB.Find(&pausedAccounts, "is_onchain_paused = ?", true)

	for _, pa := range pausedAccounts {
		if tryToResumeOnePausedAccount(&pa) {
			// Recovered
			global.Logger.Debugf("Account #%d (%s@%s) recovered", pa.ID, pa.Username, pa.Platform)
			utils.AccountOnChainResume(&pa)
		} else {
			global.Logger.Errorf("Still unable to resume account #%d (%s@%s)", pa.ID, pa.Username, pa.Platform)

		}
	}

	// Unset busy flag
	_isResumeWorkProcessing = false

	global.Logger.Debug("All paused accounts checked, and already tried best to resume them.")
}

func tryToResumeOnePausedAccount(account *models.Account) bool {
	// Try to find the earliest paused feed
	global.Logger.Debugf("Try to find the first failed feed of account %s#%d", account.Platform, account.ID)

	var pausedFeeds models.FeedsArray
	if err := global.DB.Scopes(models.FeedTable(models.Feed{
		Feed: types.Feed{
			Platform: account.Platform,
		},
	})).Find(&pausedFeeds, "transaction = ? OR transaction IS NULL", "").Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Something is wrong
			global.Logger.Errorf("Something is wrong, failed to find the first paused feed of account %s#%d", account.Platform, account.ID)
			// But should be resumable, maybe?
			return true
		} else {
			global.Logger.Errorf("Failed to get first failed feed from database with error: %s", err.Error())
			return false
		}
	}

	sort.Sort(pausedFeeds)

	allSucceeded := true

	for index, feed := range pausedFeeds {
		// Recover feeds' media
		for _, mediaIPFSUri := range feed.MediaIPFSUris {
			var media models.Media
			global.DB.First(&media, "ipfs_uri = ?", mediaIPFSUri)
			feed.Media = append(feed.Media, media.Media)
		}

		// Try to push as many feeds as we can
		ipfsUri, tx, err := OneFeedOnChain(account, &feed)
		if err != nil {
			// Oops
			global.Logger.Errorf("Failed to OnCHain feed %s#%d with error: %s", account.Platform, feed.ID, err.Error())
			allSucceeded = false
			break
		} else {
			global.Logger.Debugf("Succeeded to OnChain feed %s#%d", account.Platform, feed.ID)
			pausedFeeds[index].IPFSUri = ipfsUri
			pausedFeeds[index].Transaction = tx
		}
	}

	// Save DB
	global.DB.Scopes(models.FeedTable(models.Feed{
		Feed: types.Feed{
			Platform: account.Platform,
		},
	})).Save(&pausedFeeds)

	return allSucceeded

}