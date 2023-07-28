package utils

import (
	"fmt"

	"github.com/tendermint/tendermint/libs/log"
	"github.com/syndtr/goleveldb/leveldb"
)

func SaveToDatabase(fid string, strcid string, db *leveldb.DB, logger log.Logger) error {
	err := db.Put(MakeDowntimeKey(strcid), []byte(fmt.Sprintf("%d", 0)), nil)
	if err != nil {
		logger.Error("Downtime Database Error: %v", err)
		return err
	}
	derr := db.Put(MakeFileKey(strcid), []byte(fid), nil)
	if derr != nil {
		logger.Error("File Database Error: %v", derr)
		return err
	}

	logger.Info(fmt.Sprintf("%s %s", fid, "Added to database"))

	_, cerr := db.Get(MakeFileKey(strcid), nil)
	if cerr != nil {
		logger.Error("Hash Database Error: %s", cerr.Error())
		return err
	}

	return nil
}
