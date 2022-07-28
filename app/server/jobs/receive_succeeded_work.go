package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Crossbell-Box/OperatorSync/app/server/consts"
	"github.com/Crossbell-Box/OperatorSync/app/server/global"
	"github.com/Crossbell-Box/OperatorSync/app/server/models"
	"github.com/Crossbell-Box/OperatorSync/app/server/types"
	commonConsts "github.com/Crossbell-Box/OperatorSync/common/consts"
	commonTypes "github.com/Crossbell-Box/OperatorSync/common/types"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

func ReceiveSucceededWork() error {
	_, err := global.MQ.Subscribe(commonConsts.MQSETTINGS_SucceededChannelName, handleSucceeded)
	if err != nil {
		global.Logger.Error("Failed to subscribe to MQ succeeded queue with error: ", err.Error())
		return err
	}

	//defer sub.Drain() // Ignore errors

	return nil
}

func handleSucceeded(m *nats.Msg) {
	global.Logger.Debug("New succeeded work received: ", string(m.Data))

	var workSucceeded commonTypes.WorkSucceeded
	if err := json.Unmarshal(m.Data, &workSucceeded); err != nil {
		global.Logger.Error("Unable to parse succeeded work: ", string(m.Data))
	} else {
		// Parse successfully
		// Parse feeds
		var feeds []models.Feed
		if len(workSucceeded.Feeds) > 0 {
			for _, rawFeed := range workSucceeded.Feeds {
				feed := models.Feed{
					Feed: types.Feed{
						AccountID:   workSucceeded.AccountID,
						Platform:    workSucceeded.Platform,
						CollectedAt: workSucceeded.DispatchAt,
						RawFeed:     rawFeed,
					},
				}
				feeds = append(feeds, feed)
			}
		}

		// Find account
		var account models.Account
		global.DB.First(&account, workSucceeded.AccountID)

		// Update account
		interv := workSucceeded.NewInterval
		platform := commonConsts.SUPPORTED_PLATFORM[workSucceeded.Platform]
		if interv < platform.MinRefreshGap {
			interv = platform.MinRefreshGap
		} else if interv > platform.MaxRefreshGap {
			interv = platform.MaxRefreshGap
		}
		account.LastUpdated = workSucceeded.SucceededAt
		account.UpdateInterval = interv
		account.NextUpdate = account.LastUpdated.Add(account.UpdateInterval)

		// Update character
		var character models.Character
		global.DB.First(&character, "crossbell_character = ?", account.CrossbellCharacter)

		if err := global.DB.Transaction(func(tx *gorm.DB) error {
			// do some database operations in the transaction (use 'tx' from this point, not 'db')
			if len(feeds) > 0 {
				platformSpecifiedFeed := models.Feed{
					Feed: types.Feed{
						Platform: workSucceeded.Platform,
					},
				}

				// Insert feeds
				if err := tx.Scopes(models.FeedTable(platformSpecifiedFeed)).Create(&feeds).Error; err != nil {
					return err
				}

				// Insert medias (Can only be processed here because we need feed IDs to identify them)
				mediaMap := make(map[string]models.Media)
				for _, feed := range feeds {
					for _, media := range feed.Media {
						var singleMedia models.Media
						var ok bool
						if singleMedia, ok = mediaMap[media.IPFSURI]; !ok {
							// Try to find in database
							if err = global.DB.First(&singleMedia, "ipfs_uri = ?", media.IPFSURI).Error; errors.Is(err, gorm.ErrRecordNotFound) {
								singleMedia = models.Media{
									ID:                 0,
									CrossbellCharacter: account.CrossbellCharacter,
									Media:              media,
								}
							}
						}
						singleMedia.RelatedFeeds = append(singleMedia.RelatedFeeds, models.FeedRecord{
							Platform: workSucceeded.Platform,
							ID:       feed.ID,
						})
						mediaMap[media.IPFSURI] = singleMedia
					}
				}

				var mediaUpdateList []models.Media
				var mediaCreateList []models.Media
				for _, singleMedia := range mediaMap {
					if singleMedia.ID == 0 {
						mediaCreateList = append(mediaCreateList, singleMedia)

						character.MediaUsage += singleMedia.FileSize
					} else {
						mediaUpdateList = append(mediaUpdateList, singleMedia)
					}
				}

				if len(mediaUpdateList) > 0 {
					if err := tx.Save(&mediaUpdateList).Error; err != nil {
						return err
					}
				}

				if len(mediaCreateList) > 0 {
					if err := tx.Create(&mediaCreateList).Error; err != nil {
						return err
					}
				}
			}

			// Update account
			if err := tx.Save(&account).Error; err != nil {
				return err
			}

			// Update character
			if err := tx.Save(&character).Error; err != nil {
				return err
			}

			// return nil will commit the whole transaction
			return nil
		}); err != nil {
			global.Logger.Error("Unable to save feeds: ", workSucceeded)
		} else {
			// Succeeded
			// Clear cache
			accountsCacheKey := fmt.Sprintf("%s:%s:%s", consts.CACHE_PREFIX, "accounts", account.CrossbellCharacter)
			feedsCacheKey := fmt.Sprintf("%s:%s:%d", consts.CACHE_PREFIX, "feeds", account.ID)
			mediasCacheKey := fmt.Sprintf("%s:%s:%s", consts.CACHE_PREFIX, "medias", account.CrossbellCharacter)
			clearCacheCtx := context.Background()
			global.Redis.Del(clearCacheCtx, accountsCacheKey) // To flush account update time
			global.Redis.Del(clearCacheCtx, feedsCacheKey)    // To flush cached feeds
			global.Redis.Del(clearCacheCtx, mediasCacheKey)   // To flush cached media list

			// Update metrics
			global.Metrics.Work.Succeeded.Inc(1)
		}
	}
}
