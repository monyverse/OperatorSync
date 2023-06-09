package utils

import (
	"github.com/Crossbell-Box/OperatorSync/app/worker/global"
	"github.com/Crossbell-Box/OperatorSync/common/types"
	"html"
	"sync"
)

func UploadAllMedia(mediaUris []string) []types.Media {

	// Collect all unique media URIs
	mediaUriSet := make(map[string]struct{})
	for _, rawMediaUri := range mediaUris {
		if _, ok := mediaUriSet[rawMediaUri]; !ok {
			mediaUriSet[rawMediaUri] = struct{}{}
		}
	}

	// Upload them all
	var ipfsUploadWg sync.WaitGroup
	ipfsUploadResultChannel := make(chan types.Media, len(mediaUriSet))
	for uri := range mediaUriSet {
		innerUri := uri
		ipfsUploadWg.Add(1)
		go func() {
			media, err := UploadOneMedia(innerUri)
			if err != nil {
				global.Logger.Error("Failed to upload link (", innerUri, ") onto IPFS: ", err.Error())
			} else {
				ipfsUploadResultChannel <- *media
			}
			ipfsUploadWg.Done()
		}()
	}

	// Collect results
	ipfsUploadWg.Wait()
	close(ipfsUploadResultChannel)
	var medias []types.Media
	for media := range ipfsUploadResultChannel {
		medias = append(medias, media)
	}

	return medias

}

func UploadOneMedia(mediaUri string) (*types.Media, error) {
	unescapedUri := html.UnescapeString(mediaUri)
	media := types.Media{
		OriginalURI: unescapedUri,
	}
	var err error
	if media.FileName, media.IPFSUri, media.FileSize, media.ContentType, media.AdditionalProps, err = UploadURLToIPFS(media.OriginalURI, false); err != nil {
		return nil, err
	} else {
		return &media, nil
	}
}
