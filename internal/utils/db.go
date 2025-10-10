package utils

import (
	"context"

	"github.com/fachebot/evm-grid-bot/internal/ent"
	"github.com/fachebot/evm-grid-bot/internal/logger"
)

func Tx(ctx context.Context, db *ent.Client, callback func(tx *ent.Tx) error) (err error) {
	tx, err := db.Tx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			if e := tx.Rollback(); e != nil {
				logger.Errorf("[Database] 回滚事务失败, %v", e)
			}
		}
	}()

	err = callback(tx)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		logger.Errorf("[Database] 提交事务失败, %v", err)
	}

	return nil
}
