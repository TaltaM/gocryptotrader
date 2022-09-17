package gateio

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	gateioTradeURL                      = "https://api.gateio.ws"
	gateioFuturesTestnetTrading         = "https://fx-api-testnet.gateio.ws"
	gateioFuturesLiveTradingAlternative = "https://fx-api.gateio.ws"
	gateioAPIVersion                    = "api/v4"

	// SubAccount Endpoints
	subAccounts = "sub_accounts"

	// Spot
	spotCurrencies                                 = "spot/currencies"
	spotCurrencyPairs                              = "spot/currency_pairs"
	spotTickers                                    = "spot/tickers"
	spotOrderbook                                  = "spot/order_book"
	spotMarketTrades                               = "spot/trades"
	spotCandlesticks                               = "spot/candlesticks"
	spotFeeRate                                    = "spot/fee"
	spotAccounts                                   = "spot/accounts"
	spotBatchOrders                                = "spot/batch_orders"
	spotOpenOrders                                 = "spot/open_orders"
	spotClosePositionWhenCrossCurrencyDisabledPath = "spot/cross_liquidate_orders"
	spotOrders                                     = "spot/orders"
	spotCancelBatchOrders                          = "spot/cancel_batch_orders"
	spotMyTrades                                   = "spot/my_trades"
	spotServerTime                                 = "spot/time"
	spotAllCountdown                               = "spot/countdown_cancel_all"
	spotPriceOrders                                = "spot/price_orders"

	// Wallets
	walletCurrencyChain                 = "wallet/currency_chains"
	walletDepositAddress                = "wallet/deposit_address"
	walletWithdrawals                   = "wallet/withdrawals"
	walletDeposits                      = "wallet/deposits"
	walletTransfer                      = "wallet/transfers"
	walletSubAccountTransfer            = "wallet/sub_account_transfers"
	walletWithdrawStatus                = "wallet/withdraw_status"
	walletSubAccountBalance             = "wallet/sub_account_balances"
	walletSubAccountMarginBalance       = "wallet/sub_account_margin_balances"
	walletSubAccountFuturesBalance      = "wallet/sub_account_futures_balances"
	walletSubAccountCrossMarginBalances = "wallet/sub_account_cross_margin_balances"
	walletSavedAddress                  = "wallet/saved_address"
	walletTradingFee                    = "wallet/fee"
	walletTotalBalance                  = "wallet/total_balance"

	// Margin
	marginCurrencyPairs     = "margin/currency_pairs"
	marginFundingBook       = "margin/funding_book"
	marginAccount           = "margin/accounts"
	marginAccountBook       = "margin/account_book"
	marginFundingAccounts   = "margin/funding_accounts"
	marginLoans             = "margin/loans"
	marginMergedLoans       = "margin/merged_loans"
	marginLoanRecords       = "margin/loan_records"
	marginAutoRepay         = "margin/auto_repay"
	marginTransfer          = "margin/transferable"
	marginBorrowable        = "margin/borrowable"
	crossMarginCurrencies   = "margin/cross/currencies"
	crossMarginAccounts     = "margin/cross/accounts"
	crossMarginAccountBook  = "margin/cross/account_book"
	crossMarginLoans        = "margin/cross/loans"
	crossMarginRepayments   = "margin/cross/repayments"
	crossMarginTransferable = "margin/cross/transferable"
	crossMarginBorrowable   = "margin/cross/borrowable"

	// Futures
	futuresSettleContracts            = "futures/%s/contracts"
	futuresOrderbook                  = "futures/%s/order_book"
	futuresTrades                     = "futures/%s/trades"
	futuresCandlesticks               = "futures/%s/candlesticks"
	futuresTicker                     = "futures/%s/tickers"
	futuresFundingRate                = "futures/%s/funding_rate"
	futuresInsuranceBalance           = "futures/%s/insurance"
	futuresContractStats              = "futures/%s/contract_stats"
	futuresIndexConstituent           = "futures/%s/index_constituents/%s"
	futuresLiquidationHistory         = "futures/%s/liq_orders"
	futuresAccounts                   = "futures/%s/accounts"
	futuresAccountBook                = "futures/%s/account_book"
	futuresPositions                  = "futures/%s/positions"
	futuresSinglePosition             = "futures/%s/positions/%s"        // {settle} and {contract} respectively
	futuresUpdatePositionMargin       = "futures/%s/positions/%s/margin" // {settle} and {contract} respectively
	futuresPositionsLeverage          = "futures/%s/positions/%s/leverage"
	futuresPositionRiskLimit          = "futures/%s/positions/%s/risk_limit"
	futuresDualMode                   = "futures/%s/dual_mode"
	futuresDualModePositions          = "futures/%s/dual_comp/positions/%s" // {settle} and {contract}
	futuresDualModePositionMargin     = "futures/%s/dual_comp/positions/%s/margin"
	futuresDualModePositionLeverage   = "futures/%s/dual_comp/positions/%s/leverage" // {settle} and {contract} and respectively
	futuresDualModePositionsRiskLimit = "futures/%s/dual_comp/positions/%s/risk_limit"
	futuresOrders                     = "futures/%s/orders"
	futuresBatchOrders                = "futures/%s/batch_orders"
	futuresPersonalTradingHistory     = "futures/%s/my_trades"
	futuresPositionClose              = "futures/%s/position_close"
	futuresLiquidations               = "futures/%s/liquidates"
	futuresCountdownCancelAll         = "futures/%s/countdown_cancel_all"
	futuresPriceOrders                = "futures/%s/price_orders"

	// Delivery
	deliveryContracts              = "delivery/%s/contracts"
	deliveryOrderbook              = "delivery/%s/order_book"
	deliveryTradeHistory           = "delivery/%s/trades"
	deliveryCandlesticks           = "delivery/%s/candlesticks"
	deliveryTicker                 = "delivery/%s/tickers"
	deliveryInsuranceBalance       = "delivery/%s/insurance"
	deliveryFuturesAccounts        = "delivery/%s/accounts"
	deliveryAccountBook            = "delivery/%s/account_book"
	deliveryPositions              = "delivery/%s/positions"
	deliverySinglePosition         = deliveryPositions + "/%s"
	deliveryUpdatePositionMargin   = "delivery/%s/positions/%s/margin"
	deliveryPositionsLeverage      = "delivery/%s/positions/%s/leverage"
	deliveryPositionRiskLimit      = "delivery/%s/positions/%s/risk_limit"
	deliveryOrders                 = "delivery/%s/orders"
	deliveryMyTrades               = "delivery/%s/my_trades"
	deliveryPersonalTradingHistory = "delivery/%s/my_trades"
	deliveryPositionClose          = "delivery/%s/position_close"
	deliveryLiquidations           = "delivery/%s/liquidates"
	deliverySettlements            = "delivery/%s/settlements"
	deliveryPriceOrders            = "delivery/%s/price_orders"

	// Options
	optionUnderlyings            = "options/underlyings"
	optionExpiration             = "options/expirations"
	optionContracts              = "options/contracts"
	optionSettlement             = "options/settlements"
	optionMySettlements          = "options/my_settlements"
	optionsOrderbook             = "options/order_book"
	optionsTickers               = "options/tickers"
	optionsUnderlyingTickers     = "options/underlying/tickers/%s"
	optionCandlesticks           = "options/candlesticks"
	optionUnderlyingCandlesticks = "options/underlying/candlesticks"
	optionsTrades                = "options/trades"
	optionAccounts               = "options/accounts"
	optionsAccountbook           = "options/account_book"
	optionsPosition              = "options/positions"
	optionsPositionClose         = "options/position_close"
	optionsOrders                = "options/orders"
	optionsMyTrades              = "options/my_trades"

	// Flash Swap
	flashSwapCurrencies    = "flash_swap/currencies"
	flashSwapOrders        = "flash_swap/orders"
	flashSwapOrdersPreview = "flash_swap/orders/preview"

	// Withdrawals
	withdrawal                     = "withdrawals"
	clientWithdrawalWithSpecificID = "withdrawals/%s"
)

const (
	UTC0TimeZone = "utc0"
	UTC8TimeZone = "utc8"
)

var (
	errInvalidCurrency                     = errors.New("invalid or empty currency")
	errInvalidAssetType                    = errors.New("invalid asset type")
	errInvalidOrEmptyCurrencyPair          = errors.New("empty or invalid currency pair")
	errMissingSettleCurrency               = errors.New("missing settle currency")
	errInvalidOrMissingContractParam       = errors.New("invalid or empty contract")
	errNoValidResponseFromServer           = errors.New("no valid response from server")
	errInvalidUnderlying                   = errors.New("missing underlying")
	errInvalidOrderSize                    = errors.New("invalid order size")
	errInvalidOrderID                      = errors.New("invalid order id")
	errInvalidWithdrawalDestinationAddress = errors.New("invalid withdrawal destination addresss")
	errInvalidAmount                       = errors.New("invalid amount")
	errInvalidOrEmptySubaccount            = errors.New("invalid or empty subaccount")
	errInvalidTransferDirection            = errors.New("invalid transfer direction")
	errInvalidOrderSide                    = errors.New("invalid order side")
	errDifferentAccount                    = errors.New("account type must be identical for all orders")
	errInvalidPrice                        = errors.New("invalid price")
	errNoValidParameterPassed              = errors.New("no valid parameter passed")
	errInvalidCountdown                    = errors.New("invalid countdown, Countdown time, in seconds At least 5 seconds, 0 means cancel the countdown")
	errInvalidOrderStatus                  = errors.New("invalid order status")
	errInvalidLoanSide                     = errors.New("invalid loan side, only 'lend' and 'borrow'")
	errInvalidLoanID                       = errors.New("missing loan ID")
	errInvalidRepayMode                    = errors.New("invalid repay mode specified, must be 'all' or 'partial'")
	errInvalidAutoRepaymentStatus          = errors.New("invalid auto repayment status response")
	errMissingPreviewID                    = errors.New("Missing required parameter: preview_id")
	errChangehasToBePositive               = errors.New("change has to be positive")
	errInvalidLeverageValue                = errors.New("invalid leverage value")
	errInvalidRiskLimit                    = errors.New("new position risk limit")
	errInvalidCountTotalValue              = errors.New("invalid \"count_total\" value, supported \"count_total\" values are 0 and 1")
	errInvalidTimeInForce                  = errors.New("invalid time in force value")
	errInvalidAutoSizeValue                = errors.New("invalid \"auto_size\" value, only \"close_long\" and \"close_short\" are supported")
	errTooManyOrderRequest                 = errors.New("too many order creation request")
	errInvalidTimeout                      = errors.New("invalid timeout, should be in seconds At least 5 seconds, 0 means cancel the countdown")
	errAssetTypeNotSupported               = errors.New("asset type not supported")
	errNoTickerData                        = errors.New("no ticker data available")
	errUnsupportedSettleValue              = errors.New("only 'usd','usdt', and 'btc' quote of currency pairs are supported")
)

// Gateio is the overarching type across this package
type Gateio struct {
	exchange.Base
}

// ***************************************** SubAccounts ********************************

// CreateNewSubAccount creates a new sub-account
func (g *Gateio) CreateNewSubAccount(ctx context.Context, arg SubAccountParams) (*SubAccount, error) {
	if arg.LoginName == "" {
		return nil, errors.New("login name can not be empty")
	}
	var response SubAccount
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, subAccounts, nil, &arg, &response)
}

// GetSubAccounts retrives list of sub-accounts for given account
func (g *Gateio) GetSubAccounts(ctx context.Context) ([]SubAccount, error) {
	var response []SubAccount
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, subAccounts, nil, nil, &response)
}

// GetSingleSubAccount retrives a single sub-account for given account
func (g *Gateio) GetSingleSubAccount(ctx context.Context, userID string) (*SubAccount, error) {
	if userID == "" {
		return nil, errors.New("user ID can not be empty")
	}
	path := fmt.Sprintf(subAccounts+"/%s", userID)
	var response SubAccount
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &response)
}

// *****************************************  Spot **************************************

// ListAllCurrencies to retrive detailed list of each currency.
func (g *Gateio) ListAllCurrencies(ctx context.Context) ([]CurrencyInfo, error) {
	var resp []CurrencyInfo
	return resp, g.SendHTTPRequest(ctx, exchange.RestSpot, spotCurrencies, &resp)
}

// GetCurrencyDetail details of a specific currency
func (g *Gateio) GetCurrencyDetail(ctx context.Context, ccy currency.Code) (*CurrencyInfo, error) {
	var resp CurrencyInfo
	if ccy.IsEmpty() {
		return nil, errInvalidCurrency
	}
	path := fmt.Sprintf("%s/%s", spotCurrencies, ccy.String())
	return &resp, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// ListAllCurrencyPairs retrive all currency pairs supported by the exchange.
func (g *Gateio) ListAllCurrencyPairs(ctx context.Context) ([]CurrencyPairDetail, error) {
	var resp []CurrencyPairDetail
	return resp, g.SendHTTPRequest(ctx, exchange.RestSpot, spotCurrencyPairs, &resp)
}

// GetCurrencyPairDetal to get details of a specifc order
func (g *Gateio) GetCurrencyPairDetail(ctx context.Context, currencyPair currency.Pair) (*CurrencyPairDetail, error) {
	var resp CurrencyPairDetail
	if currencyPair.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	pair := currencyPair.Base.String() + currency.UnderscoreDelimiter + currencyPair.Quote.String()
	path := fmt.Sprintf("%s/%s", spotCurrencyPairs, pair)
	return &resp, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetTickers retrieve ticker information
// Return only related data if currency_pair is specified; otherwise return all of them
func (g *Gateio) GetTickers(ctx context.Context, currencyPair currency.Pair, timezone string) ([]Ticker, error) {
	var tickers []Ticker
	var pair string
	params := url.Values{}
	if !currencyPair.IsEmpty() {
		pair = currencyPair.Base.String() + currency.UnderscoreDelimiter + currencyPair.Quote.String()
		params.Set("currency_pair", pair)
	}
	if timezone == UTC8TimeZone || timezone == UTC0TimeZone {
		params.Set("timezone", timezone)
	} else {
		params.Set("timezone", "all")
	}
	return tickers, g.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(spotTickers, params), &tickers)
}

// GetTicker retrives a single ticker information for a currency pair.
func (g *Gateio) GetTicker(ctx context.Context, currencyPair currency.Pair, timezone string) (*Ticker, error) {
	if currencyPair.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	tickers, er := g.GetTickers(ctx, currencyPair, timezone)
	if er != nil {
		return nil, er
	}
	if len(tickers) > 0 {
		return &tickers[0], er
	}
	return nil, fmt.Errorf("no ticker data found for currency pair %v", currencyPair)
}

// GetIntervalString returns a string representation of the interval according to the Gateio exchange representation.
func (g *Gateio) GetIntervalString(interval kline.Interval) string {
	switch interval {
	case kline.TenSecond:
		return "10s"
	case kline.ThirtySecond:
		return "30s"
	case kline.OneMin:
		return "1m"
	case kline.FiveMin:
		return "5m"
	case kline.FifteenMin:
		return "15m"
	case kline.ThirtyMin:
		return "30m"
	case kline.OneHour:
		return "1h"
	case kline.TwoHour:
		return "2h"
	case kline.FourHour:
		return "4h"
	case kline.EightHour:
		return "8h"
	case kline.TwelveHour:
		return "12h"
	case kline.OneDay:
		return "1d"
	case kline.OneWeek:
		return "1w"
	case kline.ThirtyDay:
		return "30d"
	default:
		return ""
	}
}

// GetOrderbook returns the orderbook data for a suppled currency pair
func (g *Gateio) GetOrderbook(ctx context.Context, currencyPair currency.Pair, interval string, limit uint, withOrderbookID bool) (*Orderbook, error) {
	var response OrderbookData
	if currencyPair.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	params := url.Values{}
	fPair, er := g.GetPairFormat(asset.Spot, true)
	if er == nil {
		fPair = currency.PairFormat{
			Delimiter: currency.UnderscoreDelimiter,
			Uppercase: true,
		}
	}
	pairString := strings.ToUpper(fPair.Format(currencyPair))
	println(pairString)
	params.Set("currency_pair", pairString)
	if interval == OrderbookIntervalZero || interval == OrderbookIntervalZeroPt1 || interval == OrderbookIntervalZeroPtZero1 {
		params.Set("interval", interval)
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	params.Set("with_id", strconv.FormatBool(withOrderbookID))
	er = g.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(spotOrderbook, params), &response)
	if er != nil {
		return nil, er
	}
	return response.MakeOrderbook()
}

// GetMarketTrades retrieve market trades
func (g *Gateio) GetMarketTrades(ctx context.Context, currencyPair currency.Pair, limit uint, lastID string, reverse bool, from, to time.Time, page int) ([]Trade, error) {
	var response []Trade
	params := url.Values{}
	if currencyPair.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	fPair, er := g.GetPairFormat(asset.Spot, true)
	if er != nil {
		fPair = currency.PairFormat{
			Delimiter: currency.UnderscoreDelimiter,
			Uppercase: true,
		}
	}
	pairString := strings.ToUpper(fPair.Format(currencyPair))
	params.Set("currency_pair", pairString)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(uint64(limit), 10))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	if reverse {
		params.Set("reverse", strconv.FormatBool(reverse))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if page != 0 {
		params.Set("page", strconv.Itoa(page))
	}
	path := common.EncodeURLValues(spotMarketTrades, params)
	return response, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &response)
}

