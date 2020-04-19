package oracle

import (
	"github.com/bandprotocol/bandchain/chain/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type ExecutionEnvironment struct {
	request            types.Request
	now                int64
	maxResultSize      int64
	maxCalldataSize    int64
	maxRawRequestCount int64
	rawRequests        []types.RawRequest
	reports            map[string]map[types.EID]types.RawReport
}

func NewExecutionEnvironment(ctx sdk.Context, k Keeper, req types.Request) *ExecutionEnvironment {
	return &ExecutionEnvironment{
		request:            req,
		now:                ctx.BlockTime().Unix(),
		maxResultSize:      int64(k.GetParam(ctx, KeyMaxResultSize)),
		maxCalldataSize:    int64(k.GetParam(ctx, KeyMaxCalldataSize)),
		maxRawRequestCount: int64(k.GetParam(ctx, KeyMaxDataSourceCountPerRequest)),
		rawRequests:        []types.RawRequest{},
		reports:            make(map[string]map[types.EID]types.RawReport),
	}
}

// GetRawRequests returns the list of raw requests made during Owasm prepare run.
func (env *ExecutionEnvironment) GetRawRequests() []types.RawRequest {
	return env.rawRequests
}

// SetReports loads the reports to the environment. Must be called prior to Owasm execute run.
func (env *ExecutionEnvironment) SetReports(reports []types.Report) {
	for _, report := range reports {
		valReports := make(map[types.EID]types.RawReport)
		for _, each := range report.RawReports {
			valReports[each.ExternalID] = each
		}
		env.reports[report.Validator.String()] = valReports
	}
}

// GetRequestedValidatorCount implements Owasm ExecutionEnvironment interface.
func (env *ExecutionEnvironment) GetRequestedValidatorCount() int64 {
	return int64(len(env.request.RequestedValidators))
}

// GetSufficientValidatorCount implements Owasm ExecutionEnvironment interface.
func (env *ExecutionEnvironment) GetSufficientValidatorCount() int64 {
	return env.request.SufficientValidatorCount
}

// GetReceivedValidatorCount implements Owasm ExecutionEnvironment interface.
func (env *ExecutionEnvironment) GetReceivedValidatorCount() int64 {
	return int64(len(env.reports))
}

// GetPrepareBlockTime implements Owasm ExecutionEnvironment interface.
func (env *ExecutionEnvironment) GetPrepareBlockTime() int64 {
	return env.request.RequestTime
}

// GetAggregateBlockTime implements Owasm ExecutionEnvironment interface.
func (env *ExecutionEnvironment) GetAggregateBlockTime() int64 {
	if len(env.reports) == 0 { // Size of reports must be zero during prepare.
		return 0
	}
	return env.now
}

// GetMaximumResultSize implements Owasm ExecutionEnvironment interface.
func (env *ExecutionEnvironment) GetMaximumResultSize() int64 {
	return env.maxResultSize
}

// GetMaximumCalldataOfDataSourceSize implements Owasm ExecutionEnvironment interface.
func (env *ExecutionEnvironment) GetMaximumCalldataOfDataSourceSize() int64 {
	return env.maxCalldataSize
}

// RequestedValidators implements Owasm ExecutionEnvironment interface.
func (env *ExecutionEnvironment) GetValidatorAddress(validatorIndex int64) ([]byte, error) {
	if validatorIndex < 0 || validatorIndex >= int64(len(env.request.RequestedValidators)) {
		return nil, types.ErrValidatorOutOfRange
	}
	return env.request.RequestedValidators[validatorIndex], nil
}

// RequestExternalData implements Owasm ExecutionEnvironment interface.
func (env *ExecutionEnvironment) RequestExternalData(did int64, eid int64, calldata []byte) error {
	if int64(len(calldata)) > env.maxCalldataSize {
		return types.ErrValidatorOutOfRange
	}
	if int64(len(env.rawRequests)) >= env.maxRawRequestCount {
		return types.ErrTooManyRawRequests
	}
	env.rawRequests = append(env.rawRequests, types.NewRawRequest(
		types.ExternalID(eid), types.DataSourceID(did), calldata,
	))
	return nil
}

// GetExternalData implements Owasm ExecutionEnvironment interface.
func (env *ExecutionEnvironment) GetExternalData(eid int64, valIdx int64) ([]byte, uint8, error) {
	if valIdx < 0 || valIdx >= int64(len(env.request.RequestedValidators)) {
		return nil, 0, types.ErrValidatorOutOfRange
	}
	valAddr := env.request.RequestedValidators[valIdx].String()
	valReports, ok := env.reports[valAddr]
	if !ok {
		return nil, 0, types.ErrItemNotFound
	}
	valReport, ok := valReports[types.EID(eid)]
	if !ok {
		return nil, 0, types.ErrItemNotFound
	}
	return valReport.Data, valReport.ExitCode, nil
}
