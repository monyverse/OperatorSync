package chain

import (
	crossbellContract "github.com/Crossbell-Box/OperatorSync/app/worker/chain/contract"
	"github.com/Crossbell-Box/OperatorSync/app/worker/global"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"strconv"
)

func PostNoteForCharacter(characterIdStr string, metadataUri string) (string, error) {
	characterId, err := strconv.ParseInt(characterIdStr, 10, 64)
	if err != nil {
		global.Logger.Errorf("Failed to parse character id with error: %s", err.Error())
		return "", err
	}

	// Prepare contract instance
	contractInstance, operatorAuth, err := Prepare()
	if err != nil {
		global.Logger.Errorf("Failed to prepare eth contract instance")
		return "", err
	}

	var tx *ethTypes.Transaction
	tx, err = contractInstance.PostNote(
		operatorAuth,
		crossbellContract.DataTypesPostNoteData{
			CharacterId: big.NewInt(characterId),
			ContentUri:  metadataUri,
		},
	)
	if err != nil {
		global.Logger.Errorf("Failed to create transaction")
		return "", err
	}

	return tx.Hash().Hex(), nil
}