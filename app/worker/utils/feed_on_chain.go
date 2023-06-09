package utils

import (
	"encoding/json"
	"fmt"
	"github.com/Crossbell-Box/OperatorSync/app/worker/chain"
	"github.com/Crossbell-Box/OperatorSync/app/worker/global"
	"github.com/Crossbell-Box/OperatorSync/app/worker/indexer"
	"github.com/Crossbell-Box/OperatorSync/app/worker/types"
	commonConsts "github.com/Crossbell-Box/OperatorSync/common/consts"
	commonTypes "github.com/Crossbell-Box/OperatorSync/common/types"
	md "github.com/JohannesKaufmann/html-to-markdown"
	"strconv"
)

func FeedOnChain(work *commonTypes.OnChainRequest) (string, string, int64, int64, error) {
	// Step -1: Check if feed with this uri (if any) is already on chain
	if ValidateUri(work.Link) {
		// Get feed with link from indexer
		ipfsUri, tx, characterId, noteId, err := indexer.GetFeedWithLinkFromIndexer(work.Link)
		if err == nil && ipfsUri != "" && tx != "" {
			// Already post
			return ipfsUri, tx, characterId, noteId, nil
		}
	}

	// Step -0.5: What if feed is postNote4AnyUri, and target is on CrossBell?
	var (
		forNoteCharacterId int64 = 0
		forNoteNoteId      int64 = 0
	)
	if work.ForCharacterID > 0 && work.ForNoteID > 0 {
		forNoteCharacterId = work.ForCharacterID
		forNoteNoteId = work.ForNoteID
	} else if ValidateUri(work.ForURI) {
		_, _, forNoteCharacterId, forNoteNoteId, _ = indexer.GetFeedWithLinkFromIndexer(work.ForURI)
	}

	// Step 0: Prepare platform
	platform := commonConsts.SUPPORTED_PLATFORM[work.Platform]

	// Step 1: Parse feeds to note metadata
	metadata := types.NoteMetadata{
		Type: "note",
		Authors: append(
			work.Authors,
			fmt.Sprintf("csb://account:%s@%s", work.Username, work.Platform),
		),
		Title: StringPointerOmitEmpty(work.Title),
		Tags:  work.Categories,
		Sources: []string{
			"xSync",
			platform.Name,
		},
		ContentWarning: StringPointerOmitEmpty(work.ContentWarning),
		DatePublished:  StringPointerOmitEmpty(work.PublishedAt.Format("2006-01-02T15:04:05Z")),
	}

	if platform.HTML2Markdown {
		converter := md.NewConverter("", true, nil)
		mdContent, err := converter.ConvertString(work.Content)
		if err != nil {
			// Failed to parse
			global.Logger.Errorf("Failed to parse html to markdown for feed (%s-%d) with error: %s", work.Platform, work.FeedID, err.Error())
			metadata.Content = StringPointerOmitEmpty(work.Content)
		} else {
			metadata.Content = StringPointerOmitEmpty(mdContent)
		}
	} else {
		metadata.Content = StringPointerOmitEmpty(work.Content)
	}

	if ValidateUri(work.Link) {
		metadata.ExternalUrls = []string{work.Link}
	}

	if platform.IsMediaAttachments {
		for _, media := range work.Media {
			// Append basic info
			attachment := types.NoteAttachment{
				Name:    StringPointerOmitEmpty(media.FileName),
				Address: StringPointerOmitEmpty(media.IPFSUri),
				//Content:  "",
				MimeType: StringPointerOmitEmpty(media.ContentType),
				FileSize: &media.FileSize,
				Alt:      StringPointerOmitEmpty(media.FileName), // Just use filename for now
			}

			// Append additional props
			var additionalProps map[string]string
			err := json.Unmarshal([]byte(media.AdditionalProps), &additionalProps)
			if err != nil {
				global.Logger.Errorf("Failed to parse additional props for media #%s with error: %s", media.IPFSUri, err.Error())
			} else {
				// If has "width" and "height"
				if mediaWidthStr, ok := additionalProps["width"]; ok {
					mediaWidth, err := strconv.Atoi(mediaWidthStr)
					if err != nil {
						global.Logger.Errorf("Failed to parse width of media #%s with error: %s", media.IPFSUri, err.Error())
					} else {
						uintMediaWidth := uint(mediaWidth)
						attachment.Width = &uintMediaWidth
					}
				}
				if mediaHeightStr, ok := additionalProps["height"]; ok {
					mediaHeight, err := strconv.Atoi(mediaHeightStr)
					if err != nil {
						global.Logger.Errorf("Failed to parse height of media #%s with error: %s", media.IPFSUri, err.Error())
					} else {
						uintMediaHeight := uint(mediaHeight)
						attachment.Width = &uintMediaHeight
					}
				}
			}

			// Append to array
			metadata.Attachments = append(metadata.Attachments, attachment)
		}
	}

	// Step 2: Upload note metadata to IPFS, get IPFS Uri as note ContentUri
	metaBytes, err := json.Marshal(&metadata)
	if err != nil {
		global.Logger.Errorf("Failed to parse metadata to json with error: %s", err.Error())
		return "", "", 0, 0, err
	}
	ipfsUri, _, err := UploadBytesToIPFS(metaBytes, "metadata.json")
	if err != nil {
		global.Logger.Errorf("Failed to upload metadata to IPFS with error: %s", err.Error())
		return "", "", 0, 0, err
	}

	// Step 3: Upload note to Crossbell Chain with ContentUri
	tx, characterId, noteId, err := chain.PostNoteForCharacter(work.CrossbellCharacterID, ipfsUri, work.ForURI, forNoteCharacterId, forNoteNoteId)
	if err != nil {
		global.Logger.Errorf("Failed to post note to Crossbell chain for character #%s with error: %s", work.CrossbellCharacterID, err.Error())
		return ipfsUri, tx, characterId, noteId, err // Transaction might be invalid
	}

	// Step 4: Return Transaction hash and done!
	return ipfsUri, tx, characterId, noteId, nil
}