// GetCandlesticks retrives market candlesticks.
func (g *Gateio) GetCandlesticks(ctx context.Context, currencyPair currency.Pair, limit uint, from, to time.Time, interval kline.Interval) ([]Candlestick, error) {
	var candles [][7]string
	params := url.Values{}
	if currencyPair.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	fPair, er := g.GetPairFormat(asset.Spot, true)
	if er != nil {
		fPair = currency.PairFormat{
			Delimiter: currency.UnderscoreDelimiter,
			Uppercase: true,
		}
	}
	pairString := strings.ToUpper(fPair.Format(currencyPair))
	params.Set("currency_pair", pairString)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(uint64(limit), 10))
	}
	if intervalString := g.GetIntervalString(interval); intervalString != "" {
		params.Set("interval", intervalString)
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	path := common.EncodeURLValues(spotCandlesticks, params)
	er = g.SendHTTPRequest(ctx, exchange.RestSpot, path, &candles)
	if er != nil {
		return nil, er
	}
	if len(candles) == 0 {
		return nil, errors.New("no candlesticks available")
	}
	candlesticks := make([]Candlestick, len(candles))
	for x := range candles {
		timestamp, er := strconv.ParseInt(candles[x][0], 10, 64)
		if er != nil {
			return nil, er
		}
		quoteTradingVolume, er := strconv.ParseFloat(candles[x][1], 64)
		if er != nil {
			return nil, er
		}
		closePrice, er := strconv.ParseFloat(candles[x][2], 64)
		if er != nil {
			return nil, er
		}
		highestPrice, er := strconv.ParseFloat(candles[x][3], 64)
		if er != nil {
			return nil, er
		}
		lowestPrice, er := strconv.ParseFloat(candles[x][4], 64)
		if er != nil {
			return nil, er
		}
		openPrice, er := strconv.ParseFloat(candles[x][5], 64)
		if er != nil {
			return nil, er
		}
		baseCurrencyAmount, er := strconv.ParseFloat(candles[x][6], 64)
		if er != nil {
			return nil, er
		}
		candlesticks[x] = Candlestick{
			Timestamp:      time.Unix(timestamp, 0),
			QuoteCcyVolume: quoteTradingVolume,
			ClosePrice:     closePrice,
			HighestPrice:   highestPrice,
			LowestPrice:    lowestPrice,
			OpenPrice:      openPrice,
			BaseCcyAmount:  baseCurrencyAmount,
		}
	}
	return candlesticks, nil
}

// GetTradingFeeRatio retrives user trading fee rates
func (g *Gateio) GetTradingFeeRatio(ctx context.Context, currencyPair currency.Pair) (*SpotTradingFeeRate, error) {
	var response SpotTradingFeeRate
	params := url.Values{}
	if !(currencyPair.Quote.IsEmpty() || currencyPair.Base.IsEmpty()) {
		// specify a currency pair to retrieve precise fee rate
		currencyPair.Delimiter = currency.UnderscoreDelimiter
		params.Set("currency_pair", currencyPair.String())
	}
	return &response, g.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpot, http.MethodGet, spotFeeRate, params, nil, &response)
}

// GetSpotAccounts retrives spot account.
func (g *Gateio) GetSpotAccounts(ctx context.Context, ccy currency.Code) ([]SpotAccount, error) {
	var response []SpotAccount
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	return response, g.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpot, http.MethodGet, spotAccounts, params, nil, &response)
}

// CreateBatchOrders Create a batch of orders
// Batch orders requirements:
// custom order field text is required
// At most 4 currency pairs, maximum 10 orders each, are allowed in one request
// No mixture of spot orders and margin orders, i.e. account must be identical for all orders
func (g *Gateio) CreateBatchOrders(ctx context.Context, args []CreateOrderRequestData) ([]SpotOrder, error) {
	var response []SpotOrder
	input := args
	for x := range args {
		if x == 10 {
			break
		} else if (x != 0) && args[x-1].Account != args[x].Account {
			return nil, errDifferentAccount
		}
		if args[x].CurrencyPair.Base.IsEmpty() || args[x].CurrencyPair.Quote.IsEmpty() {
			return nil, errInvalidOrEmptyCurrencyPair
		}
		args[x].CurrencyPair.Delimiter = currency.UnderscoreDelimiter
		if !(args[x].TimeInForce == "gtc" || args[x].TimeInForce == "ioc" || args[x].TimeInForce == "poc" || args[x].TimeInForce == "foc") {
			args[x].TimeInForce = "gtc"
		}
		if args[x].Type != "limit" {
			return nil, fmt.Errorf("only order type %s is allowed", "limit")
		}
		args[x].Side = strings.ToLower(args[x].Side)
		if !(args[x].Side == "buy" || args[x].Side == "sell") {
			return nil, errInvalidOrderSide
		}
		if !(args[x].Account == asset.Spot || args[x].Account == asset.CrossMargin || args[x].Account == asset.Margin) {
			return nil, fmt.Errorf("only %v, %v, and %v area allowed", asset.Spot, asset.CrossMargin, asset.Margin)
		}
		if args[x].Text == "" {
			args[x].Text = fmt.Sprintf("t-%s", common.GenerateRandomString(10, common.NumberCharacters))
		}
		if args[x].Amount <= 0 {
			return nil, errInvalidAmount
		}
		if args[x].Price <= 0 {
			return nil, errInvalidPrice
		}
	}
	if len(args) > 10 {
		input = input[:10]
	}
	val, _ := json.Marshal(input)
	println(string(val))
	if er := g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, spotBatchOrders, nil, &input, &response); er != nil {
		return nil, er
	}
	if len(args) > 10 {
		if newResponse, er := g.CreateBatchOrders(ctx, args[10:]); er == nil {
			response = append(response, newResponse...)
		}
	}
	return response, nil
}

// GetSpotOpenOrders retrives all open orders
// List open orders in all currency pairs.
// Note that pagination parameters affect record number in each currency pair's open order list. No pagination is applied to the number of currency pairs returned. All currency pairs with open orders will be returned.
// Spot and margin orders are returned by default. To list cross margin orders, account must be set to cross_margin
func (g *Gateio) GetSpotOpenOrders(ctx context.Context, page, limit int, account asset.Item) ([]SpotOrdersDetail, error) {
	params := url.Values{}
	if page > 0 {
		params.Set("page", strconv.Itoa(page))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if account == asset.Spot || account == asset.Margin || account == asset.CrossMargin {
		params.Set("account", account.String())
	}
	var response []SpotOrdersDetail
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, spotOpenOrders, params, nil, &response)
}

// SpotClosePositionWhenCrossCurrencyDisabled set close position when cross-currency is disabled
func (g *Gateio) SpotClosePositionWhenCrossCurrencyDisabled(ctx context.Context, arg ClosePositionRequestParam) (*SpotOrder, error) {
	var response SpotOrder
	if arg.CurrencyPair.Base.IsEmpty() || arg.CurrencyPair.Quote.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	arg.CurrencyPair.Delimiter = currency.UnderscoreDelimiter
	if arg.Amount <= 0 {
		return nil, errInvalidAmount
	}
	if arg.Price <= 0 {
		return nil, errInvalidPrice
	}
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot,
		http.MethodPost, spotClosePositionWhenCrossCurrencyDisabledPath, nil, &arg, &response)
}

// CreateSpotOrder creates a spot order
// you can place orders with spot, margin or cross margin account through setting the accountfield.
// It defaults to spot, which means spot account is used to place orders.
func (g *Gateio) CreateSpotOrder(ctx context.Context, arg CreateOrderRequestData) (*SpotOrder, error) {
	var response SpotOrder
	if arg.CurrencyPair.Base.IsEmpty() || arg.CurrencyPair.Quote.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	arg.CurrencyPair.Delimiter = currency.UnderscoreDelimiter
	if !(arg.TimeInForce == "gtc" || arg.TimeInForce == "ioc" || arg.TimeInForce == "poc" || arg.TimeInForce == "foc") {
		arg.TimeInForce = "gtc"
	}
	if arg.Type != "limit" {
		return nil, fmt.Errorf("only order type %s is allowed", "limit")
	}
	arg.Side = strings.ToLower(arg.Side)
	if !(arg.Side == "buy" || arg.Side == "sell") {
		return nil, errInvalidOrderSide
	}
	if !(arg.Account == asset.Spot || arg.Account == asset.CrossMargin || arg.Account == asset.Margin) {
		return nil, fmt.Errorf("only %v, %v, and %v area allowed", asset.Spot, asset.CrossMargin, asset.Margin)
	}
	if arg.Text == "" {
		arg.Text = fmt.Sprintf("t-%s", common.GenerateRandomString(10, common.NumberCharacters))
	}
	if arg.Amount <= 0 {
		return nil, errInvalidAmount
	}
	if arg.Price <= 0 {
		return nil, errInvalidPrice
	}
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, spotOrders, nil, &arg, &response)
}

// GetSpotOrders retrives spot orders.
func (g *Gateio) GetSpotOrders(ctx context.Context, currencyPair currency.Pair, status string, page, limit int) ([]SpotOrder, error) {
	var response []SpotOrder
	if currencyPair.Base.IsEmpty() || currencyPair.Quote.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	currencyPair.Delimiter = currency.UnderscoreDelimiter
	params := url.Values{}
	params.Set("currency_pair", currencyPair.String())
	if status == "open" || status == "finished" {
		params.Set("status", status)
	}
	if page > 0 {
		params.Set("page", strconv.Itoa(page))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, spotOrders, params, nil, &response)
}

// CancelAllOpenOrdersSpecifiedCurrencyPair cancel all open orders in specified currency pair
func (g *Gateio) CancelAllOpenOrdersSpecifiedCurrencyPair(ctx context.Context, currencyPair currency.Pair, side order.Side, account asset.Item) ([]SpotOrder, error) {
	var response []SpotOrder
	if currencyPair.Base.IsEmpty() || currencyPair.Quote.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	currencyPair.Delimiter = currency.UnderscoreDelimiter
	params := url.Values{}
	params.Set("currency_pair", currencyPair.String())
	if side == order.Buy || side == order.Sell {
		params.Set("side", strings.ToLower(side.Title()))
	}
	if account == asset.Spot || account == asset.Margin || account == asset.CrossMargin {
		params.Set("account", account.String())
	}
	return response, g.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpot, http.MethodDelete, spotOrders, params, nil, &response)
}

// CancelBatchOrdersWithIDList cancels batch orders specifiying the order ID and currency pair information
// Multiple currency pairs can be specified, but maximum 20 orders are allowed per request
func (g *Gateio) CancelBatchOrdersWithIDList(ctx context.Context, args []CancelOrderByIDParam) ([]CancelOrderByIDResponse, error) {
	var response []CancelOrderByIDResponse
	var inputs []CancelOrderByIDParam
	for x := 0; x < len(args); x++ {
		if (args[x].CurrencyPair.Base.IsEmpty() || args[x].CurrencyPair.Quote.IsEmpty()) || args[x].ID == "" {
			if x == 0 {
				args = args[1:]
			} else if len(args) >= (x + 1) {
				args = append(args[:x], args[x+1:]...)
			} else {
				args = args[:x]
			}
			x -= 1
		} else {
			args[x].CurrencyPair.Delimiter = currency.UnderscoreDelimiter
		}
	}
	if len(args) == 0 {
		return nil, errNoValidParameterPassed
	} else if len(args) > 20 {
		inputs = args[:20]
	}
	er := g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, spotCancelBatchOrders, nil, &inputs, &response)
	if er != nil {
		return nil, er
	}
	if len(args) > 20 {
		newResponse, er := g.CancelBatchOrdersWithIDList(ctx, args)
		if er == nil {
			response = append(response, newResponse...)
		}
	}
	return response, nil
}

// GetSpotOrder retrives a single spot order using the order id and currency pair information.
func (g *Gateio) GetSpotOrder(ctx context.Context, orderID string, currencyPair currency.Pair, account asset.Item) (*SpotOrder, error) {
	var response SpotOrder
	params := url.Values{}
	if orderID == "" {
		return nil, errInvalidOrderID
	}
	if currencyPair.Base.IsEmpty() || currencyPair.Quote.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	currencyPair.Delimiter = currency.UnderscoreDelimiter
	params.Set("currency_pair", currencyPair.String())
	if account == asset.Margin || account == asset.Spot || account == asset.CrossMargin {
		params.Set("account", account.String())
	}
	path := fmt.Sprintf("%s/%s", spotOrders, orderID)
	return &response, g.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpot, http.MethodGet, path, params, nil, &response)
}

// CancelSingleSpotOrder cancels a single order
// Spot and margin orders are cancelled by default.
// If trying to cancel cross margin orders or portfolio margin account are used, account must be set to cross_margin
func (g *Gateio) CancelSingleSpotOrder(ctx context.Context, orderID string, currencyPair currency.Pair, account asset.Item) (*SpotOrder, error) {
	var response SpotOrder
	params := url.Values{}
	if orderID == "" {
		return nil, errInvalidOrderID
	}
	if currencyPair.Base.IsEmpty() || currencyPair.Quote.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	currencyPair.Delimiter = currency.UnderscoreDelimiter
	params.Set("currency_pair", currencyPair.String())
	if account == asset.Margin || account == asset.Spot || account == asset.CrossMargin {
		params.Set("account", account.String())
	}
	path := fmt.Sprintf("%s/%s", spotOrders, orderID)
	return &response, g.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpot, http.MethodDelete, path, params, nil, &response)
}

// GetPersonalTradingHistory retrives personal trading history
func (g *Gateio) GetPersonalTradingHistory(ctx context.Context, currencyPair currency.Pair,
	orderID string, limit, page int, account asset.Item, from, to time.Time) ([]SpotPersonalTradeHistory, error) {
	var response []SpotPersonalTradeHistory
	params := url.Values{}
	if currencyPair.Base.IsEmpty() || currencyPair.Quote.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	currencyPair.Delimiter = currency.UnderscoreDelimiter
	params.Set("currency_pair", currencyPair.String())
	if orderID != "" {
		params.Set("order_id", orderID)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if page > 0 {
		params.Set("page", strconv.Itoa(page))
	}
	if account == asset.Spot || account == asset.Margin || account == asset.CrossMargin {
		params.Set("account", account.String())
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() && to.After(from) {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, spotMyTrades, params, nil, &response)
}

// GetCurrencyServerTime retrives current server time
func (g *Gateio) GetServerTime(ctx context.Context) (time.Time, error) {
	type response struct {
		ServerTime int64 `json:"server_time"`
	}
	var resp response
	er := g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, spotServerTime, nil, nil, &resp)
	if er != nil {
		return time.Time{}, er
	}
	return time.Unix(resp.ServerTime, 0), nil
}

// CountdownCancelorder Countdown cancel orders
// When the timeout set by the user is reached, if there is no cancel or set a new countdown, the related pending orders will be automatically cancelled.
// This endpoint can be called repeatedly to set a new countdown or cancel the countdown.
func (g *Gateio) CountdownCancelorder(ctx context.Context, arg CountdownCancelOrderParam) (*TriggerTimeResponse, error) {
	var response TriggerTimeResponse
	if arg.Timeout < 0 || (arg.Timeout > 0 && arg.Timeout < 5) {
		return nil, errInvalidCountdown
	}
	if !arg.CurrencyPair.IsEmpty() {
		arg.CurrencyPair.Delimiter = currency.UnderscoreDelimiter
	}
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, spotAllCountdown, nil, &arg, &response)
}

// CreatePriceTriggeredOrder create a price-triggered order
func (g *Gateio) CreatePriceTriggeredOrder(ctx context.Context, arg PriceTriggeredOrderParam) (*OrderID, error) {
	var response OrderID
	if arg.Trigger.Price < 0 {
		return nil, fmt.Errorf("%v %s", errInvalidPrice, "invalid trigger price")
	}
	if !(arg.Trigger.Rule == "<=" || arg.Trigger.Rule == ">=") {
		return nil, errors.New("invalid price trigger condition or rule")
	}
	if arg.Trigger.Expiration <= 0 {
		return nil, errors.New("invalid expiration(seconds to wait for the condition to be triggered before cancelling the order)")
	}
	arg.Put.Side = strings.ToLower(arg.Put.Side)
	arg.Put.Type = strings.ToLower(arg.Put.Type)
	if arg.Put.Type != "limit" {
		return nil, errors.New("invalid order type, only order type 'limit' is allowed")
	}
	if !(arg.Put.Side == "buy" || arg.Put.Side == "sell") {
		return nil, errInvalidOrderSide
	}
	if arg.Put.Price < 0 {
		return nil, fmt.Errorf("%v, %s", errInvalidPrice, "put price has to be greater than 0")
	}
	if arg.Put.Amount <= 0 {
		return nil, errInvalidAmount
	}
	arg.Put.Account = strings.ToLower(arg.Put.Account)
	if !(arg.Put.Account == "margin" || arg.Put.Account == "cross_margin") || arg.Put.Account == "spot" {
		arg.Put.Account = "normal"
	}
	if !(arg.Put.TimeInForce == "gtc" || arg.Put.TimeInForce == "ioc") {
		arg.Put.TimeInForce = ""
	}
	if arg.Market.IsEmpty() {
		return nil, fmt.Errorf("%v, %s", errInvalidOrEmptyCurrencyPair, "field market is required")
	}
	arg.Market.Delimiter = currency.UnderscoreDelimiter
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, spotPriceOrders, nil, &arg, &response)
}

