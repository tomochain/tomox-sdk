package services

import (
	"fmt"
	"math/big"

	"github.com/tomochain/tomox-sdk/interfaces"
	"github.com/tomochain/tomox-sdk/types"
	"github.com/tomochain/tomox-sdk/utils"
	"github.com/tomochain/tomox-sdk/utils/math"
)

type ValidatorService struct {
	ethereumProvider interfaces.EthereumProvider
	accountDao       interfaces.AccountDao
	orderDao         interfaces.OrderDao
	lendingDao       interfaces.LendingOrderDao
	pairDao          interfaces.PairDao
	tokenDao         interfaces.TokenDao
}

func NewValidatorService(
	ethereumProvider interfaces.EthereumProvider,
	accountDao interfaces.AccountDao,
	orderDao interfaces.OrderDao,
	lendingDao interfaces.LendingOrderDao,
	pairDao interfaces.PairDao,
	tokenDao interfaces.TokenDao,
) *ValidatorService {

	return &ValidatorService{
		ethereumProvider,
		accountDao,
		orderDao,
		lendingDao,
		pairDao,
		tokenDao,
	}
}

// ValidateAvailablExchangeBalance get balance
func (s *ValidatorService) ValidateAvailablExchangeBalance(o *types.Order) error {
	logger.Info("ValidateAvailableBalance start...")
	pair, err := s.pairDao.GetByTokenAddress(o.BaseToken, o.QuoteToken)
	if err != nil {
		logger.Error(err)
		return err
	}

	totalRequiredAmount := o.TotalRequiredSellAmount(pair)

	var sellTokenBalance *big.Int

	// we implement retries in the case the provider connection fell asleep
	err = utils.Retry(3, func() error {
		sellTokenBalance, err = s.ethereumProvider.Balance(o.UserAddress, o.SellToken())
		return nil
	})

	if err != nil {
		logger.Error(err)
		return err
	}
	tokenInfo, err := s.tokenDao.GetByAddress(o.SellToken())
	if err != nil {
		logger.Error(err)
		return err
	}
	listPairs, err := s.pairDao.GetActivePairs()
	if err != nil {
		logger.Error(err)
		return err
	}
	sellExchangeTokenLockedBalance, err := s.orderDao.GetUserLockedBalance(o.UserAddress, o.SellToken(), listPairs)
	if err != nil {
		logger.Error(err)
		return err
	}
	sellLendingTokenLockedBalance, err := s.lendingDao.GetUserLockedBalance(o.UserAddress, o.SellToken(), tokenInfo.Decimals)
	if err != nil {
		logger.Error(err)
		return err
	}
	sellTokenLockedBalance := new(big.Int).Add(sellExchangeTokenLockedBalance, sellLendingTokenLockedBalance)
	availableSellTokenBalance := math.Sub(sellTokenBalance, sellTokenLockedBalance)

	//Sell Token Balance
	if sellTokenBalance.Cmp(totalRequiredAmount) == -1 {
		return fmt.Errorf("insufficient %v Balance", o.SellTokenSymbol())
	}

	if availableSellTokenBalance.Cmp(totalRequiredAmount) == -1 {
		return fmt.Errorf("insufficient %v available", o.SellTokenSymbol())
	}

	return nil
}

// ValidateAvailablLendingBalance validate avalable lending order
func (s *ValidatorService) ValidateAvailablLendingBalance(o *types.LendingOrder) error {
	totalRequiredAmount := o.Quantity

	var sellTokenBalance *big.Int
	var err error
	sellToken := o.CollateralToken
	if o.Side == types.LEND {
		sellToken = o.LendingToken
	}
	err = utils.Retry(3, func() error {
		sellTokenBalance, err = s.ethereumProvider.Balance(o.UserAddress, sellToken)
		return err
	})

	if err != nil {
		logger.Error(err, "addr:", sellToken.Hex())
		return err
	}
	tokenInfo, err := s.tokenDao.GetByAddress(sellToken)
	if err != nil {
		logger.Error(err)
		return err
	}
	listPairs, err := s.pairDao.GetActivePairs()
	if err != nil {
		logger.Error(err)
		return err
	}
	sellExchangeTokenLockedBalance, err := s.orderDao.GetUserLockedBalance(o.UserAddress, sellToken, listPairs)
	if err != nil {
		logger.Error(err)
		return err
	}
	sellLendingTokenLockedBalance, err := s.lendingDao.GetUserLockedBalance(o.UserAddress, sellToken, tokenInfo.Decimals)
	if err != nil {
		logger.Error(err)
		return err
	}
	sellTokenLockedBalance := new(big.Int).Add(sellExchangeTokenLockedBalance, sellLendingTokenLockedBalance)
	availableSellTokenBalance := math.Sub(sellTokenBalance, sellTokenLockedBalance)

	//Sell Token Balance
	if sellTokenBalance.Cmp(totalRequiredAmount) == -1 {
		return fmt.Errorf("insufficient %v Balance", sellToken.Hex())
	}

	if availableSellTokenBalance.Cmp(totalRequiredAmount) == -1 {
		return fmt.Errorf("insufficient %v available", sellToken.Hex())
	}

	return nil
}
