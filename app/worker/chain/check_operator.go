package chain

import (
	"github.com/Crossbell-Box/OperatorSync/app/worker/global"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"math/big"
	"strconv"
)

func CheckOperator(characterIdStr string) (bool, error) {
	// Prepare character
	characterId, err := strconv.ParseInt(characterIdStr, 10, 64)
	if err != nil {
		global.Logger.Errorf("Failed to parse character id with error: %s", err.Error())
		return false, err
	}

	// Prepare contract instance
	_, contractInstance, operatorAuth, err := Prepare()
	if err != nil {
		global.Logger.Errorf("Failed to prepare eth contract instance")
		return false, err
	}

	// Do query
	operatorPermissions, err := contractInstance.GetOperatorPermissions(&bind.CallOpts{}, big.NewInt(characterId), operatorAuth.From)
	if err != nil {
		global.Logger.Errorf("Failed to check if is operator of character %s with error: %s", characterIdStr, err.Error())
		return false, err
	}

	// Check contract definitions at https://github.com/Crossbell-Box/Crossbell-Contracts/blob/main/contracts/libraries/OP.sol#L63
	return operatorPermissions.Bit(236) == 1, nil

}