// GetPriceTriggeredOrderList retrives price orders created with an order detail and trigger price information.
func (g *Gateio) GetPriceTriggeredOrderList(ctx context.Context, status string, market currency.Pair, account asset.Item, offset, limit int) ([]SpotPriceTriggeredOrder, error) {
	var response []SpotPriceTriggeredOrder
	params := url.Values{}
	if !(status == "open" || status == "finished") {
		return nil, errInvalidOrderStatus
	}
	params.Set("status", status)
	if !(market.Base.IsEmpty() || market.Quote.IsEmpty()) {
		market.Delimiter = currency.UnderscoreDelimiter
		params.Set("market", market.String())
	}
	if account == asset.CrossMargin {
		params.Set("account", account.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, spotPriceOrders, params, nil, &response)
}

// CancelAllOpenOrders deletes price triggered orders.
func (g *Gateio) CancelAllOpenOrders(ctx context.Context, currencyPair currency.Pair, account asset.Item) ([]SpotPriceTriggeredOrder, error) {
	params := url.Values{}
	if !(currencyPair.Base.IsEmpty() || currencyPair.Quote.IsEmpty()) {
		params.Set("currency_pair", currencyPair.String())
	}
	if !(account == asset.Margin || account == asset.CrossMargin) || account == asset.Spot {
		params.Set("account", "normal")
	} else {
		params.Set("account", account.String())
	}
	var response []SpotPriceTriggeredOrder
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, spotPriceOrders, params, nil, &response)
}

// GetSinglePriceTriggeredOrder get a single order
func (g *Gateio) GetSinglePriceTriggeredOrder(ctx context.Context, orderID string) (*SpotPriceTriggeredOrder, error) {
	var response SpotPriceTriggeredOrder
	if orderID == "" {
		return nil, errInvalidOrderID
	}
	path := fmt.Sprintf("%s/%s", spotPriceOrders, orderID)
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &response)
}

// CancelPriceTriggeredOrder cancel a price-triggered order
func (g *Gateio) CancelPriceTriggeredOrder(ctx context.Context, orderID string) (*SpotPriceTriggeredOrder, error) {
	var response SpotPriceTriggeredOrder
	if orderID == "" {
		return nil, errInvalidOrderID
	}
	path := fmt.Sprintf("%s/%s", spotPriceOrders, orderID)
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &response)
}

