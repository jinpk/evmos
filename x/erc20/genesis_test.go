package erc20_test

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cometbft/cometbft/crypto/tmhash"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmversion "github.com/cometbft/cometbft/proto/tendermint/version"
	"github.com/cometbft/cometbft/version"

	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"

	utiltx "github.com/evmos/evmos/v16/testutil/tx"
	"github.com/evmos/evmos/v16/utils"
	feemarkettypes "github.com/evmos/evmos/v16/x/feemarket/types"

	"github.com/evmos/evmos/v16/app"
	"github.com/evmos/evmos/v16/x/erc20"
	"github.com/evmos/evmos/v16/x/erc20/types"
)

type GenesisTestSuite struct {
	suite.Suite
	ctx     sdk.Context
	app     *app.Evmos
	genesis types.GenesisState
}

const osmoERC20ContractAddr = "0x5dCA2483280D9727c80b5518faC4556617fb19ZZ"

var osmoDenomTrace = transfertypes.DenomTrace{
	BaseDenom: "uosmo",
	Path:      "transfer/channel-0",
}

func TestGenesisTestSuite(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}

func (suite *GenesisTestSuite) SetupTest() {
	// consensus key
	consAddress := sdk.ConsAddress(utiltx.GenerateAddress().Bytes())

	chainID := utils.TestnetChainID + "-1"
	suite.app = app.Setup(false, feemarkettypes.DefaultGenesisState(), chainID)
	suite.ctx = suite.app.BaseApp.NewContext(false, tmproto.Header{
		Height:          1,
		ChainID:         chainID,
		Time:            time.Now().UTC(),
		ProposerAddress: consAddress.Bytes(),

		Version: tmversion.Consensus{
			Block: version.BlockProtocol,
		},
		LastBlockId: tmproto.BlockID{
			Hash: tmhash.Sum([]byte("block_id")),
			PartSetHeader: tmproto.PartSetHeader{
				Total: 11,
				Hash:  tmhash.Sum([]byte("partset_header")),
			},
		},
		AppHash:            tmhash.Sum([]byte("app")),
		DataHash:           tmhash.Sum([]byte("data")),
		EvidenceHash:       tmhash.Sum([]byte("evidence")),
		ValidatorsHash:     tmhash.Sum([]byte("validators")),
		NextValidatorsHash: tmhash.Sum([]byte("next_validators")),
		ConsensusHash:      tmhash.Sum([]byte("consensus")),
		LastResultsHash:    tmhash.Sum([]byte("last_result")),
	})

	suite.genesis = *types.DefaultGenesisState()
}

func (suite *GenesisTestSuite) TestERC20InitGenesis() {
	testCases := []struct {
		name         string
		genesisState types.GenesisState
		malleate     func()
	}{
		{
			name:         "empty genesis",
			genesisState: types.GenesisState{},
			malleate:     nil,
		},
		{
			name:         "default genesis",
			genesisState: *types.DefaultGenesisState(),
			malleate:     nil,
		},
		{
			name: "custom genesis",
			genesisState: types.NewGenesisState(
				types.DefaultParams(),
				[]types.TokenPair{
					{
						Erc20Address:  osmoERC20ContractAddr,
						Denom:         osmoDenomTrace.IBCDenom(),
						Enabled:       true,
						ContractOwner: types.OWNER_MODULE,
					},
				},
			),
			malleate: func() {
				suite.app.TransferKeeper.SetDenomTrace(
					suite.ctx,
					transfertypes.DenomTrace{
						BaseDenom: "uosmo",
						Path:      "transfer/channel-0",
					},
				)
			},
		},
	}

	for _, tc := range testCases {
		if tc.malleate != nil {
			tc.malleate()
		}

		suite.Require().NotPanics(func() {
			erc20.InitGenesis(suite.ctx, suite.app.Erc20Keeper, suite.app.AccountKeeper, tc.genesisState)
		})

		params := suite.app.Erc20Keeper.GetParams(suite.ctx)

		tokenPairs := suite.app.Erc20Keeper.GetTokenPairs(suite.ctx)
		suite.Require().Equal(tc.genesisState.Params, params)
		if len(tokenPairs) > 0 {
			suite.Require().Equal(tc.genesisState.TokenPairs, tokenPairs)
			// check ERC20 contract was created successfully
			acc := suite.app.EvmKeeper.GetAccount(suite.ctx, common.HexToAddress(osmoERC20ContractAddr))
			suite.Require().True(acc.IsContract())
		} else {
			suite.Require().Len(tc.genesisState.TokenPairs, 0)
		}
	}
}

func (suite *GenesisTestSuite) TestErc20ExportGenesis() {
	testGenCases := []struct {
		name         string
		genesisState types.GenesisState
		malleate     func()
	}{
		{
			name:         "empty genesis",
			genesisState: types.GenesisState{},
			malleate:     nil,
		},
		{
			name:         "default genesis",
			genesisState: *types.DefaultGenesisState(),
			malleate:     nil,
		},
		{
			name: "custom genesis",
			genesisState: types.NewGenesisState(
				types.DefaultParams(),
				[]types.TokenPair{
					{
						Erc20Address:  osmoERC20ContractAddr,
						Denom:         osmoDenomTrace.IBCDenom(),
						Enabled:       true,
						ContractOwner: types.OWNER_MODULE,
					},
				},
			),
			malleate: func() {
				suite.app.TransferKeeper.SetDenomTrace(suite.ctx, osmoDenomTrace)
			},
		},
	}

	for _, tc := range testGenCases {
		if tc.malleate != nil {
			tc.malleate()
		}
		erc20.InitGenesis(suite.ctx, suite.app.Erc20Keeper, suite.app.AccountKeeper, tc.genesisState)
		suite.Require().NotPanics(func() {
			genesisExported := erc20.ExportGenesis(suite.ctx, suite.app.Erc20Keeper)
			params := suite.app.Erc20Keeper.GetParams(suite.ctx)
			suite.Require().Equal(genesisExported.Params, params)

			tokenPairs := suite.app.Erc20Keeper.GetTokenPairs(suite.ctx)
			if len(tokenPairs) > 0 {
				suite.Require().Equal(genesisExported.TokenPairs, tokenPairs)
			} else {
				suite.Require().Len(genesisExported.TokenPairs, 0)
			}
		})
		// }
	}
}
