package eth

import (
	"context"
	"sync"

	"github.com/fachebot/evm-grid-bot/internal/ent"
	"github.com/fachebot/evm-grid-bot/internal/logger"
	"github.com/fachebot/evm-grid-bot/internal/model"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type NonceManager struct {
	mutex     sync.Mutex
	userLocks map[string]*sync.Mutex
	dbClient  *ent.Client
	ethClient *ethclient.Client
}

type NonceConsumeFunc func(ctx context.Context, nonce uint64) (hash string, err error)

func NewNonceManager(dbClient *ent.Client, ethClient *ethclient.Client) *NonceManager {
	return &NonceManager{
		dbClient:  dbClient,
		ethClient: ethClient,
		userLocks: make(map[string]*sync.Mutex),
	}
}

func (m *NonceManager) Request(ctx context.Context, account common.Address, consume NonceConsumeFunc) error {
	m.mutex.Lock()
	userMutex, ok := m.userLocks[account.Hex()]
	if !ok {
		userMutex = new(sync.Mutex)
		m.userLocks[account.Hex()] = userMutex
	}
	m.mutex.Unlock()

	userMutex.Lock()
	defer userMutex.Unlock()

	nextNonce, err := m.ethClient.PendingNonceAt(ctx, account)
	if err != nil {
		return err
	}

	nonceModel := model.NewNonceModel(m.dbClient.Nonce)
	storedNonce, findErr := nonceModel.FindOne(ctx, account.Hex())
	if findErr != nil && !ent.IsNotFound(findErr) {
		return findErr
	}

	if findErr == nil && storedNonce.Nonce >= nextNonce {
		nextNonce = storedNonce.Nonce + 1
	}

	_, err = consume(ctx, nextNonce)
	if err == nil {
		var err2 error
		if ent.IsNotFound(findErr) {
			err2 = nonceModel.Save(ctx, account.Hex(), nextNonce)
		} else {
			err2 = nonceModel.UpdateNonce(ctx, account.Hex(), nextNonce)
		}

		if err2 != nil {
			logger.Errorf("[NonceManager] 更新账户nonce失败, account: %s, nonce: %d, %+v", account, nextNonce, err2)
		}
	}

	return err
}