// GenerateSignature returns hash for authenticated requests
func (g *Gateio) GenerateSignature(secret, method, path, query string, body interface{}, dtime time.Time) (string, error) {
	h := sha512.New()
	if body != nil {
		val, er := json.Marshal(body)
		if er != nil {
			return "", er
		}
		h.Write(val)
	}
	h.Write(nil)
	hashedPayload := hex.EncodeToString(h.Sum(nil))
	t := strconv.FormatInt(dtime.Unix(), 10)
	rawQuery, err := url.QueryUnescape(query)
	if err != nil {
		return "", err
	}
	msg := fmt.Sprintf("%s\n%s\n%s\n%s\n%s", method, path, rawQuery, hashedPayload, t)
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// SendAuthenticatedHTTPRequest sends authenticated requests to the Gateio API
// To use this you must setup an APIKey and APISecret from the exchange
func (g *Gateio) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, method, endpoint string, param url.Values, data, result interface{}) error {
	creds, err := g.GetCredentials(ctx)
	if err != nil {
		return err
	}
	ePoint, err := g.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	headers := make(map[string]string)
	urlPath := fmt.Sprintf("/%s/%s", gateioAPIVersion, endpoint)
	timestamp := time.Now()
	if err != nil {
		return err
	}
	var paramValue string
	if param != nil {
		paramValue = param.Encode()
	}
	hmac, err := g.GenerateSignature(creds.Secret, method, urlPath, paramValue, data, timestamp)
	if err != nil {
		return err
	}
	headers["Content-Type"] = "application/json"
	headers["KEY"] = creds.Key
	headers["TIMESTAMP"] = strconv.FormatInt(timestamp.Unix(), 10)
	headers["Accept"] = "application/json"
	headers["SIGN"] = hmac
	var intermidiary json.RawMessage
	urlPath = fmt.Sprintf("%s%s", ePoint, urlPath)
	if param != nil {
		urlPath = common.EncodeURLValues(urlPath, param)
	}
	item := &request.Item{
		Method:        method,
		Path:          urlPath,
		Headers:       headers,
		Result:        &intermidiary,
		AuthRequest:   true,
		Verbose:       g.Verbose,
		HTTPDebugging: g.HTTPDebugging,
		HTTPRecording: g.HTTPRecording,
	}
	err = g.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		var body io.Reader
		if data != nil {
			payload, err := json.Marshal(data)
			if err != nil {
				return nil, err
			}
			body = bytes.NewBuffer(payload)
		}
		item.Body = body
		return item, nil
	})
	if err != nil {
		return err
	}
	errCap := struct {
		Label   string `json:"label"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}{}

	if err := json.Unmarshal(intermidiary, &errCap); err == nil && errCap.Code != "" {
		return fmt.Errorf("%s auth request error, code: %s message: %s",
			g.Name,
			errCap.Label,
			errCap.Message)
	}
	if result == nil {
		return nil
	}
	return json.Unmarshal(intermidiary, result)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (g *Gateio) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, result interface{}) error {
	endpoint, err := g.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	path = fmt.Sprintf("/%s/%s", gateioAPIVersion, path)
	item := &request.Item{
		Method:        http.MethodGet,
		Path:          endpoint + path,
		Result:        result,
		Verbose:       g.Verbose,
		HTTPDebugging: g.HTTPDebugging,
		HTTPRecording: g.HTTPRecording,
	}
	return g.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return item, nil
	})
}

// *********************************** Withdrawals ******************************

// WithdrawCurrency to withdraw a currency.
func (g *Gateio) WithdrawCurrency(ctx context.Context, arg WithdrawalRequestParam) (*WithdrawalResponse, error) {
	var response WithdrawalResponse
	if arg.Amount <= 0 {
		return nil, errInvalidAmount
	}
	if arg.Currency.String() == "" {
		return nil, errInvalidCurrency
	}
	if arg.Address == "" {
		return nil, errInvalidWithdrawalDestinationAddress
	}
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, withdrawal, nil, &arg, &response)
}

func (g *Gateio) CancelWithdrawalWithSpecifiedID(ctx context.Context, withdrawalID string) (*WithdrawalResponse, error) {
	var response WithdrawalResponse
	path := fmt.Sprint(withdrawal, withdrawalID)
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, nil, nil, &response)
}

// *********************************** Wallet ***********************************

// ListCurrencyChain retrives a list of currency chain name
func (g *Gateio) ListCurrencyChain(ctx context.Context, ccy currency.Code) ([]CurrencyChain, error) {
	params := url.Values{}
	if ccy.IsEmpty() {
		return nil, errInvalidCurrency
	}
	params.Set("currency", ccy.String())
	var resp []CurrencyChain
	path := common.EncodeURLValues(walletCurrencyChain, params)
	return resp, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GenerateCurrencyDepositAddress generate currency deposit address
func (g *Gateio) GenerateCurrencyDepositAddress(ctx context.Context, ccy currency.Code) (*CurrencyDepositAddressInfo, error) {
	var response CurrencyDepositAddressInfo
	if ccy.IsEmpty() {
		return nil, errInvalidCurrency
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot,
		http.MethodGet, walletDepositAddress, params, nil, &response)
}

// GetWithdrawalRecords retrieves withdrawal records. Record time range cannot exceed 30 days
func (g *Gateio) GetWithdrawalRecords(ctx context.Context, ccy currency.Code, from, to time.Time, offset, limit int) ([]WithdrawalResponse, error) {
	var withdrawals []WithdrawalResponse
	if ccy.IsEmpty() {
		return nil, errInvalidCurrency
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	if offset > 0 {
		params.Set("offset", fmt.Sprintf("%v", offset))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() && to.Before(from.Add(time.Hour*720)) {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	return withdrawals, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot,
		http.MethodGet, walletWithdrawals, params, nil, &withdrawals)
}

// GetDepositRecords retrives deposit records. Record time range cannot exceed 30 days
func (g *Gateio) GetDepositRecords(ctx context.Context, ccy currency.Code, from, to time.Time, offset, limit int) ([]DepositRecord, error) {
	var depositHistories []DepositRecord
	if ccy.IsEmpty() {
		return nil, errInvalidCurrency
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	if offset > 0 {
		params.Set("offset", fmt.Sprintf("%v", offset))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() && to.Before(from.Add(time.Hour*720)) {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	return depositHistories, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot,
		http.MethodGet, walletDeposits, params, nil, &depositHistories)
}

// TransferCurrency Transfer between different accounts. Currently support transfers between the following:
// spot - margin, spot - futures(perpetual), spot - delivery
// spot - cross margin, spot - options
func (g *Gateio) TransferCurrency(ctx context.Context, arg TransferCurrencyParam) (*TransactionIDResponse, error) {
	var response TransactionIDResponse
	if arg.Currency.String() == "" {
		return nil, errInvalidCurrency
	}
	if !arg.CurrencyPair.IsEmpty() {
		arg.CurrencyPair.Delimiter = currency.UnderscoreDelimiter
	}
	if arg.From != asset.Spot {
		return nil, fmt.Errorf("%v, only %s accounts can be used to transfer from", errInvalidAssetType, asset.Spot)
	}
	if !(arg.To == asset.Spot ||
		arg.To == asset.Margin ||
		arg.To == asset.Futures ||
		arg.To == asset.DeliveryFutures ||
		arg.To == asset.CrossMargin ||
		arg.To == asset.Options) {
		return nil, fmt.Errorf("%v, only %v,%v,%v,%v,%v,and %v", errInvalidAssetType, asset.Spot, asset.Margin, asset.Futures, asset.DeliveryFutures, asset.CrossMargin, asset.Options)
	}
	if arg.Amount < 0 {
		return nil, errInvalidAmount
	}
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, walletTransfer, nil, &arg, &response)
}

// SubAccountTransfer to transfer between main and sub accounts
// Support transferring with sub user's spot or futures account. Note that only main user's spot account is used no matter which sub user's account is operated.
func (g *Gateio) SubAccountTransfer(ctx context.Context, arg SubAccountTransferParam) error {
	if arg.Currency.IsEmpty() {
		return errInvalidCurrency
	}
	if arg.SubAccount == "" {
		return errInvalidOrEmptySubaccount
	}
	arg.Direction = strings.ToLower(arg.Direction)
	if !(arg.Direction == "to" || arg.Direction == "from") {
		return errInvalidTransferDirection
	}
	if arg.Amount <= 0 {
		return errInvalidAmount
	}
	if arg.SubAccountType != asset.Empty && !(arg.SubAccountType == asset.Spot || arg.SubAccountType == asset.Futures || arg.SubAccountType == asset.CrossMargin) {
		return fmt.Errorf("%v; only %v,%v, and %v are allowed", errInvalidAssetType, asset.Spot, asset.Futures, asset.CrossMargin)
	}
	return g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, walletSubAccountTransfer, nil, &arg, nil)
}

// GetSubAccountTransferHistory retrieve transfer records between main and sub accounts.
// retrieve transfer records between main and sub accounts. Record time range cannot exceed 30 days
// Note: only records after 2020-04-10 can be retrieved
func (g *Gateio) GetSubAccountTransferHistory(ctx context.Context, subAccountUserID string, from, to time.Time, offset, limit int) ([]SubAccountTransferResponse, error) {
	var response []SubAccountTransferResponse
	params := url.Values{}
	if subAccountUserID != "" {
		params.Set("sub_uid", subAccountUserID)
	}
	startingTime, er := time.Parse("2006-Jan-02", "2020-Apr-10")
	if er != nil {
		return nil, er
	}
	if !from.IsZero() && from.After(startingTime) {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() && to.After(from) && to.Before(from.Add(time.Hour*720)) {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot,
		http.MethodGet, walletSubAccountTransfer, params, nil, &response)
}

// GetWithdrawalStatus retrives withdrawal status
func (g *Gateio) GetWithdrawalStatus(ctx context.Context, ccy currency.Code) ([]WithdrawalStatus, error) {
	var response []WithdrawalStatus
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, walletWithdrawStatus, params, nil, &response)
}

// GetSubAccountBalances retrieve sub account balances
func (g *Gateio) GetSubAccountBalances(ctx context.Context, subAccountUserID string) ([]SubAccountBalance, error) {
	var response []SubAccountBalance
	params := url.Values{}
	if subAccountUserID != "" {
		params.Set("sub_uid", subAccountUserID)
	}
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, walletSubAccountBalance, params, nil, &response)
}

// GetSubAccountMarginBalances query sub accounts' margin balances
func (g *Gateio) GetSubAccountMarginBalances(ctx context.Context, subAccountUserID string) ([]SubAccountMarginBalance, error) {
	var response []SubAccountMarginBalance
	params := url.Values{}
	if subAccountUserID != "" {
		params.Set("sub_uid", subAccountUserID)
	}
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, walletSubAccountMarginBalance, params, nil, &response)
}

// GetSubAccountFuturesBalances retrives sub accounts' futures account balances
func (g *Gateio) GetSubAccountFuturesBalances(ctx context.Context, subAccountUserID, settle string) ([]SubAccountBalance, error) {
	var response []SubAccountBalance
	params := url.Values{}
	if subAccountUserID != "" {
		params.Set("sub_uid", subAccountUserID)
	}
	if settle != "" {
		params.Set("settle", settle)
	}
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, walletSubAccountFuturesBalance, params, nil, &response)
}

// GetSubAccountCrossMarginBalances query subaccount's cross_margin account info
func (g *Gateio) GetSubAccountCrossMarginBalances(ctx context.Context, subAccountUserID string) ([]SubAccountCrossMarginInfo, error) {
	var response []SubAccountCrossMarginInfo
	params := url.Values{}
	if subAccountUserID != "" {
		params.Set("sub_uid", subAccountUserID)
	}
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, walletSubAccountCrossMarginBalances, params, nil, &response)
}

// GetSavedAddresses retrives saved currency address info and related details.
func (g *Gateio) GetSavedAddresses(ctx context.Context, currency currency.Code, chain string, limit int) ([]WalletSavedAddress, error) {
	var response []WalletSavedAddress
	params := url.Values{}
	if !currency.IsEmpty() {
		params.Set("currency", currency.String())
	}
	if chain != "" {
		params.Set("chain", chain)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, walletSavedAddress, params, nil, &response)
}

// GetPersonalTradingFee retrives personal trading fee
func (g *Gateio) GetPersonalTradingFee(ctx context.Context, currencyPair currency.Pair) (*PersonalTradingFee, error) {
	var response PersonalTradingFee
	params := url.Values{}
	if !(currencyPair.Quote.IsEmpty() || currencyPair.Base.IsEmpty()) {
		// specify a currency pair to retrieve precise fee rate
		currencyPair.Delimiter = currency.UnderscoreDelimiter
		params.Set("currency_pair", currencyPair.String())
	}
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, walletTradingFee, params, nil, &response)
}

// GetUsersTotalBalance retrieves user's total balances
func (g *Gateio) GetUsersTotalBalance(ctx context.Context, ccy currency.Code) (*UsersAllAccountBalance, error) {
	var response UsersAllAccountBalance
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, walletTotalBalance, params, nil, &response)
}

// ********************************* Margin *******************************************

// GetMarginSupportedCurrencyPair retrives margin supported currency pairs.
func (g *Gateio) GetMarginSupportedCurrencyPairs(ctx context.Context) ([]MarginCurrencyPairInfo, error) {
	var currenciePairsInfo []MarginCurrencyPairInfo
	return currenciePairsInfo, g.SendHTTPRequest(ctx, exchange.RestSpot, marginCurrencyPairs, &currenciePairsInfo)
}

// GetMarginSupportedCurrencyPair retrives margin supported currency pair detail given the currency pair.
func (g *Gateio) GetMarginSupportedCurrencyPair(ctx context.Context, cp currency.Pair) (*MarginCurrencyPairInfo, error) {
	var currencyPairInfo MarginCurrencyPairInfo
	if cp.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	fPair, er := g.GetPairFormat(asset.Margin, true)
	if er != nil {
		fPair = currency.PairFormat{
			Delimiter: currency.UnderscoreDelimiter,
			Uppercase: true,
		}
	}
	pairString := strings.ToUpper(fPair.Format(cp))
	path := fmt.Sprintf("%s/%s", marginCurrencyPairs, pairString)
	return &currencyPairInfo, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &currencyPairInfo)
}

// GetOrderbookOfLendingLoans retrives order book of lending loans for specific currency
func (g *Gateio) GetOrderbookOfLendingLoans(ctx context.Context, ccy currency.Code) ([]OrderbookOfLendingLoan, error) {
	var lendingLoans []OrderbookOfLendingLoan
	if ccy.IsEmpty() {
		return nil, errInvalidCurrency
	}
	path := fmt.Sprintf("%s?currency=%s", marginFundingBook, ccy.String())
	return lendingLoans, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &lendingLoans)
}

// GetMarginAccountList margin account list
func (g *Gateio) GetMarginAccountList(ctx context.Context, currencyPair currency.Pair) ([]MarginAccountItem, error) {
	var response []MarginAccountItem
	params := url.Values{}
	if !currencyPair.IsEmpty() {
		currencyPair.Delimiter = "_"
		params.Set("currency_pair", currencyPair.String())
	}
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginAccount, params, nil, &response)
}

// ListMarginAccountBalanceChangeHistory retrives retrives margin account balance change history
// Only transferals from and to margin account are provided for now. Time range allows 30 days at most
func (g *Gateio) ListMarginAccountBalanceChangeHistory(ctx context.Context, ccy currency.Code, currencyPair currency.Pair, from, to time.Time, page, limit int) ([]MarginAccountBalanceChangeInfo, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if !currencyPair.IsEmpty() {
		currencyPair.Delimiter = currency.UnderscoreDelimiter
		params.Set("currency_pair", currencyPair.String())
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() && ((!from.IsZero() && to.After(from)) || from.IsZero()) {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.Itoa(page))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	var response []MarginAccountBalanceChangeInfo
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginAccountBook, params, nil, &response)
}

// GetMarginFundingAccountList retrives funding account list
func (g *Gateio) GetMarginFundingAccountList(ctx context.Context, ccy currency.Code) ([]MarginFundingAccountItem, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var response []MarginFundingAccountItem
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginFundingAccounts, params, nil, &response)
}

// MarginLoan represents lend or borrow request
func (g *Gateio) MarginLoan(ctx context.Context, arg MarginLoanRequestParam) (*MarginLoanResponse, error) {
	if !(arg.Side == "lend" || arg.Side == "borrow") {
		return nil, errInvalidLoanSide
	}
	if arg.Side == "borrow" && arg.Rate == 0 {
		return nil, errors.New("`rate` is required in borrowing")
	}
	if arg.Currency.IsEmpty() {
		return nil, errInvalidCurrency
	}
	if arg.Amount <= 0 {
		return nil, errInvalidAmount
	}
	if !(arg.Rate <= 0.002 && arg.Rate >= 0.0002) {
		arg.Rate = 0
	}
	var response MarginLoanResponse
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, marginLoans, nil, &arg, &response)
}

// GetMarginAllLoans retrives all loans (borrow and lending) orders.
func (g *Gateio) GetMarginAllLoans(ctx context.Context, status, side string, ccy currency.Code, currencyPair currency.Pair, sortBy string, reverseSort bool, page, limit int) ([]MarginLoanResponse, error) {
	params := url.Values{}
	if side == "lend" || side == "borrow" {
		params.Set("side", side)
	}
	if status == "open" || status == "loaned" || status == "finished" || status == "auto_repair" {
		params.Set("status", status)
	} else {
		return nil, errors.New("loan status \"status\" is required")
	}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if !currencyPair.IsEmpty() {
		currencyPair.Delimiter = currency.UnderscoreDelimiter
		params.Set("currency_pair", currencyPair.String())
	}
	if sortBy == "create_time" || sortBy == "rate" {
		params.Set("sort_by", sortBy)
	}
	if reverseSort {
		params.Set("reverse_sort", strconv.FormatBool(reverseSort))
	}
	if page > 0 {
		params.Set("page", strconv.Itoa(page))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	var response []MarginLoanResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginLoans, params, nil, &response)
}

// MergeMultipleLendingLoans merge multiple lending loans
func (g *Gateio) MergeMultipleLendingLoans(ctx context.Context, ccy currency.Code, IDs []string) (*MarginLoanResponse, error) {
	params := url.Values{}
	if ccy.IsEmpty() {
		return nil, errInvalidCurrency
	}
	if len(IDs) < 2 || len(IDs) > 20 {
		return nil, errors.New("number of loans to be merged must be between [2-20], inclusive")
	}
	params.Set("currency", ccy.String())
	params.Set("ids", strings.Join(IDs, ","))
	var response MarginLoanResponse
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, marginMergedLoans, params, nil, &response)
}

// RetriveOneSingleLoanDetail retrieve one single loan detail
func (g *Gateio) RetriveOneSingleLoanDetail(ctx context.Context, side, loanID string) (*MarginLoanResponse, error) {
	params := url.Values{}
	if !(side == "borrow" || side == "lend") {
		return nil, errInvalidLoanSide
	}
	if loanID == "" {
		return nil, errInvalidLoanID
	}
	params.Set("side", side)
	path := fmt.Sprintf("%s/%s/", marginLoans, loanID)
	var response MarginLoanResponse
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, nil, &response)
}

// ModifyALoan Modify a loan
// Only auto_renew modification is supported currently
func (g *Gateio) ModifyALoan(ctx context.Context, loanID string, arg ModifyLoanRequestParam) (*MarginLoanResponse, error) {
	if loanID == "" {
		return nil, fmt.Errorf("%v, %v", errInvalidLoanID, " loan_id is required")
	}
	if arg.Currency.IsEmpty() {
		return nil, errInvalidCurrency
	}
	if !(arg.Side == "borrow" || arg.Side == "lend") {
		return nil, errInvalidLoanSide
	}
	if !arg.CurrencyPair.IsEmpty() {
		arg.CurrencyPair.Delimiter = currency.UnderscoreDelimiter
	}
	path := fmt.Sprintf("%s/%s", marginLoans, loanID)
	var response MarginLoanResponse
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPatch, path, nil, &arg, &response)
}

// CancelLendingLoan cancels lending loans. only lent loans can be canceled.
func (g *Gateio) CancelLendingLoan(ctx context.Context, currency currency.Code, loanID string) (*MarginLoanResponse, error) {
	if loanID == "" {
		return nil, fmt.Errorf("%v, %v", errInvalidLoanID, " loan_id is required")
	}
	if currency.IsEmpty() {
		return nil, errInvalidCurrency
	}
	params := url.Values{}
	params.Set("currency", currency.String())
	path := fmt.Sprintf("%s/%s", marginLoans, loanID)
	var response MarginLoanResponse
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, params, nil, &response)
}

// RepayALoan execute a loan repay.
func (g *Gateio) RepayALoan(ctx context.Context, loanID string, arg RepayLoanRequestParam) (*MarginLoanResponse, error) {
	if loanID == "" {
		return nil, fmt.Errorf("%v, %v", errInvalidLoanID, " loan_id is required")
	}
	if arg.Currency.IsEmpty() {
		return nil, errInvalidCurrency
	}
	if arg.CurrencyPair.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	arg.CurrencyPair.Delimiter = currency.UnderscoreDelimiter
	if !(arg.Mode == "all" || arg.Mode == "partial") {
		return nil, errInvalidRepayMode
	}
	if arg.Mode == "partial" && arg.Amount <= 0 {
		return nil, fmt.Errorf("%v, repay amount for partial repay mode must be greater than 0", errInvalidAmount)
	}
	var response MarginLoanResponse
	path := fmt.Sprintf("%s/%s/repayment", marginLoans, loanID)
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, &arg, &response)
}

// ListLoanRepaymentRecords retrives loan repayment records for specified loan ID
func (g *Gateio) ListLoanRepaymentRecords(ctx context.Context, loanID string) ([]LoanRepaymentRecord, error) {
	if loanID == "" {
		return nil, fmt.Errorf("%v, %v", errInvalidLoanID, " loan_id is required")
	}
	path := fmt.Sprintf("%s/%s/repayment", marginLoans, loanID)
	var response []LoanRepaymentRecord
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &response)
}

// ListRepaymentRecordsOfSpecificLoan retrieves repayment records of specific loan
func (g *Gateio) ListRepaymentRecordsOfSpecificLoan(ctx context.Context, loanID, status string, page, limit int) ([]LoanRecord, error) {
	if loanID == "" {
		return nil, fmt.Errorf("%v, %v", errInvalidLoanID, " loan_id is required")
	}
	params := url.Values{}
	params.Set("loan_id", loanID)
	if status == "loaned" || status == "finished" {
		params.Set("status", status)
	}
	if page > 0 {
		params.Set("page", strconv.Itoa(page))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	var response []LoanRecord
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginLoanRecords, params, nil, &response)
}

// GetOneSingleloanRecord get one single loan record
func (g *Gateio) GetOneSingleloanRecord(ctx context.Context, loanID, loanRecordID string) (*LoanRecord, error) {
	if loanID == "" {
		return nil, fmt.Errorf("%v, %v", errInvalidLoanID, " loan_id is required")
	}
	if loanRecordID == "" {
		return nil, fmt.Errorf("%v, %v", errInvalidLoanID, " loan_record_id is required")
	}
	path := fmt.Sprintf("%s/%s", marginLoanRecords, loanRecordID)
	params := url.Values{}
	params.Set("loan_id", loanID)
	var response LoanRecord
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, nil, &response)
}

// ModifyALoanRecord modify a loan record
// Only auto_renew modification is supported currently
func (g *Gateio) ModifyALoanRecord(ctx context.Context, loanRecordID string, arg ModifyLoanRequestParam) (*LoanRecord, error) {
	if loanRecordID == "" {
		return nil, fmt.Errorf("%v, %v", errInvalidLoanID, " loan_record_id is required")
	}
	if arg.LoanID == "" {
		return nil, fmt.Errorf("%v, %v", errInvalidLoanID, " loan_id is required")
	}
	if arg.Currency.IsEmpty() {
		return nil, errInvalidCurrency
	}
	if !(arg.Side == "borrow" || arg.Side == "lend") {
		return nil, errInvalidLoanSide
	}
	if !arg.CurrencyPair.IsEmpty() {
		arg.CurrencyPair.Delimiter = currency.UnderscoreDelimiter
	}
	path := fmt.Sprintf("%s/%s", marginLoanRecords, loanRecordID)
	var response LoanRecord
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPatch, path, nil, &arg, &response)
}

// UpdateUsersAutoRepaymentSetting represents update user's auto repayment setting
func (g *Gateio) UpdateUsersAutoRepaymentSetting(ctx context.Context, status string) (*OnOffStatus, error) {
	if !(status == "on" || status == "off") {
		return nil, errInvalidAutoRepaymentStatus
	}
	params := url.Values{}
	params.Set("status", status)
	var response OnOffStatus
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, marginAutoRepay, params, nil, &response)
}

// GetUserAutoRepaymentSetting retrieve user auto repayment setting
func (g *Gateio) GetUserAutoRepaymentSetting(ctx context.Context) (*OnOffStatus, error) {
	var response OnOffStatus
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginAutoRepay, nil, nil, &response)
}

// GetMaxTransferableAmountForSpecificMarginCurrency get the max transferable amount for a specific margin currency.
func (g *Gateio) GetMaxTransferableAmountForSpecificMarginCurrency(ctx context.Context, ccy currency.Code, currencyPair currency.Pair) (*MaxTransferAndLoanAmount, error) {
	if ccy.IsEmpty() {
		return nil, errInvalidCurrency
	}
	params := url.Values{}
	if !currencyPair.IsEmpty() {
		currencyPair.Delimiter = currency.UnderscoreDelimiter
		params.Set("currency_pair", currencyPair.String())
	}
	params.Set("currency", ccy.String())
	var response MaxTransferAndLoanAmount
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginTransfer, params, nil, &response)
}

// GetMaxBorrowableAmountForSpecificMarginCurrency retrives the max borrowble amount for specific currency
func (g *Gateio) GetMaxBorrowableAmountForSpecificMarginCurrency(ctx context.Context, ccy currency.Code, currencyPair currency.Pair) (*MaxTransferAndLoanAmount, error) {
	if ccy.IsEmpty() {
		return nil, errInvalidCurrency
	}
	params := url.Values{}
	if !currencyPair.IsEmpty() {
		currencyPair.Delimiter = currency.UnderscoreDelimiter
		params.Set("currency_pair", currencyPair.String())
	}
	params.Set("currency", ccy.String())
	var response MaxTransferAndLoanAmount
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginBorrowable, params, nil, &response)
}

// CurrencySupportedByCrossMargin currencies supported by cross margin.
func (g *Gateio) CurrencySupportedByCrossMargin(ctx context.Context) ([]CrossMarginCurrencies, error) {
	var response []CrossMarginCurrencies
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, crossMarginCurrencies, nil, nil, &response)
}

// GetCrossMarginSupportedCurrencyDetail retrieve detail of one single currency supported by cross margin
func (g *Gateio) GetCrossMarginSupportedCurrencyDetail(ctx context.Context, currency currency.Code) (*CrossMarginCurrencies, error) {
	if currency.IsEmpty() {
		return nil, errInvalidCurrency
	}
	path := fmt.Sprintf("%s/%s", crossMarginCurrencies, currency.String())
	var response CrossMarginCurrencies
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &response)
}

// GetCrossMarginAccounts retrieve cross margin account
func (g *Gateio) GetCrossMarginAccounts(ctx context.Context) (*CrossMarginAccount, error) {
	var response CrossMarginAccount
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, crossMarginAccounts, nil, nil, &response)
}

// GetCrossMarginAccountChangeHistory retrieve cross margin account change history
// Record time range cannot exceed 30 days
func (g *Gateio) GetCrossMarginAccountChangeHistory(ctx context.Context, currency currency.Code, from, to time.Time, page, limit int, accountChangeType string) ([]CrossMarginAccountHistoryItem, error) {
	params := url.Values{}
	if !currency.IsEmpty() {
		params.Set("currency", currency.String())
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.Itoa(page))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if accountChangeType == "in" || accountChangeType == "out" || accountChangeType == "repay" || accountChangeType == "borrow" || accountChangeType == "new_order" || accountChangeType == "order_fill" || accountChangeType == "referral_fee" || accountChangeType == "order_fee" || accountChangeType == "unknown" {
		params.Set("type", accountChangeType)
	}
	var response []CrossMarginAccountHistoryItem
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, crossMarginAccountBook, params, nil, &response)
}

// CreateCrossMarginBoBorrowLoan create a cross margin borrow loan
// Borrow amount cannot be less than currency minimum borrow amount
func (g *Gateio) CreateCrossMarginBorrowLoan(ctx context.Context, arg CrossMarginBorrowLoanParams) (*CrossMarginLoanResponse, error) {
	if arg.Currency.IsEmpty() {
		return nil, errInvalidCurrency
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%v, borrow amount must be greater than 0", errInvalidAmount)
	}
	var response CrossMarginLoanResponse
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, crossMarginLoans, nil, &arg, &response)
}

// ExecuteRepayment when the liquidity of the currency is insufficient and the transaction risk is high, the currency will be disabled,
// and funds cannot be transferred.When the available balance of cross-margin is insufficient, the balance of the spot account can be used for repayment.
// Please ensure that the balance of the spot account is sufficient, and system uses cross-margin account for repayment first
func (g *Gateio) ExecuteRepayment(ctx context.Context, arg CurrencyAndAmount) ([]CrossMarginLoanResponse, error) {
	if arg.Currency.IsEmpty() {
		return nil, errInvalidCurrency
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%v, repay amount must be greater than 0", errInvalidAmount)
	}
	var response []CrossMarginLoanResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, crossMarginRepayments, nil, &arg, &response)
}

// GetCrossMarginRepayments retrives list of cross margin repayments
func (g *Gateio) GetCrossMarginRepayments(ctx context.Context, ccy currency.Code, loanID string, limit, offset int, reverse bool) ([]CrossMarginLoanResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if loanID != "" {
		params.Set("loanId", loanID)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if reverse {
		params.Set("reverse", "true")
	}
	var response []CrossMarginLoanResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, crossMarginRepayments, params, nil, &response)
}

// GetMaxTransferableAmountForSpecificCrossMarginCurrency get the max transferable amount for a specific cross margin currency
func (g *Gateio) GetMaxTransferableAmountForSpecificCrossMarginCurrency(ctx context.Context, ccy currency.Code) (*CurrencyAndAmount, error) {
	if ccy.IsEmpty() {
		return nil, errInvalidCurrency
	}
	var response CurrencyAndAmount
	params := url.Values{}
	params.Set("currency", ccy.String())
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, crossMarginTransferable, params, nil, &response)
}

// GetMaxBorrowableAmountForSpecificCrossMarginCurrency returns the max borrowable amount for a specific cross margin currency
func (g *Gateio) GetMaxBorrowableAmountForSpecificCrossMarginCurrency(ctx context.Context, ccy currency.Code) (*CurrencyAndAmount, error) {
	if ccy.IsEmpty() {
		return nil, errInvalidCurrency
	}
	var response CurrencyAndAmount
	params := url.Values{}
	params.Set("currency", ccy.String())
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, crossMarginBorrowable, params, nil, &response)
}

// GetCrossMarginBorrowHistory retrives cross margin borrow history sorted by creation time in descending order by default.
// Set reverse=false to return ascending results.
func (g *Gateio) GetCrossMarginBorrowHistory(ctx context.Context, status uint, currency currency.Code, limit, offset int, reverse bool) ([]CrossMarginLoanResponse, error) {
	params := url.Values{}
	if !(status >= 1 && status <= 3) {
		return nil, fmt.Errorf("%s %v, only allowed status values are 1:failed, 2:borrowed, and 3:repayment", g.Name, errInvalidOrderStatus)
	}
	params.Set("status", strconv.Itoa(int(status)))
	if !currency.IsEmpty() {
		params.Set("currency", currency.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if reverse {
		params.Set("reverse", strconv.FormatBool(reverse))
	}
	var response []CrossMarginLoanResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, crossMarginLoans, params, nil, &response)
}

// GetSingleBorrowLoanDetail retrieve single borrow loan detail
func (g *Gateio) GetSingleBorrowLoanDetail(ctx context.Context, loanID string) (*CrossMarginLoanResponse, error) {
	if loanID == "" {
		return nil, errInvalidLoanID
	}
	var response CrossMarginLoanResponse
	path := fmt.Sprintf("%s/%s", crossMarginLoans, loanID)
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &response)
}

// *********************************Futures***************************************

// GetAllFutureContracts  retrives list all futures contracts
func (g *Gateio) GetAllFutureContracts(ctx context.Context, settle string) ([]FuturesContract, error) {
	var contracts []FuturesContract
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	path := fmt.Sprintf(futuresSettleContracts, settle)
	return contracts, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &contracts)
}

// GetSingleContract returns a single contract info for the specified settle and Currency Pair (contract << in this case)
func (g *Gateio) GetSingleContract(ctx context.Context, settle string, contract currency.Pair) (*FuturesContract, error) {
	var futureContract FuturesContract
	if contract.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	fPair, er := g.GetPairFormat(asset.Futures, true)
	if er != nil {
		fPair = currency.PairFormat{
			Delimiter: currency.UnderscoreDelimiter,
			Uppercase: true,
		}
	}
	pairString := strings.ToUpper(fPair.Format(contract))
	path := fmt.Sprintf(futuresSettleContracts+"/%s", settle, pairString)
	return &futureContract, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &futureContract)
}

// GetFuturesOrderbook retrives futures order book data
func (g *Gateio) GetFuturesOrderbook(ctx context.Context, settle string, contract currency.Pair, interval string, limit uint, withOrderbookID bool) (*Orderbook, error) {
	var response Orderbook
	if contract.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	params := url.Values{}
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	fPair, er := g.GetPairFormat(asset.Spot, true)
	if er != nil {
		fPair = currency.PairFormat{
			Delimiter: currency.UnderscoreDelimiter,
			Uppercase: true,
		}
	}
	pairString := strings.ToUpper(fPair.Format(contract))
	params.Set("contract", pairString)
	if interval == OrderbookIntervalZero || interval == OrderbookIntervalZeroPt1 || interval == OrderbookIntervalZeroPtZero1 {
		params.Set("interval", interval)
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	params.Set("with_id", strconv.FormatBool(withOrderbookID))
	return &response, g.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(fmt.Sprintf(futuresOrderbook, settle), params), &response)
}

// GetFuturesTradingHistory retrives futures trading history
func (g *Gateio) GetFuturesTradingHistory(ctx context.Context, settle string, contract currency.Pair, limit, offset uint, lastID string, from, to time.Time) ([]TradingHistoryItem, error) {
	var response []TradingHistoryItem
	params := url.Values{}
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	fPair, er := g.GetPairFormat(asset.Futures, true)
	if er != nil {
		fPair = currency.PairFormat{
			Delimiter: currency.UnderscoreDelimiter,
			Uppercase: true,
		}
	}
	pairString := strings.ToUpper(fPair.Format(contract))
	params.Set("contract", pairString)
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	if offset > 0 {
		params.Set("offset", fmt.Sprintf("%v", offset))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	path := common.EncodeURLValues(fmt.Sprintf(futuresTrades, settle), params)
	return response, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &response)
}

// GetFuturesCandlesticks retrives specified contract candlesticks.
func (g *Gateio) GetFuturesCandlesticks(ctx context.Context, settle string, contract currency.Pair, from, to time.Time, limit uint, interval kline.Interval) ([]FuturesCandlestick, error) {
	var candlesticks []FuturesCandlestick
	params := url.Values{}
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	fPair, er := g.GetPairFormat(asset.Futures, true)
	if er != nil {
		fPair = currency.PairFormat{
			Delimiter: currency.UnderscoreDelimiter,
			Uppercase: true,
		}
	}
	pairString := strings.ToUpper(fPair.Format(contract))
	params.Set("contract", pairString)
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	if intervalString := g.GetIntervalString(interval); intervalString != "" {
		params.Set("interval", intervalString)
	}
	path := common.EncodeURLValues(fmt.Sprintf(futuresCandlesticks, settle), params)
	return candlesticks, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &candlesticks)
}

// GetFuturesTickers retrives futures ticker information for a specific settle and contract info.
func (g *Gateio) GetFuturesTickers(ctx context.Context, settle string, contract currency.Pair) ([]FuturesTicker, error) {
	var tickers []FuturesTicker
	params := url.Values{}
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if !contract.IsEmpty() {
		fPair, er := g.GetPairFormat(asset.Futures, true)
		if er != nil {
			fPair = currency.PairFormat{
				Delimiter: currency.UnderscoreDelimiter,
				Uppercase: true,
			}
		}
		pairString := strings.ToUpper(fPair.Format(contract))
		params.Set("contract", pairString)
	}
	return tickers, g.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(fmt.Sprintf(futuresTicker, settle), params), &tickers)
}

// GetFutureFundingRates retrives funding rate information.
func (g *Gateio) GetFutureFundingRates(ctx context.Context, settle string, contract currency.Pair, limit uint) ([]FuturesFundingRate, error) {
	var rates []FuturesFundingRate
	params := url.Values{}
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	fPair, er := g.GetPairFormat(asset.Futures, true)
	if er != nil {
		fPair = currency.PairFormat{
			Delimiter: currency.UnderscoreDelimiter,
			Uppercase: true,
		}
	}
	pairString := strings.ToUpper(fPair.Format(contract))
	params.Set("contract", pairString)
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	path := common.EncodeURLValues(fmt.Sprintf(futuresFundingRate, settle), params)
	return rates, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &rates)
}

// GetFuturesInsuranceBalanceHistory retrives futures insurance balance history
func (g *Gateio) GetFuturesInsuranceBalanceHistory(ctx context.Context, settle string, limit uint) ([]InsuranceBalance, error) {
	var balances []InsuranceBalance
	params := url.Values{}
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	path := common.EncodeURLValues(fmt.Sprintf(futuresInsuranceBalance, settle), params)
	return balances, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &balances)
}

// GetFutureStats retrives futures stats
func (g *Gateio) GetFutureStats(ctx context.Context, settle string, contract currency.Pair, from time.Time, interval kline.Interval, limit uint) ([]ContractStat, error) {
	var stats []ContractStat
	params := url.Values{}
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	fPair, er := g.GetPairFormat(asset.Futures, true)
	if er != nil {
		fPair = currency.PairFormat{
			Delimiter: currency.UnderscoreDelimiter,
			Uppercase: true,
		}
	}
	pairString := strings.ToUpper(fPair.Format(contract))
	params.Set("contract", pairString)
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if intervalString := g.GetIntervalString(interval); intervalString != "" {
		params.Set("interval", intervalString)
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	path := common.EncodeURLValues(fmt.Sprintf(futuresContractStats, settle), params)
	return stats, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &stats)
}

// GetIndexConstituent retrives index constituents
func (g *Gateio) GetIndexConstituent(ctx context.Context, settle string, index currency.Pair) (*IndexConstituent, error) {
	var constituents IndexConstituent
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if index.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	fPair, er := g.GetPairFormat(asset.Futures, true)
	if er != nil {
		fPair = currency.PairFormat{
			Delimiter: currency.UnderscoreDelimiter,
			Uppercase: true,
		}
	}
	indexString := strings.ToUpper(fPair.Format(index))
	path := fmt.Sprintf(futuresIndexConstituent, settle, indexString)
	return &constituents, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &constituents)
}

// GetLiquidationHistory retrives liqudiation history
func (g *Gateio) GetLiquidationHistory(ctx context.Context, settle string, contract currency.Pair, from, to time.Time, limit uint) ([]LiquidationHistory, error) {
	var histories []LiquidationHistory
	params := url.Values{}
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	fPair, er := g.GetPairFormat(asset.Futures, true)
	if er != nil {
		fPair = currency.PairFormat{
			Delimiter: currency.UnderscoreDelimiter,
			Uppercase: true,
		}
	}
	pairString := strings.ToUpper(fPair.Format(contract))
	params.Set("contract", pairString)
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	path := common.EncodeURLValues(fmt.Sprintf(futuresLiquidationHistory, settle), params)
	return histories, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &histories)
}

// QueryFuturesAccount retrives futures account
func (g *Gateio) QueryFuturesAccount(ctx context.Context, settle string) (*FuturesAccount, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	path := fmt.Sprintf(futuresAccounts, settle)
	var response FuturesAccount
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &response)
}

// GetFuturesAccountBooks retrives account books
func (g *Gateio) GetFuturesAccountBooks(ctx context.Context, settle string, limit int, from, to time.Time, changingType string) ([]AccountBookItem, error) {
	params := url.Values{}
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if changingType == "dnw" ||
		changingType == "pnl" ||
		changingType == "fee" ||
		changingType == "refr" ||
		changingType == "fund" ||
		changingType == "point_dnw" ||
		changingType == "point_fee" ||
		changingType == "point_refr" {
		params.Set("type", changingType)
	}
	path := fmt.Sprintf(futuresAccountBook, settle)
	var response []AccountBookItem
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, nil, &response)
}

// GetAllFuturesPositionsOfUsers list all positions of users.
func (g *Gateio) GetAllFuturesPositionsOfUsers(ctx context.Context, settle string) (*Position, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	path := fmt.Sprintf(futuresPositions, settle)
	var response Position
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &response)
}

// GetSinglePosition returns a single position
func (g *Gateio) GetSinglePosition(ctx context.Context, settle string, contract currency.Pair) (*Position, error) {
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract.Base.IsEmpty() || contract.Quote.IsEmpty() {
		return nil, fmt.Errorf("%v, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	contract.Delimiter = currency.UnderscoreDelimiter
	path := fmt.Sprintf(futuresSinglePosition, settle, contract.String())
	var response Position
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, nil, &response)
}

// UpdateFuturesPositionMargin represents account position margin for a futures contract.
func (g *Gateio) UpdateFuturesPositionMargin(ctx context.Context, settle string, change float64, contract currency.Pair) (*Position, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract.Base.IsEmpty() || contract.Quote.IsEmpty() {
		return nil, fmt.Errorf("%v, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	if change <= 0 {
		return nil, fmt.Errorf("%v, futures margin change must be positive", errChangehasToBePositive)
	}
	params := url.Values{}
	params.Set("change", strconv.FormatFloat(change, 'f', -1, 64))
	contract.Delimiter = currency.UnderscoreDelimiter
	path := fmt.Sprintf(futuresUpdatePositionMargin, settle, contract.String())
	var response Position
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, params, nil, &response)
}

// UpdateFuturesPositionLeverage update position leverage
func (g *Gateio) UpdateFuturesPositionLeverage(ctx context.Context, settle string, contract currency.Pair, leverage, crossLeverageLimit float64) (*Position, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract.Base.IsEmpty() || contract.Quote.IsEmpty() {
		return nil, fmt.Errorf("%v, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	contract.Delimiter = currency.UnderscoreDelimiter
	params := url.Values{}
	if leverage < 0 {
		return nil, errInvalidLeverageValue
	}
	params.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	if leverage == 0 && crossLeverageLimit > 0 {
		params.Set("cross_leverage_limit", strconv.FormatFloat(crossLeverageLimit, 'f', -1, 64))
	}
	path := fmt.Sprintf(futuresPositionsLeverage, settle, contract.String())
	var response Position
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, params, nil, &response)
}

// UpdateFuturesPositionRiskLimit updates the position risk limit
func (g *Gateio) UpdateFuturesPositionRiskLimit(ctx context.Context, settle string, contract currency.Pair, riskLimit int) (*Position, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract.Base.IsEmpty() || contract.Quote.IsEmpty() {
		return nil, fmt.Errorf("%v, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	contract.Delimiter = currency.UnderscoreDelimiter
	params := url.Values{}
	if riskLimit < 0 {
		return nil, errInvalidRiskLimit
	}
	params.Set("risk_limit", strconv.Itoa(riskLimit))
	path := fmt.Sprintf(futuresPositionRiskLimit, settle, contract.String())
	var response Position
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, params, nil, &response)
}

// EnableOrDisableDualMode enable or disable dual mode
// Before setting dual mode, make sure all positions are closed and no orders are open
func (g *Gateio) EnableOrDisableDualMode(ctx context.Context, settle string, dualMode bool) (*DualModeResponse, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	path := fmt.Sprintf(futuresDualMode, settle)
	params := url.Values{}
	params.Set("dual_mode", strconv.FormatBool(dualMode))
	var response DualModeResponse
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, nil, &response)
}

// RetrivePositionDetailInDualMode retrieve position detail in dual mode
func (g *Gateio) RetrivePositionDetailInDualMode(ctx context.Context, settle string, contract currency.Pair) ([]Position, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract.Base.IsEmpty() || contract.Quote.IsEmpty() {
		return nil, fmt.Errorf("%v, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	contract.Delimiter = currency.UnderscoreDelimiter
	path := fmt.Sprintf(futuresDualModePositions, settle, contract.String())
	var response []Position
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &response)
}

// UpdatePositionMarginInDualMode update position margin in dual mode
func (g *Gateio) UpdatePositionMarginInDualMode(ctx context.Context, settle string, contract currency.Pair, change float64, dualSide string) ([]Position, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract.Base.IsEmpty() || contract.Quote.IsEmpty() {
		return nil, fmt.Errorf("%v, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	contract.Delimiter = currency.UnderscoreDelimiter
	params := url.Values{}
	params.Set("change", strconv.FormatFloat(change, 'f', -1, 64))
	if !(dualSide == "dual_long" || dualSide == "dual_short") {
		return nil, fmt.Errorf("invalid 'dual_side' should be 'dual_short' or 'dual_long'")
	}
	params.Set("dual_side", dualSide)
	path := fmt.Sprintf(futuresDualModePositionMargin, settle, contract.String())
	var response []Position
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, nil, &response)
}

// UpdatePositionLeverageInDualMode update position leverage in dual mode
func (g *Gateio) UpdatePositionLeverageInDualMode(ctx context.Context, settle string, contract currency.Pair, leverage, crossLeverageLimit float64) (*Position, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract.Base.IsEmpty() || contract.Quote.IsEmpty() {
		return nil, fmt.Errorf("%v, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	contract.Delimiter = currency.UnderscoreDelimiter
	params := url.Values{}
	if leverage < 0 {
		return nil, errInvalidLeverageValue
	}
	params.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	if leverage == 0 && crossLeverageLimit > 0 {
		params.Set("cross_leverage_limit", strconv.FormatFloat(crossLeverageLimit, 'f', -1, 64))
	}
	path := fmt.Sprintf(futuresDualModePositionLeverage, settle, contract.String())
	var response Position
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, nil, &response)
}

// UpdatePositionRiskLimitinDualMode update position risk limit in dual mode
func (g *Gateio) UpdatePositionRiskLimitinDualMode(ctx context.Context, settle string, contract currency.Pair, riskLimit float64) ([]Position, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract.Base.IsEmpty() || contract.Quote.IsEmpty() {
		return nil, fmt.Errorf("%v, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	params := url.Values{}
	contract.Delimiter = currency.UnderscoreDelimiter
	if riskLimit < 0 {
		return nil, errInvalidRiskLimit
	}
	params.Set("risk_limit", strconv.FormatFloat(riskLimit, 'f', -1, 64))
	path := fmt.Sprintf(futuresDualModePositionsRiskLimit, settle, contract.String())
	var response []Position
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, nil, &response)
}

// CreateFuturesOrder creates futures order
// Create a futures order
// Creating futures orders requires size, which is number of contracts instead of currency amount. You can use quanto_multiplier in contract detail response to know how much currency 1 size contract represents
// Zero-filled order cannot be retrieved 10 minutes after order cancellation. You will get a 404 not found for such orders
// Set reduce_only to true can keep the position from changing side when reducing position size
// In single position mode, to close a position, you need to set size to 0 and close to true
// In dual position mode, to close one side position, you need to set auto_size side, reduce_only to true and size to 0
func (g *Gateio) CreateFuturesOrder(ctx context.Context, arg OrderCreateParams) (*Order, error) {
	if arg.Contract.Base.IsEmpty() || arg.Contract.Quote.IsEmpty() {
		return nil, fmt.Errorf("%v, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	if arg.Size == 0 {
		return nil, fmt.Errorf("%v, specify positive number to make a bid, and negative number to ask", errInvalidOrderSide)
	}
	if !(arg.TimeInForce == "gtc" || arg.TimeInForce == "ioc" || arg.TimeInForce == "poc" || arg.TimeInForce == "fok") {
		return nil, errInvalidTimeInForce
	}
	if arg.Price > 0 && arg.TimeInForce == "ioc" {
		arg.Price = 0
	}
	if arg.Price < 0 {
		return nil, errInvalidPrice
	}
	if arg.Text != "" && !strings.HasPrefix(arg.Text, "t-") {
		arg.Text = "t-" + arg.Text
	}
	if arg.AutoSize != "" && (arg.AutoSize == "close_long" || arg.AutoSize == "close_short") {
		return nil, errInvalidAutoSizeValue
	}
	if !(arg.Settle == "btc" || arg.Settle == "usd" || arg.Settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	path := fmt.Sprintf(futuresOrders, arg.Settle)
	var response Order
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, &arg, &response)
}

// GetFuturesOrders retrives list of futures orders
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (g *Gateio) GetFuturesOrders(ctx context.Context, contract currency.Pair, status string, limit, offset int, lastID string, countTotal int, settle string) ([]Order, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract.Base.IsEmpty() || contract.Quote.IsEmpty() {
		return nil, fmt.Errorf("%v, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	params := url.Values{}
	contract.Delimiter = currency.UnderscoreDelimiter
	params.Set("contract", contract.String())
	if !(status == "open" || status == "finished") {
		return nil, fmt.Errorf("%v, only 'open' and 'finished' status are supported", errInvalidOrderStatus)
	}
	params.Set("status", status)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	if countTotal == 1 && status != "open" {
		params.Set("count_total", strconv.Itoa(countTotal))
	} else if countTotal != 0 && countTotal != 1 {
		return nil, errInvalidCountTotalValue
	}
	path := fmt.Sprintf(futuresOrders, settle)
	var response []Order
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, nil, &response)
}

// CancelAllFuturesOpenOrdersMatched ancel all open orders
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (g *Gateio) CancelAllFuturesOpenOrdersMatched(ctx context.Context, contract currency.Pair, side, settle string) ([]Order, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract.Base.IsEmpty() || contract.Quote.IsEmpty() {
		return nil, fmt.Errorf("%v, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	params := url.Values{}
	contract.Delimiter = currency.UnderscoreDelimiter
	if side == "ask" || side == "bid" {
		params.Set("side", side)
	}
	params.Set("contract", contract.String())
	path := fmt.Sprintf(futuresOrders, settle)
	var response []Order
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, params, nil, &response)
}

// CreateBatchFuturesOrders creates a list of futures orders
// Up to 10 orders per request
// If any of the order's parameters are missing or in the wrong format, all of them will not be executed, and a http status 400 error will be returned directly
// If the parameters are checked and passed, all are executed. Even if there is a business logic error in the middle (such as insufficient funds), it will not affect other execution orders
// The returned result is in array format, and the order corresponds to the orders in the request body
// In the returned result, the succeeded field of type bool indicates whether the execution was successful or not
// If the execution is successful, the normal order content is included; if the execution fails, the label field is included to indicate the cause of the error
// In the rate limiting, each order is counted individually
func (g *Gateio) CreateBatchFuturesOrders(ctx context.Context, settle string, args []OrderCreateParams) ([]Order, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if len(args) > 10 {
		return nil, errTooManyOrderRequest
	}
	for x := range args {
		if args[x].Size == 0 {
			return nil, fmt.Errorf("%v, specify positive number to make a bid, and negative number to ask", errInvalidOrderSide)
		}
		if !(args[x].TimeInForce == "gtc" || args[x].TimeInForce == "ioc" || args[x].TimeInForce == "poc" || args[x].TimeInForce == "fok") {
			return nil, errInvalidTimeInForce
		}
		if args[x].Price > 0 && args[x].TimeInForce == "ioc" {
			args[x].Price = 0
		}
		if args[x].Price < 0 {
			return nil, errInvalidPrice
		}
		if args[x].Text != "" && !strings.HasPrefix(args[x].Text, "t-") {
			args[x].Text = "t-" + args[x].Text
		}
		if args[x].AutoSize != "" && (args[x].AutoSize == "close_long" || args[x].AutoSize == "close_short") {
			return nil, errInvalidAutoSizeValue
		}
		if !(args[x].Settle == "btc" || args[x].Settle == "usd" || args[x].Settle == "usdt") {
			return nil, errMissingSettleCurrency
		}
	}
	path := fmt.Sprintf(futuresBatchOrders, settle)
	var response []Order
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, &args, &response)
}

// GetSingleOrder retrives a single order by its identifier
func (g *Gateio) GetSingleFuturesOrder(ctx context.Context, settle, orderID string) (*Order, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if orderID == "" {
		return nil, fmt.Errorf("%v, 'order_id' cannot be empty", errInvalidOrderID)
	}
	path := fmt.Sprintf(futuresOrders+"/%s", settle, orderID)
	var response Order
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &response)
}

// CancelSingleFuturesOrder cancel a single order
func (g *Gateio) CancelSingleFuturesOrder(ctx context.Context, settle, orderID string) (*Order, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if orderID == "" {
		return nil, fmt.Errorf("%v, 'order_id' cannot be empty", errInvalidOrderID)
	}
	path := fmt.Sprintf(futuresOrders+"/%s", settle, orderID)
	var response Order
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, nil, nil, &response)
}

// AmendFuturesOrder amends an existing futures order
func (g *Gateio) AmendFuturesOrder(ctx context.Context, settle, orderID string, arg AmendFuturesOrderParam) (*Order, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if orderID == "" {
		return nil, fmt.Errorf("%v, 'order_id' cannot be empty", errInvalidOrderID)
	}
	if arg.Size <= 0 && arg.Price <= 0 {
		return nil, errors.New("missing update 'size' or 'price', please specify 'size' or 'price' or both information")
	}
	path := fmt.Sprintf(futuresOrders+"/%s", settle, orderID)
	var response Order
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPut, path, nil, &arg, &response)
}

// GetMyPersonalTradingHistory retrives my personal trading history
func (g *Gateio) GetMyPersonalTradingHistory(ctx context.Context, settle string, contract currency.Pair, orderID string, limit, offset, countTotal int, lastID string) ([]TradingHistoryItem, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	params := url.Values{}
	if !(contract.Base.IsEmpty() || contract.Quote.IsEmpty()) {
		contract.Delimiter = currency.UnderscoreDelimiter
		params.Set("contract", contract.String())
	}
	if orderID != "" {
		params.Set("order", orderID)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	if countTotal == 1 {
		params.Set("count_total", strconv.Itoa(countTotal))
	}
	path := fmt.Sprintf(futuresPersonalTradingHistory, settle)
	var response []TradingHistoryItem
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, nil, &response)
}

// GetFuturesPositionCloseHistory lists position close history
func (g *Gateio) GetFuturesPositionCloseHistory(ctx context.Context, settle string, contract currency.Pair, limit, offset int, from, to time.Time) ([]PositionCloseHistoryResponse, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	params := url.Values{}
	if !(contract.Base.IsEmpty() || contract.Quote.IsEmpty()) {
		contract.Delimiter = currency.UnderscoreDelimiter
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	path := fmt.Sprintf(futuresPositionClose, settle)
	var response []PositionCloseHistoryResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, nil, &response)
}

// GetFuturesLiquidationHistory list liquidation history
func (g *Gateio) GetFuturesLiquidationHistory(ctx context.Context, settle string, contract currency.Pair, limit int, at time.Time) ([]LiquidationHistoryItem, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	params := url.Values{}
	if !(contract.Base.IsEmpty() || contract.Quote.IsEmpty()) {
		contract.Delimiter = currency.UnderscoreDelimiter
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if !at.IsZero() {
		params.Set("at", strconv.FormatInt(at.Unix(), 10))
	}
	path := fmt.Sprintf(futuresLiquidations, settle)
	var response []LiquidationHistoryItem
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, nil, &response)
}

// CountdownCancelOrders represents a trigger time response
func (g *Gateio) CountdownCancelOrders(ctx context.Context, settle string, arg CountdownParams) (*TriggerTimeResponse, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if arg.Timeout < 0 {
		return nil, errInvalidTimeout
	}
	if !(arg.Contract.Base.IsEmpty() || arg.Contract.Quote.IsEmpty()) {
		arg.Contract.Delimiter = currency.UnderscoreDelimiter
	}
	path := fmt.Sprintf(futuresCountdownCancelAll, settle)
	var response TriggerTimeResponse
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, &arg, &response)
}

// CreatePriceTriggeredFuturesOrder create a price-triggered order
func (g *Gateio) CreatePriceTriggeredFuturesOrder(ctx context.Context, settle string, arg FuturesPriceTriggeredOrderParam) (*OrderID, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if arg.Initial.Contract.Base.IsEmpty() || arg.Initial.Contract.Quote.IsEmpty() {
		return nil, fmt.Errorf("%v, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	arg.Initial.Contract.Delimiter = currency.UnderscoreDelimiter
	if arg.Initial.Price < 0 {
		return nil, fmt.Errorf("%v, price must be greater than 0", errInvalidPrice)
	}
	// if arg.Initial.Size <= 0 {
	// 	return nil, errors.New("invalid argument: initial.size out of range")
	// }
	if arg.Initial.TimeInForce != "" && !(arg.Initial.TimeInForce == "gtc" || arg.Initial.TimeInForce == "ioc") {
		return nil, fmt.Errorf("%v, only time in force value 'gtc' and 'ioc' are supported", errInvalidTimeInForce)
	}
	if !(arg.Trigger.StrategyType == 0 || arg.Trigger.StrategyType == 1) {
		return nil, fmt.Errorf("strategy type must be 0 or 1, 0: by price, and 1: by price gap")
	}
	if !(arg.Trigger.Rule == 1 || arg.Trigger.Rule == 2) {
		return nil, fmt.Errorf("invalid trigger condition('rule') value, rule must be 1 or 2")
	}
	if !(arg.Trigger.PriceType == 0 || arg.Trigger.PriceType == 1 || arg.Trigger.PriceType == 2) {
		return nil, fmt.Errorf("price type must be 0 or 1 or 2")
	}
	// if arg.Trigger.Price <= 0 {
	// 	return nil, errors.New("invalid argument: trigger.price")
	// }
	if arg.Trigger.OrderType != "" && !(arg.Trigger.OrderType == "close-long-order" ||
		arg.Trigger.OrderType == "close-short-order" ||
		arg.Trigger.OrderType == "close-long-position" ||
		arg.Trigger.OrderType == "close-short-position" ||
		arg.Trigger.OrderType == "plan-close-long-position" ||
		arg.Trigger.OrderType == "plan-close-short-position") {
		return nil, errors.New("invalid order type, only 'close-long-order', 'close-short-order', 'close-long-position', 'close-short-position', 'plan-close-long-position', and 'plan-close-short-position'")
	}
	var response OrderID
	path := fmt.Sprintf(futuresPriceOrders, settle)
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, &arg, &response)
}

// ListAllFuturesAutoOrders lists all open orders
func (g *Gateio) ListAllFuturesAutoOrders(ctx context.Context, status, settle string, contract currency.Pair, limit, offset int) ([]PriceTriggeredOrder, error) {
	if !(status == "open" || status == "finished") {
		return nil, errInvalidOrderStatus
	}
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	path := fmt.Sprintf(futuresPriceOrders, settle)
	params := url.Values{}
	params.Set("status", status)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if !(contract.Base.IsEmpty() || contract.Quote.IsEmpty()) {
		contract.Delimiter = currency.UnderscoreDelimiter
		params.Set("contract", contract.String())
	}
	var response []PriceTriggeredOrder
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, nil, &response)
}

// CancelAllFuturesOpenOrders cancels all futures open orders
func (g *Gateio) CancelAllFuturesOpenOrders(ctx context.Context, settle string, contract currency.Pair) ([]PriceTriggeredOrder, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract.Base.IsEmpty() || contract.Quote.IsEmpty() {
		return nil, fmt.Errorf("%v, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	params := url.Values{}
	contract.Delimiter = currency.UnderscoreDelimiter
	path := fmt.Sprintf(futuresPriceOrders, settle)
	params.Set("contract", contract.String())
	var response []PriceTriggeredOrder
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, params, nil, &response)
}

// GetSingleFuturesPriceTriggeredOrder retrives a single price triggered order
func (g *Gateio) GetSingleFuturesPriceTriggeredOrder(ctx context.Context, settle, orderID string) (*PriceTriggeredOrder, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if orderID == "" {
		return nil, errInvalidOrderID
	}
	path := fmt.Sprintf(futuresPriceOrders+"/%s", settle, orderID)
	var response PriceTriggeredOrder
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &response)
}

// CancelFuturesPriceTriggeredOrder cancel a price-triggered order
func (g *Gateio) CancelFuturesPriceTriggeredOrder(ctx context.Context, settle, orderID string) (*PriceTriggeredOrder, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if orderID == "" {
		return nil, errInvalidOrderID
	}
	path := fmt.Sprintf(futuresPriceOrders+"/%s", settle, orderID)
	var response PriceTriggeredOrder
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, nil, nil, &response)
}

// *************************************** Delivery ***************************************

// GetAllDeliveryContracts retrives all futures contracts
func (g *Gateio) GetAllDeliveryContracts(ctx context.Context, settle string) ([]DeliveryContract, error) {
	var contracts []DeliveryContract
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	path := fmt.Sprintf(deliveryContracts, settle)
	return contracts, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &contracts)
}

// GetSingleDeliveryContracts retrives a single delivery contract instance.
func (g *Gateio) GetSingleDeliveryContracts(ctx context.Context, settle, contract string) (*DeliveryContract, error) {
	var deliveryContract DeliveryContract
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	path := fmt.Sprintf(deliveryContracts+"/%s", settle, contract)
	return &deliveryContract, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &deliveryContract)
}

// GetDeliveryOrderbook delivery orderbook
func (g *Gateio) GetDeliveryOrderbook(ctx context.Context, settle, contract, interval string, limit uint, withOrderbookID bool) (*Orderbook, error) {
	var orderbook Orderbook
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	params := url.Values{}
	if contract == "" {
		return nil, errInvalidOrMissingContractParam
	}
	params.Set("contract", contract)
	if interval == OrderbookIntervalZero || interval == OrderbookIntervalZeroPt1 || interval == OrderbookIntervalZeroPtZero1 {
		params.Set("interval", interval)
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	if withOrderbookID {
		params.Set("with_id", strconv.FormatBool(withOrderbookID))
	}
	return &orderbook, g.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(fmt.Sprintf(deliveryOrderbook, settle), params), &orderbook)
}

// GetDeliveryTradingHistory retrives futures trading history
func (g *Gateio) GetDeliveryTradingHistory(ctx context.Context, settle, contract string, limit uint, lastID string, from, to time.Time) ([]DeliveryTradingHistory, error) {
	var histories []DeliveryTradingHistory
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	params := url.Values{}
	if contract == "" {
		return nil, errInvalidOrMissingContractParam
	}
	params.Set("contract", contract)
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	path := common.EncodeURLValues(fmt.Sprintf(deliveryTradeHistory, settle), params)
	return histories, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &histories)
}

// GetDeliveryFuturesCandlesticks retrives specified contract candlesticks
func (g *Gateio) GetDeliveryFuturesCandlesticks(ctx context.Context, settle, contract string, from, to time.Time, limit uint, interval kline.Interval) ([]FuturesCandlestick, error) {
	var candlesticks []FuturesCandlestick
	params := url.Values{}
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract == "" {
		return nil, errInvalidOrMissingContractParam
	}
	params.Set("contract", contract)
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	if intervalString := g.GetIntervalString(interval); intervalString != "" {
		params.Set("interval", intervalString)
	}
	path := common.EncodeURLValues(fmt.Sprintf(deliveryCandlesticks, settle), params)
	return candlesticks, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &candlesticks)
}

// GetDeliveryFutureTickers retrives futures ticker information for a specific settle and contract info.
func (g *Gateio) GetDeliveryFutureTickers(ctx context.Context, settle string, contract currency.Pair) ([]FuturesTicker, error) {
	var tickers []FuturesTicker
	params := url.Values{}
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if !(contract.Base.IsEmpty() || contract.Quote.IsEmpty()) {
		contract.Delimiter = currency.UnderscoreDelimiter
		con, err := g.GetContractFromCurrencyPair(context.Background(), contract, asset.DeliveryFutures)
		if err != nil {
			return nil, err
		}
		params.Set("contract", con)
	}
	return tickers, g.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(fmt.Sprintf(deliveryTicker, settle), params), &tickers)
}

// GetDeliveryInsuranceBalanceHistory retrives delivery futures insurance balance history
func (g *Gateio) GetDeliveryInsuranceBalanceHistory(ctx context.Context, settle string, limit uint) ([]InsuranceBalance, error) {
	var balances []InsuranceBalance
	params := url.Values{}
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	path := common.EncodeURLValues(fmt.Sprintf(deliveryInsuranceBalance, settle), params)
	return balances, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &balances)
}

// GetDeliveryFuturesAccounts retrives futures account
func (g *Gateio) GetDeliveryFuturesAccounts(ctx context.Context, settle string) (*FuturesAccount, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	path := fmt.Sprintf(deliveryFuturesAccounts, settle)
	var response FuturesAccount
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &response)
}

// GetDeliveryAccountBooks retrives account books
func (g *Gateio) GetDeliveryAccountBooks(ctx context.Context, settle string, limit int, from, to time.Time, changingType string) ([]AccountBookItem, error) {
	params := url.Values{}
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if changingType == "dnw" ||
		changingType == "pnl" ||
		changingType == "fee" ||
		changingType == "refr" ||
		changingType == "fund" ||
		changingType == "point_dnw" ||
		changingType == "point_fee" ||
		changingType == "point_refr" {
		params.Set("type", changingType)
	}
	path := fmt.Sprintf(deliveryAccountBook, settle)
	var response []AccountBookItem
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, nil, &response)
}

// GetAllDeliveryPositionsOfUser retrives all positions of user
func (g *Gateio) GetAllDeliveryPositionsOfUser(ctx context.Context, settle string) (*Position, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	path := fmt.Sprintf(deliveryPositions, settle)
	var response Position
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &response)
}

// GetSingleDeliveryPosition get single position
func (g *Gateio) GetSingleDeliveryPosition(ctx context.Context, settle string, contract currency.Pair) (*Position, error) {
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract.Base.IsEmpty() || contract.Quote.IsEmpty() {
		return nil, fmt.Errorf("%v, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	contract.Delimiter = currency.UnderscoreDelimiter
	path := fmt.Sprintf(deliverySinglePosition, settle, contract.String())
	var response Position
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &response)
}

// UpdateDeliveryPositionMargin updates position margin
func (g *Gateio) UpdateDeliveryPositionMargin(ctx context.Context, settle string, change float64, contract currency.Pair) (*Position, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract.Base.IsEmpty() || contract.Quote.IsEmpty() {
		return nil, fmt.Errorf("%v, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	if change <= 0 {
		return nil, fmt.Errorf("%v, futures margin change must be positive", errChangehasToBePositive)
	}
	params := url.Values{}
	params.Set("change", strconv.FormatFloat(change, 'f', -1, 64))
	contract.Delimiter = currency.UnderscoreDelimiter
	path := fmt.Sprintf(deliveryUpdatePositionMargin, settle, contract.String())
	var response Position
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, params, nil, &response)
}

// UpdateDeliveryPositionLeverage updates position leverage
func (g *Gateio) UpdateDeliveryPositionLeverage(ctx context.Context, settle string, contract currency.Pair, leverage float64) (*Position, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract.Base.IsEmpty() || contract.Quote.IsEmpty() {
		return nil, fmt.Errorf("%v, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	contract.Delimiter = currency.UnderscoreDelimiter
	params := url.Values{}
	if leverage < 0 {
		return nil, errInvalidLeverageValue
	}
	params.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	path := fmt.Sprintf(deliveryPositionsLeverage, settle, contract.String())
	var response Position
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, params, nil, &response)
}

// UpdateDeliveryPositionRiskLimit update position risk limit
func (g *Gateio) UpdateDeliveryPositionRiskLimit(ctx context.Context, settle string, contract currency.Pair, riskLimit int) (*Position, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract.Base.IsEmpty() || contract.Quote.IsEmpty() {
		return nil, fmt.Errorf("%v, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	contract.Delimiter = currency.UnderscoreDelimiter
	params := url.Values{}
	if riskLimit < 0 {
		return nil, errInvalidRiskLimit
	}
	params.Set("risk_limit", strconv.Itoa(riskLimit))
	path := fmt.Sprintf(deliveryPositionRiskLimit, settle, contract.String())
	var response Position
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, params, nil, &response)
}

// CreateDeliveryOrder create a futures order
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (g *Gateio) CreateDeliveryOrder(ctx context.Context, arg OrderCreateParams) (*Order, error) {
	if arg.Contract.Base.IsEmpty() || arg.Contract.Quote.IsEmpty() {
		return nil, fmt.Errorf("%v, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	if arg.Size == 0 {
		return nil, fmt.Errorf("%v, specify positive number to make a bid, and negative number to ask", errInvalidOrderSide)
	}
	if !(arg.TimeInForce == "gtc" || arg.TimeInForce == "ioc" || arg.TimeInForce == "poc" || arg.TimeInForce == "fok") {
		return nil, errInvalidTimeInForce
	}
	if arg.Price > 0 && arg.TimeInForce == "ioc" {
		arg.Price = 0
	}
	if arg.Price < 0 {
		return nil, errInvalidPrice
	}
	if arg.Text != "" && !strings.HasPrefix(arg.Text, "t-") {
		arg.Text = "t-" + arg.Text
	}
	if arg.AutoSize != "" && (arg.AutoSize == "close_long" || arg.AutoSize == "close_short") {
		return nil, errInvalidAutoSizeValue
	}
	if !(arg.Settle == "btc" || arg.Settle == "usd" || arg.Settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	path := fmt.Sprintf(deliveryOrders, arg.Settle)
	var response Order
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, &arg, &response)
}

// GetDeliveryOrders list futures orders
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (g *Gateio) GetDeliveryOrders(ctx context.Context, contract currency.Pair, status string, limit, offset int, lastID string, countTotal int, settle string) ([]Order, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract.Base.IsEmpty() || contract.Quote.IsEmpty() {
		return nil, fmt.Errorf("%v, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	params := url.Values{}
	contract.Delimiter = currency.UnderscoreDelimiter
	params.Set("contract", contract.String())
	if !(status == "open" || status == "finished") {
		return nil, fmt.Errorf("%v, only 'open' and 'finished' status are supported", errInvalidOrderStatus)
	}
	params.Set("status", status)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	if countTotal == 1 && status != "open" {
		params.Set("count_total", strconv.Itoa(countTotal))
	} else if countTotal != 0 && countTotal != 1 {
		return nil, errInvalidCountTotalValue
	}
	path := fmt.Sprintf(deliveryOrders, settle)
	var response []Order
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, nil, &response)
}

// CancelAllDeliveryOrders cancel all open orders matched
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (g *Gateio) CancelAllDeliveryOrders(ctx context.Context, contract currency.Pair, side, settle string) ([]Order, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract.Base.IsEmpty() || contract.Quote.IsEmpty() {
		return nil, fmt.Errorf("%v, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	params := url.Values{}
	contract.Delimiter = currency.UnderscoreDelimiter
	if side == "ask" || side == "bid" {
		params.Set("side", side)
	}
	params.Set("contract", contract.String())
	path := fmt.Sprintf(deliveryOrders, settle)
	var response []Order
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, params, nil, &response)
}

// GetSingleDeliveryOrder Get a single order
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (g *Gateio) GetSingleDeliveryOrder(ctx context.Context, settle, orderID string) (*Order, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if orderID == "" {
		return nil, fmt.Errorf("%v, 'order_id' cannot be empty", errInvalidOrderID)
	}
	path := fmt.Sprintf(deliveryOrders+"/%s", settle, orderID)
	var response Order
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &response)
}

// CancelSingleDeliveryOrder cancel a single order
func (g *Gateio) CancelSingleDeliveryOrder(ctx context.Context, settle, orderID string) (*Order, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if orderID == "" {
		return nil, fmt.Errorf("%v, 'order_id' cannot be empty", errInvalidOrderID)
	}
	path := fmt.Sprintf(deliveryOrders+"/%s", settle, orderID)
	var response Order
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, nil, nil, &response)
}

// GetDeliveryPersonalTradingHistory retrives personal trading history
func (g *Gateio) GetDeliveryPersonalTradingHistory(ctx context.Context, settle string, contract currency.Pair, orderID string, limit, offset, countTotal int, lastID string) ([]TradingHistoryItem, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	params := url.Values{}
	if !(contract.Base.IsEmpty() || contract.Quote.IsEmpty()) {
		contract.Delimiter = currency.UnderscoreDelimiter
		params.Set("contract", contract.String())
	}
	if orderID != "" {
		params.Set("order", orderID)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	if countTotal == 1 {
		params.Set("count_total", strconv.Itoa(countTotal))
	}
	path := fmt.Sprintf(deliveryPersonalTradingHistory, settle)
	var response []TradingHistoryItem
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, nil, &response)
}

// GetDeliveryPositionCloseHistory retrives position history
func (g *Gateio) GetDeliveryPositionCloseHistory(ctx context.Context, settle string, contract currency.Pair, limit, offset int, from, to time.Time) ([]PositionCloseHistoryResponse, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	params := url.Values{}
	if !(contract.Base.IsEmpty() || contract.Quote.IsEmpty()) {
		contract.Delimiter = currency.UnderscoreDelimiter
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	path := fmt.Sprintf(deliveryPositionClose, settle)
	var response []PositionCloseHistoryResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, nil, &response)
}

// GetDeliveryLiquidationHistory lists liquidation history
func (g *Gateio) GetDeliveryLiquidationHistory(ctx context.Context, settle string, contract currency.Pair, limit int, at time.Time) ([]LiquidationHistoryItem, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	params := url.Values{}
	if !(contract.Base.IsEmpty() || contract.Quote.IsEmpty()) {
		contract.Delimiter = currency.UnderscoreDelimiter
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if !at.IsZero() {
		params.Set("at", strconv.FormatInt(at.Unix(), 10))
	}
	path := fmt.Sprintf(deliveryLiquidations, settle)
	var response []LiquidationHistoryItem
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, nil, &response)
}

// GetDeliverySettlementHistory retrives settlement history
func (g *Gateio) GetDeliverySettlementHistory(ctx context.Context, settle string, contract currency.Pair, limit int, at time.Time) ([]SettlementHistoryItem, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	params := url.Values{}
	if !(contract.Base.IsEmpty() || contract.Quote.IsEmpty()) {
		contract.Delimiter = currency.UnderscoreDelimiter
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if !at.IsZero() {
		params.Set("at", strconv.FormatInt(at.Unix(), 10))
	}
	var response []SettlementHistoryItem
	path := fmt.Sprintf(deliverySettlements, settle)
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, nil, &response)
}

// GetDeliveryPriceTriggeredOrder creates a price-triggered order
func (g *Gateio) GetDeliveryPriceTriggeredOrder(ctx context.Context, settle string, arg FuturesPriceTriggeredOrderParam) (*OrderID, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if arg.Initial.Contract.Base.IsEmpty() || arg.Initial.Contract.Quote.IsEmpty() {
		return nil, fmt.Errorf("%v, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	arg.Initial.Contract.Delimiter = currency.UnderscoreDelimiter
	if arg.Initial.Price < 0 {
		return nil, fmt.Errorf("%v, price must be greater than 0", errInvalidPrice)
	}
	if arg.Initial.Size <= 0 {
		return nil, errors.New("invalid argument: initial.size out of range")
	}
	if arg.Initial.TimeInForce != "" && !(arg.Initial.TimeInForce == "gtc" || arg.Initial.TimeInForce == "ioc") {
		return nil, fmt.Errorf("%v, only time in force value 'gtc' and 'ioc' are supported", errInvalidTimeInForce)
	}
	if !(arg.Trigger.StrategyType == 0 || arg.Trigger.StrategyType == 1) {
		return nil, fmt.Errorf("strategy type must be 0 or 1, 0: by price, and 1: by price gap")
	}
	if !(arg.Trigger.Rule == 1 || arg.Trigger.Rule == 2) {
		return nil, fmt.Errorf("invalid trigger condition('rule') value, rule must be 1 or 2")
	}
	if !(arg.Trigger.PriceType == 0 || arg.Trigger.PriceType == 1 || arg.Trigger.PriceType == 2) {
		return nil, fmt.Errorf("price type must be 0 or 1 or 2")
	}
	if arg.Trigger.Price <= 0 {
		return nil, errors.New("invalid argument: trigger.price")
	}
	if arg.Trigger.OrderType != "" &&
		!(arg.Trigger.OrderType == "close-long-order" ||
			arg.Trigger.OrderType == "close-short-order" ||
			arg.Trigger.OrderType == "close-long-position" ||
			arg.Trigger.OrderType == "close-short-position" ||
			arg.Trigger.OrderType == "plan-close-long-position" ||
			arg.Trigger.OrderType == "plan-close-short-position") {
		return nil, errors.New("invalid order type, only 'close-long-order', 'close-short-order', 'close-long-position', 'close-short-position', 'plan-close-long-position', and 'plan-close-short-position'")
	}
	var response OrderID
	path := fmt.Sprintf(deliveryPriceOrders, settle)
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, &arg, &response)
}

// GetDeliveryAllAutoOrder retrives all auto orders
func (g *Gateio) GetDeliveryAllAutoOrder(ctx context.Context, status, settle string, contract currency.Pair, limit, offset int) ([]PriceTriggeredOrder, error) {
	if !(status == "open" || status == "finished") {
		return nil, errInvalidOrderStatus
	}
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	path := fmt.Sprintf(deliveryPriceOrders, settle)
	params := url.Values{}
	params.Set("status", status)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if !(contract.Base.IsEmpty() || contract.Quote.IsEmpty()) {
		contract.Delimiter = currency.UnderscoreDelimiter
		params.Set("contract", contract.String())
	}
	var response []PriceTriggeredOrder
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, nil, &response)
}

// CancelAllDeliveryPriceTriggeredOrder cancels all delivery price triggered orders
func (g *Gateio) CancelAllDeliveryPriceTriggeredOrder(ctx context.Context, settle string, contract currency.Pair) ([]PriceTriggeredOrder, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract.Base.IsEmpty() || contract.Quote.IsEmpty() {
		return nil, fmt.Errorf("%v, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	params := url.Values{}
	contract.Delimiter = currency.UnderscoreDelimiter
	path := fmt.Sprintf(deliveryPriceOrders, settle)
	params.Set("contract", contract.String())
	var response []PriceTriggeredOrder
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, params, nil, &response)
}

// GetSingleDeliveryPriceTriggeredOrder retrives a single price triggered order
func (g *Gateio) GetSingleDeliveryPriceTriggeredOrder(ctx context.Context, settle, orderID string) (*PriceTriggeredOrder, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if orderID == "" {
		return nil, errInvalidOrderID
	}
	path := fmt.Sprintf(deliveryPriceOrders+"/%s", settle, orderID)
	var response PriceTriggeredOrder
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &response)
}

// CancelDeliveryPriceTriggeredOrder cancel a price-triggered order
func (g *Gateio) CancelDeliveryPriceTriggeredOrder(ctx context.Context, settle, orderID string) (*PriceTriggeredOrder, error) {
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if orderID == "" {
		return nil, errInvalidOrderID
	}
	path := fmt.Sprintf(deliveryPriceOrders+"/%s", settle, orderID)
	var response PriceTriggeredOrder
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, nil, nil, &response)
}

// ********************************** Options ***************************************************

// GetAllUnderlyings retrives all option underlyings
func (g *Gateio) GetAllUnderlyings(ctx context.Context) ([]OptionUnderlying, error) {
	var underlyings []OptionUnderlying
	return underlyings, g.SendHTTPRequest(ctx, exchange.RestSpot, optionUnderlyings, &underlyings)
}

// GetExpirationTime return the expiration time for the provided underlying.
func (g *Gateio) GetExpirationTime(ctx context.Context, underlying string) (time.Time, error) {
	var timestamp []float64
	path := optionExpiration + "?underlying=" + underlying
	er := g.SendHTTPRequest(ctx, exchange.RestSpot, path, &timestamp)
	if er != nil {
		return time.Time{}, er
	}
	if len(timestamp) == 0 {
		return time.Time{}, errNoValidResponseFromServer
	}
	return time.Unix(int64(timestamp[0]), 0), nil
}

// GetAllContractOfUnderlyingWithinExpiryDate retrives list of contracts of the specified underlying and expiry time.
func (g *Gateio) GetAllContractOfUnderlyingWithinExpiryDate(ctx context.Context, underlying string, expTime time.Time) ([]OptionContract, error) {
	var contracts []OptionContract
	params := url.Values{}
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params.Set("underlying", underlying)
	if !expTime.IsZero() {
		params.Set("expires", strconv.FormatInt(expTime.Unix(), 10))
	}
	path := common.EncodeURLValues(optionContracts, params)
	return contracts, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &contracts)
}

// GetSpecifiedContractDetail query specified contract detail
func (g *Gateio) GetSpecifiedContractDetail(ctx context.Context, contract string) (*OptionContract, error) {
	var contr OptionContract
	if contract == "" {
		return nil, errInvalidOrMissingContractParam
	}
	path := fmt.Sprintf(optionContracts+"/%s", contract)
	return &contr, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &contr)
}

// GetSettlementHistory retrives list of settlement history
func (g *Gateio) GetSettlementHistory(ctx context.Context, underlying string, offset, limit uint, from, to time.Time) ([]OptionSettlement, error) {
	var settlements []OptionSettlement
	params := url.Values{}
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params.Set("underlying", underlying)
	if offset > 0 {
		params.Set("offset", strconv.Itoa(int(offset)))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(int(limit)))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	path := common.EncodeURLValues(optionSettlement, params)
	return settlements, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &settlements)
}

// GetSpecifiedSettlementHistory retrive a single contract settlement detail passing the underlying and contract name
func (g *Gateio) GetSpecifiedSettlementHistory(ctx context.Context, contract, underlying string, at uint) (*OptionSettlement, error) {
	var settlement OptionSettlement
	params := url.Values{}
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	if contract == "" {
		return nil, errInvalidOrMissingContractParam
	}
	params.Set("underlying", underlying)
	params.Set("at", strconv.Itoa(int(at)))
	path := common.EncodeURLValues(fmt.Sprintf(optionSettlement+"/%s", contract), params)
	return &settlement, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &settlement)
}

// GetMyOptionsSettlements retrives accounts option settlements.
func (g *Gateio) GetMyOptionsSettlements(ctx context.Context, underlying, contract string, offset, limit uint, to time.Time) ([]MyOptionSettlement, error) {
	var settlements []MyOptionSettlement
	params := url.Values{}
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params.Set("underlying", underlying)
	if contract != "" {
		params.Set("contract", contract)
	}
	if to.After(time.Now()) {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(int(offset)))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(int(limit)))
	}
	return settlements, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, optionMySettlements, params, nil, &settlements)
}

// GetOptionsOrderbook returns the orderbook data for the given contract.
func (g *Gateio) GetOptionsOrderbook(ctx context.Context, contract currency.Pair, interval string, limit uint, withOrderbookID bool) (*Orderbook, error) {
	var response Orderbook
	if contract.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	params := url.Values{}
	fPair, er := g.GetPairFormat(asset.Spot, true)
	if er != nil {
		fPair = currency.PairFormat{
			Delimiter: currency.UnderscoreDelimiter,
			Uppercase: true,
		}
	}
	pairString := strings.ToUpper(fPair.Format(contract))
	params.Set("contract", pairString)
	if interval == OrderbookIntervalZero || interval == OrderbookIntervalZeroPt1 || interval == OrderbookIntervalZeroPtZero1 {
		params.Set("interval", interval)
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	params.Set("with_id", strconv.FormatBool(withOrderbookID))
	return &response, g.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(optionsOrderbook, params), &response)
}

// GetOptionAccounts lists option accounts
func (g *Gateio) GetOptionAccounts(ctx context.Context) (*OptionAccount, error) {
	var resp OptionAccount
	return &resp, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, optionAccounts, nil, nil, &resp)
}

// GetAccountChangingHistory retrives list of account changing history
func (g *Gateio) GetAccountChangingHistory(ctx context.Context, offset, limit int, from, to time.Time, changingType string) ([]AccountBook, error) {
	params := url.Values{}
	var accountBook []AccountBook
	if changingType == "dnw" || changingType == "prem" ||
		changingType == "fee" || changingType == "refr" ||
		changingType == "set" {
		params.Set("type", changingType)
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() && ((!from.IsZero() && to.After(from)) || to.Before(time.Now())) {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	return accountBook, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, optionsAccountbook, params, nil, &accountBook)
}

// GetUsersPositionSpecifiedUnderlying lists user's positions of specified underlying
func (g *Gateio) GetUsersPositionSpecifiedUnderlying(ctx context.Context, underlying string) ([]UsersPositionForUnderlying, error) {
	var response []UsersPositionForUnderlying
	params := url.Values{}
	if underlying != "" {
		params.Set("underlying", underlying)
	}
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, optionsPosition, params, nil, &response)
}

// GetSpecifiedContractPosition retrives specified contract position
func (g *Gateio) GetSpecifiedContractPosition(ctx context.Context, contract string) (*UsersPositionForUnderlying, error) {
	var response UsersPositionForUnderlying
	if contract == "" {
		return nil, errInvalidOrMissingContractParam
	}
	path := fmt.Sprintf("%s/%s", optionsPosition, contract)
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &response)
}

// GetUsersLiquidationHistoryForSpecifiedUnderlying retrives user's liquidation history of specified underlying
func (g *Gateio) GetUsersLiquidationHistoryForSpecifiedUnderlying(ctx context.Context, underlying, contract string) ([]ContractClosePosition, error) {
	params := url.Values{}
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params.Set("underlying", underlying)
	if contract != "" {
		params.Set("contract", contract)
	}
	var response []ContractClosePosition
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, optionsPositionClose, params, nil, &response)
}

// PlaceOptionOrder creates an options order
func (g *Gateio) PlaceOptionOrder(ctx context.Context, arg OptionOrderParam) (*OptionOrderResponse, error) {
	var response OptionOrderResponse
	if arg.Contract == "" {
		return nil, errInvalidOrMissingContractParam
	}
	if arg.OrderSize == 0 {
		return nil, errInvalidOrderSize
	}
	if arg.Iceberg < 0 {
		arg.Iceberg = 0
	}
	if !(arg.TimeInForce == "gtc" || arg.TimeInForce == "ioc" || arg.TimeInForce == "poc") {
		arg.TimeInForce = ""
	}
	if arg.TimeInForce == "ioc" || arg.Price < 0 {
		arg.Price = 0
	}
	if arg.Close {
		arg.OrderSize = 0
	}
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost,
		optionsOrders, nil, &arg, &response)
}

// GetOptionFuturesOrders retrives futures orders
func (g *Gateio) GetOptionFuturesOrders(ctx context.Context, contract, underlying, status string, offset, limit int, from, to time.Time) ([]OptionOrderResponse, error) {
	var response []OptionOrderResponse
	params := url.Values{}
	if contract != "" {
		params.Set("contract", contract)
	}
	if underlying != "" {
		params.Set("underlying", underlying)
	}
	status = strings.ToLower(status)
	if status == "open" || status == "finished" {
		params.Set("status", status)
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() && ((!from.IsZero() && to.After(from)) || to.Before(time.Now())) {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpotSupplementary,
		http.MethodGet, optionsOrders, params, nil, &response)
}

// CancelAllOptionOpenOrders cancels all open orders matched
func (g *Gateio) CancelAllOptionOpenOrders(ctx context.Context, contract currency.Pair, underlying, side string) ([]OptionOrderResponse, error) {
	params := url.Values{}
	if !(contract.Base.IsEmpty() || contract.Quote.IsEmpty()) {
		contract.Delimiter = currency.UnderscoreDelimiter
		params.Set("contract", contract.String())
	}
	if underlying != "" {
		params.Set("underlying", underlying)
	}
	if side == "ask" || side == "bid" {
		params.Set("side", side)
	}
	var response []OptionOrderResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpotSupplementary, http.MethodDelete, optionsOrders, params, nil, &response)
}

// GetSingleOptionOrder retrives a single option order
func (g *Gateio) GetSingleOptionOrder(ctx context.Context, orderID string) (*OptionOrderResponse, error) {
	var order OptionOrderResponse
	if orderID == "" {
		return nil, errInvalidOrderID
	}
	path := fmt.Sprintf("%s/%s", optionsOrders, orderID)
	return &order, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, path, nil, nil, &order)
}

// CancelSingleOrder cancel a single order.
func (g *Gateio) CancelOptionSingleOrder(ctx context.Context, orderID string) (*OptionOrderResponse, error) {
	var response OptionOrderResponse
	if orderID == "" {
		return nil, errInvalidOrderID
	}
	path := fmt.Sprintf("%s/%s", optionsOrders, orderID)
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodDelete, path, nil, nil, &response)
}

// GetOptionsPersonalTradingHistory retrives personal tradign histories given the underlying{Required}, contract, and other pagination params.
func (g *Gateio) GetOptionsPersonalTradingHistory(ctx context.Context, underlying, contract string, offset, limit int, from, to time.Time) ([]OptionTradingHistory, error) {
	var resp []OptionTradingHistory
	params := url.Values{}
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params.Set("underlying", underlying)
	if contract != "" {
		params.Set("contract", contract)
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() && ((!from.IsZero() && to.After(from)) || to.Before(time.Now())) {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	return resp, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, optionsMyTrades, params, nil, &resp)
}

// GetOptionsTickers lists  tickers of options contracts
func (g *Gateio) GetOptionsTickers(ctx context.Context, underlying string) ([]OptionsTicker, error) {
	var response []OptionsTicker
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	underlying = strings.ToUpper(underlying)
	path := optionsTickers + "?underlying=" + underlying
	return response, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &response)
}

// GetOptionUnderlyingTickers retrives options underlying ticker
func (g *Gateio) GetOptionUnderlyingTickers(ctx context.Context, underlying string) (*OptionsUnderlyingTicker, error) {
	var respos OptionsUnderlyingTicker
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	path := fmt.Sprintf(optionsUnderlyingTickers, underlying)
	println(path)
	return &respos, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &respos)
}

// GetOptionFuturesCandlesticks retrives option futures candlesticks
func (g *Gateio) GetOptionFuturesCandlesticks(ctx context.Context, contract string, limit int, from, to time.Time, interval kline.Interval) ([]FuturesCandlestick, error) {
	var candles []FuturesCandlestick
	params := url.Values{}
	if contract == "" {
		return nil, errInvalidOrMissingContractParam
	}
	params.Set("contract", contract)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if intervalString := g.GetIntervalString(interval); intervalString != "" {
		params.Set("interval", intervalString)
	}
	path := common.EncodeURLValues(optionCandlesticks, params)
	return candles, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &candles)
}

// GetOptionFuturesMarkPriceCandlesticks retrives mark price candlesticks of an underlying
func (g *Gateio) GetOptionFuturesMarkPriceCandlesticks(ctx context.Context, underlying string, limit int, from, to time.Time, interval kline.Interval) ([]FuturesCandlestick, error) {
	var candles []FuturesCandlestick
	params := url.Values{}
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params.Set("underlying", underlying)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if intervalString := g.GetIntervalString(interval); intervalString != "" {
		params.Set("interval", intervalString)
	}
	path := common.EncodeURLValues(optionUnderlyingCandlesticks, params)
	return candles, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &candles)
}

// GetOptionsTradeHistory retrives options trade history
func (g *Gateio) GetOptionsTradeHistory(ctx context.Context, contract /*C is call, while P is put*/, callType string,
	offset, limit int, from, to time.Time) ([]TradingHistoryItem, error) {
	var trades []TradingHistoryItem
	params := url.Values{}
	callType = strings.ToUpper(callType)
	if callType == "C" || callType == "P" {
		params.Set("type", callType)
	}
	if contract != "" {
		params.Set("contract", contract)
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	return trades, g.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(optionsTrades, params), &trades)
}

// ********************************** Flash_SWAP *************************

// GetSupportedFlashSwapCurrencies retrives all supported currencies in flash swap
func (g *Gateio) GetSupportedFlashSwapCurrencies(ctx context.Context) ([]SwapCurrencies, error) {
	var currencies []SwapCurrencies
	return currencies, g.SendHTTPRequest(ctx, exchange.RestSpot, flashSwapCurrencies, &currencies)
}

// CreateFlashSwapOrder creates a new flash swap order
// initiate a flash swap preview in advance because order creation requires a preview result
func (g *Gateio) CreateFlashSwapOrder(ctx context.Context, arg FlashSwapOrderParams) (*FlashSwapOrderResponse, error) {
	if arg.PreviewID == "" {
		return nil, errMissingPreviewID
	}
	if arg.BuyCurrency.IsEmpty() {
		return nil, fmt.Errorf("%v, buy currency can not empty", errInvalidCurrency)
	}
	if arg.SellCurrency.IsEmpty() {
		return nil, fmt.Errorf("%v, sell currency can not empty", errInvalidCurrency)
	}
	if arg.SellAmount <= 0 {
		return nil, fmt.Errorf("%v, sell_amount can not be less than or equal to 0", errInvalidAmount)
	}
	if arg.BuyAmount <= 0 {
		return nil, fmt.Errorf("%v, buy_amount amount can not be less than or equal to 0", errInvalidAmount)
	}
	var response FlashSwapOrderResponse
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, flashSwapOrders, nil, &arg, &response)
}

// GetAllFlashSwapOrders retrives list of flash swap orders filtered by the params
func (g *Gateio) GetAllFlashSwapOrders(ctx context.Context, status int, sellCurrency, buyCurrency currency.Code, reverse bool, limit, page int) ([]FlashSwapOrderResponse, error) {
	params := url.Values{}
	if status == 1 || status == 2 {
		params.Set("status", strconv.Itoa(status))
	}
	if !sellCurrency.IsEmpty() {
		params.Set("sell_currency", sellCurrency.String())
	}
	if !buyCurrency.IsEmpty() {
		params.Set("buy_currency", buyCurrency.String())
	}
	params.Set("reverse", strconv.FormatBool(reverse))
	if page > 0 {
		params.Set("page", strconv.Itoa(page))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	var response []FlashSwapOrderResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, flashSwapOrders, params, nil, &response)
}

// GetSingleFlashSwapOrder get a single flash swap order's detail
func (g *Gateio) GetSingleFlashSwapOrder(ctx context.Context, orderID string) (*FlashSwapOrderResponse, error) {
	if orderID == "" {
		return nil, fmt.Errorf("%v, flash order order_id must not be empty", errInvalidOrderID)
	}
	path := fmt.Sprintf("%s/%s", flashSwapOrders, orderID)
	var response FlashSwapOrderResponse
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &response)
}

// InitiateFlashSwapOrderReview initiate a flash swap order preview
func (g *Gateio) InitiateFlashSwapOrderReview(ctx context.Context, arg FlashSwapOrderParams) (*InitFlashSwapOrderPreviewResponse, error) {
	if arg.PreviewID == "" {
		return nil, errMissingPreviewID
	}
	if arg.BuyCurrency.IsEmpty() {
		return nil, fmt.Errorf("%v, buy currency can not empty", errInvalidCurrency)
	}
	if arg.SellCurrency.IsEmpty() {
		return nil, fmt.Errorf("%v, sell currency can not empty", errInvalidCurrency)
	}
	if !((arg.SellAmount >= 1 && arg.SellAmount <= 10000) || arg.SellAmount == 0) {
		return nil, fmt.Errorf("%v, sell_amount must greater than 1 less than 10000", errInvalidAmount)
	}
	if !((arg.BuyAmount >= 0.0001 && arg.BuyAmount <= 0.1) || arg.BuyAmount == 0) {
		return nil, fmt.Errorf("%v, buy_amount must greater than 0.0001 less than 0.1", errInvalidAmount)
	}
	var response InitFlashSwapOrderPreviewResponse
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, flashSwapOrdersPreview, nil, &arg, &response)
}

// IsValidPairString returns true if the string represents a valid currency pair
func (g *Gateio) IsValidPairString(currencyPair string) bool {
	if len(currencyPair) < 3 {
		return false
	}
	if strings.Contains(currencyPair, g.CurrencyPairs.RequestFormat.Delimiter) {
		result := strings.Split(currencyPair, g.CurrencyPairs.RequestFormat.Delimiter)
		if len(result) < 2 {
			return false
		}
		return true
	}
	return false
}

// GetContractFromCurrencyPair returns a contract string given a currency pair
func (g *Gateio) GetContractFromCurrencyPair(ctx context.Context, pair currency.Pair, a asset.Item) (string, error) {
	if pair.IsEmpty() {
		return "", errors.New("currency pair not supported")
	}
	pair.Delimiter = currency.UnderscoreDelimiter
	if a == asset.DeliveryFutures {
		if !(strings.EqualFold(pair.Quote.String(), currency.USD.String()) || strings.EqualFold(pair.Quote.String(), currency.USDT.String()) || strings.EqualFold(pair.Quote.String(), currency.BTC.String())) {
			return "", errUnsupportedSettleValue
		}
		contracts, err := g.GetAllDeliveryContracts(ctx, pair.Quote.String())
		if err != nil && !strings.Contains(err.Error(), "404 Not Found") {
			return "", err
		}
		for x := range contracts {
			if strings.HasPrefix(contracts[x].Name, strings.ToUpper(pair.String())) {
				return contracts[x].Name, nil
			}
		}
		return "", errors.New("no contract found")
	}
	return pair.String(), nil
}
