package gateio

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own APIKEYS here for due diligence testing

const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var g Gateio
var wsSetupRan bool

func TestMain(m *testing.M) {
	g.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("GateIO load config error", err)
	}
	gConf, err := cfg.GetExchangeConfig("GateIO")
	if err != nil {
		log.Fatal("GateIO Setup() init error")
	}
	gConf.API.AuthenticatedSupport = true
	gConf.API.AuthenticatedWebsocketSupport = true
	gConf.API.Credentials.Key = apiKey
	gConf.API.Credentials.Secret = apiSecret
	g.Websocket = sharedtestvalues.NewTestWebsocket()
	g.Verbose = true
	err = g.Setup(gConf)
	if err != nil {
		log.Fatal("GateIO setup error", err)
	}

	os.Exit(m.Run())
}

func TestStart(t *testing.T) {
	t.Parallel()
	err := g.Start(nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrNilPointer)
	}
	var testWg sync.WaitGroup
	err = g.Start(&testWg)
	if err != nil {
		t.Fatal(err)
	}
	testWg.Wait()
}

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	_, err := g.GetSymbols(context.Background())
	if err != nil {
		t.Errorf("Gateio TestGetSymbols: %s", err)
	}
}

func TestGetMarketInfo(t *testing.T) {
	t.Parallel()
	_, err := g.GetMarketInfo(context.Background())
	if err != nil {
		t.Errorf("Gateio GetMarketInfo: %s", err)
	}
}

func TestSpotNewOrder(t *testing.T) {
	t.Parallel()

	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip()
	}

	_, err := g.SpotNewOrder(context.Background(),
		SpotNewOrderRequestParams{
			Symbol: "btc_usdt",
			Amount: -1,
			Price:  100000,
			Type:   order.Sell.Lower(),
		})
	if err != nil {
		t.Errorf("Gateio SpotNewOrder: %s", err)
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()

	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip()
	}

	_, err := g.CancelExistingOrder(context.Background(), 917591554, "btc_usdt")
	if err != nil {
		t.Errorf("Gateio CancelExistingOrder: %s", err)
	}
}

func TestGetBalances(t *testing.T) {
	t.Parallel()

	if !areTestAPIKeysSet() {
		t.Skip()
	}

	_, err := g.GetBalances(context.Background())
	if err != nil {
		t.Errorf("Gateio GetBalances: %s", err)
	}
}

// func TestGetLatestSpotPrice(t *testing.T) {
// 	t.Parallel()
// 	_, err := g.GetLatestSpotPrice(context.Background(), "btc_usdt")
// 	if err != nil {
// 		t.Errorf("Gateio GetLatestSpotPrice: %s", err)
// 	}
// }

// func TestGetTicker(t *testing.T) {
// 	t.Parallel()
// 	_, err := g.GetTicker(context.Background(), "btc_usdt")
// 	if err != nil {
// 		t.Errorf("Gateio GetTicker: %s", err)
// 	}
// }

// func TestGetTickers(t *testing.T) {
// 	t.Parallel()
// 	_, err := g.GetTickers(context.Background())
// 	if err != nil {
// 		t.Errorf("Gateio GetTicker: %s", err)
// 	}
// }

func TestGetSpotKline(t *testing.T) {
	t.Parallel()
	_, err := g.GetSpotKline(context.Background(),
		KlinesRequestParams{
			Symbol:   "btc_usdt",
			GroupSec: "5", // 5 minutes or less
			HourSize: 1,   // 1 hour data
		})

	if err != nil {
		t.Errorf("Gateio GetSpotKline: %s", err)
	}
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:  1,
		FeeType: exchange.CryptocurrencyTradeFee,
		Pair: currency.NewPairWithDelimiter(currency.BTC.String(),
			currency.USDT.String(), "_"),
		IsMaker:             false,
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

func TestGetTradeHistory(t *testing.T) {
	_, err := g.GetTrades(context.Background(),
		currency.NewPairWithDelimiter(currency.BTC.String(),
			currency.USDT.String(), "_").String())
	if err != nil {
		t.Error(err)
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	var feeBuilder = setFeeBuilder()
	_, err := g.GetFeeByType(context.Background(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
	if !areTestAPIKeysSet() {
		if feeBuilder.FeeType != exchange.OfflineTradeFee {
			t.Errorf("Expected %v, received %v", exchange.OfflineTradeFee, feeBuilder.FeeType)
		}
	} else {
		if feeBuilder.FeeType != exchange.CryptocurrencyTradeFee {
			t.Errorf("Expected %v, received %v", exchange.CryptocurrencyTradeFee, feeBuilder.FeeType)
		}
	}
}

func TestGetFee(t *testing.T) {
	var feeBuilder = setFeeBuilder()
	if areTestAPIKeysSet() {
		// CryptocurrencyTradeFee Basic
		if _, err := g.GetFee(context.Background(), feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if _, err := g.GetFee(context.Background(), feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if _, err := g.GetFee(context.Background(), feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if _, err := g.GetFee(context.Background(), feeBuilder); err != nil {
			t.Error(err)
		}
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := g.GetFee(context.Background(), feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := g.GetFee(context.Background(), feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	if _, err := g.GetFee(context.Background(), feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if _, err := g.GetFee(context.Background(), feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if _, err := g.GetFee(context.Background(), feeBuilder); err != nil {
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	expectedResult := exchange.AutoWithdrawCryptoText + " & " + exchange.NoFiatWithdrawalsText
	withdrawPermissions := g.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	var getOrdersRequest = order.GetOrdersRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
	}

	_, err := g.GetActiveOrders(context.Background(), &getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	var getOrdersRequest = order.GetOrdersRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
	}

	currPair := currency.NewPair(currency.LTC, currency.BTC)
	currPair.Delimiter = "_"
	getOrdersRequest.Pairs = []currency.Pair{currPair}

	_, err := g.GetOrderHistory(context.Background(), &getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func areTestAPIKeysSet() bool {
	return g.ValidateAPICredentials(g.GetDefaultCredentials()) == nil
}

func TestSubmitOrder(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip()
	}

	var orderSubmission = &order.Submit{
		Exchange: g.Name,
		Pair: currency.Pair{
			Delimiter: "_",
			Base:      currency.LTC,
			Quote:     currency.BTC,
		},
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  "meowOrder",
		AssetType: asset.Spot,
	}
	response, err := g.SubmitOrder(context.Background(), orderSubmission)
	if areTestAPIKeysSet() && (err != nil || response.Status != order.New) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip()
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
		AssetType:     asset.Spot,
	}

	err := g.CancelOrder(context.Background(), orderCancellation)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip()
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
		AssetType:     asset.Spot,
	}

	resp, err := g.CancelAllOrders(context.Background(), orderCancellation)

	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}

	if len(resp.Status) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.Status))
	}
}

func TestGetAccountInfo(t *testing.T) {
	if apiSecret == "" || apiKey == "" {
		_, err := g.UpdateAccountInfo(context.Background(), asset.Spot)
		if err == nil {
			t.Error("GetAccountInfo() Expected error")
		}
	} else {
		_, err := g.UpdateAccountInfo(context.Background(), asset.Spot)
		if err != nil {
			t.Error("GetAccountInfo() error", err)
		}
	}
}

func TestModifyOrder(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := g.ModifyOrder(context.Background(),
		&order.Modify{AssetType: asset.Spot})
	if err == nil {
		t.Error("ModifyOrder() Expected error")
	}
}

func TestWithdraw(t *testing.T) {
	withdrawCryptoRequest := withdraw.Request{
		Exchange:    g.Name,
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	_, err := g.WithdrawCryptocurrencyFunds(context.Background(),
		&withdrawCryptoRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = withdraw.Request{}
	_, err := g.WithdrawFiatFunds(context.Background(), &withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = withdraw.Request{}
	_, err := g.WithdrawFiatFundsToInternationalBank(context.Background(),
		&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	if areTestAPIKeysSet() {
		_, err := g.GetDepositAddress(context.Background(), currency.USDT, "", "TRX")
		if err != nil {
			t.Error("Test Fail - GetDepositAddress error", err)
		}
	} else {
		_, err := g.GetDepositAddress(context.Background(), currency.ETC, "", "")
		if err == nil {
			t.Error("Test Fail - GetDepositAddress error cannot be nil")
		}
	}
}
func TestGetOrderInfo(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip("no API keys set skipping test")
	}

	_, err := g.GetOrderInfo(context.Background(),
		"917591554", currency.EMPTYPAIR, asset.Spot)
	if err != nil {
		if err.Error() != "no order found with id 917591554" && err.Error() != "failed to get open orders" {
			t.Fatalf("GetOrderInfo() returned an error skipping test: %v", err)
		}
	}
}

// TestWsGetBalance dials websocket, sends balance request.
func TestWsGetBalance(t *testing.T) {
	if !g.Websocket.IsEnabled() && !g.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := g.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	go g.wsReadData()
	err = g.wsServerSignIn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.wsGetBalance([]string{"EOS", "BTC"})
	if err != nil {
		t.Error(err)
	}
	_, err = g.wsGetBalance([]string{})
	if err != nil {
		t.Error(err)
	}
}

// TestWsGetOrderInfo dials websocket, sends order info request.
func TestWsGetOrderInfo(t *testing.T) {
	if !g.Websocket.IsEnabled() && !g.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := g.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	go g.wsReadData()
	err = g.wsServerSignIn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.wsGetOrderInfo("EOS_USDT", 0, 100)
	if err != nil {
		t.Error(err)
	}
}

func setupWSTestAuth(t *testing.T) {
	t.Helper()
	if wsSetupRan {
		return
	}
	if !g.Websocket.IsEnabled() && !g.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip(stream.WebsocketNotEnabled)
	}
	if err := g.Websocket.Connect(); err != nil {
		t.Fatal(err)
	}
	wsSetupRan = true
}

// TestWsSubscribe dials websocket, sends a subscribe request.
func TestWsSubscribe(t *testing.T) {
	setupWSTestAuth(t)
	err := g.Subscribe([]stream.ChannelSubscription{
		{
			Channel:  "ticker.subscribe",
			Currency: currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_"),
		},
	})
	if err != nil {
		t.Error(err)
	}

	err = g.Unsubscribe([]stream.ChannelSubscription{
		{
			Channel:  "ticker.subscribe",
			Currency: currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_"),
		},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestWsTicker(t *testing.T) {
	pressXToJSON := []byte(`{
    "method": "ticker.update",
    "params":
        [
            "BTC_USDT",
                {
                    "period": 86400,
                    "open": "0",
                    "close": "0",
                    "high": "0",
                    "low": "0",
                    "last": "0.2844",
                    "change": "0",
                    "quoteVolume": "0",
                    "baseVolume": "0"
                }
     ],
     "id": null
}`)
	err := g.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsTrade(t *testing.T) {
	pressXToJSON := []byte(`{
    "method": "trades.update",
    "params":
        [
             "BTC_USDT",
             [
                 {
                 "id": 7172173,
                 "time": 1523339279.761838,
                 "price": "398.59",
                 "amount": "0.027",
                 "type": "buy"
                 }
             ]
         ],
     "id": null
 }
`)
	err := g.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsDepth(t *testing.T) {
	pressXToJSON := []byte(`{
    "method": "depth.update",
    "params": [
        true,
        {
            "asks": [
                [
                    "8000.00",
                    "9.6250"
                ]
            ],
            "bids": [
                [
                    "8000.00",
                    "9.6250"
                ]
            ]
         },
         "BTC_USDT"
    ],
    "id": null
 }`)
	err := g.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsKLine(t *testing.T) {
	pressXToJSON := []byte(`{
    "method": "kline.update",
    "params":
        [
            [
                1492358400,
                "7000.00",
                "8000.0",
                "8100.00",
                "6800.00",
                "1000.00",
                "123456.00",
                "BTC_USDT"
            ]
        ],
    "id": null
}`)
	err := g.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsOrderUpdate(t *testing.T) {
	pressXToJSON := []byte(`{
  "method": "order.update",
  "params": [
    3,
    {
      "id": 34628963,
      "market": "BTC_USDT",
      "orderType": 1,
      "type": 2,
      "user": 602123,
      "ctime": 1523013969.6271579,
      "mtime": 1523013969.6271579,
      "price": "0.1",
      "amount": "1000",
      "left": "1000",
      "filledAmount": "0",
      "filledTotal": "0",
      "dealFee": "0"
    }
  ],
  "id": null
}`)
	err := g.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsBalanceUpdate(t *testing.T) {
	pressXToJSON := []byte(`{
    "method": "balance.update",
    "params": [{"EOS": {"available": "96.765323611874", "freeze": "11"}}],
    "id": 1234
}`)
	err := g.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestParseTime(t *testing.T) {
	// Test REST example
	r := convert.TimeFromUnixTimestampDecimal(1574846296.995313).UTC()
	if r.Year() != 2019 ||
		r.Month().String() != "November" ||
		r.Day() != 27 {
		t.Error("unexpected result")
	}

	// Test websocket example
	r = convert.TimeFromUnixTimestampDecimal(1523887354.256974).UTC()
	if r.Year() != 2018 ||
		r.Month().String() != "April" ||
		r.Day() != 16 {
		t.Error("unexpected result")
	}
}

func TestGetHistoricCandles(t *testing.T) {
	currencyPair, err := currency.NewPairFromString("BTC_USDT")
	if err != nil {
		t.Fatal(err)
	}
	startTime := time.Now().Add(-time.Hour * 6)
	_, err = g.GetHistoricCandles(context.Background(),
		currencyPair, asset.Spot, startTime, time.Now(), kline.OneMin)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	currencyPair, err := currency.NewPairFromString("BTC_USDT")
	if err != nil {
		t.Fatal(err)
	}
	startTime := time.Now().Add(-time.Minute * 2)
	_, err = g.GetHistoricCandlesExtended(context.Background(),
		currencyPair, asset.Spot, startTime, time.Now(), kline.OneMin)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_FormatExchangeKlineInterval(t *testing.T) {
	testCases := []struct {
		name     string
		interval kline.Interval
		output   string
	}{
		{
			"OneMin",
			kline.OneMin,
			"60",
		},
		{
			"OneDay",
			kline.OneDay,
			"86400",
		},
	}

	for x := range testCases {
		test := testCases[x]

		t.Run(test.name, func(t *testing.T) {
			ret := g.FormatExchangeKlineInterval(test.interval)

			if ret != test.output {
				t.Fatalf("unexpected result return expected: %v received: %v", test.output, ret)
			}
		})
	}
}

func TestGenerateDefaultSubscriptions(t *testing.T) {
	err := g.CurrencyPairs.EnablePair(asset.Spot, currency.NewPair(
		currency.LTC,
		currency.USDT,
	))
	if err != nil {
		t.Fatal(err)
	}
	subs, err := g.GenerateDefaultSubscriptions()
	if err != nil {
		t.Fatal(err)
	}

	payload, err := g.generatePayload(subs)
	if err != nil {
		t.Fatal(err)
	}

	if len(payload) != 4 {
		t.Fatal("unexpected payload length")
	}
}

// func TestGetRecentTrades(t *testing.T) {
// 	t.Parallel()
// 	currencyPair, err := currency.NewPairFromString("btc_usdt")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	_, err = g.GetRecentTrades(context.Background(), currencyPair, asset.Spot)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("btc_usdt")
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.GetHistoricTrades(context.Background(),
		currencyPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	if err != nil && err != common.ErrFunctionNotSupported {
		t.Error(err)
	}
}

// func TestUpdateTicker(t *testing.T) {
// 	t.Parallel()
// 	cp, err := currency.NewPairFromString("btc_usdt")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	_, err = g.UpdateTicker(context.Background(), cp, asset.Spot)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestUpdateTickers(t *testing.T) {
// 	err := g.UpdateTickers(context.Background(), asset.Spot)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

func TestGetCryptoDepositAddress(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("api keys not set")
	}
	_, err := g.GetCryptoDepositAddress(context.Background(), currency.USDT.String())
	if err != nil {
		t.Error(err)
	}
}

func TestGetAvailableTransferTrains(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("api keys not set")
	}
	_, err := g.GetAvailableTransferChains(context.Background(), currency.USDT)
	if err != nil {
		t.Error(err)
	}
}

// ******************************************BEGIN****************************************************
func TestListAllCurrencies(t *testing.T) {
	t.Parallel()
	if _, er := g.ListAllCurrencies(context.Background()); er != nil {
		t.Errorf("%s ListAllCurrencies() error %v", g.Name, er)
	}
}

func TestGetCurrencyDetail(t *testing.T) {
	t.Parallel()
	if _, er := g.GetCurrencyDetail(context.Background(), currency.BTC); er != nil {
		t.Errorf("%s GetCurrencyDetail() error %v", g.Name, er)
	}
}

func TestListAllCurrencyPairs(t *testing.T) {
	t.Parallel()
	if _, er := g.ListAllCurrencyPairs(context.Background()); er != nil {
		t.Errorf("%s ListAllCurrencyPairs() error %v", g.Name, er)
	}
}

func TestGetCurrencyPairDetal(t *testing.T) {
	t.Parallel()
	if _, er := g.GetCurrencyPairDetail(context.Background(), currency.NewPair(currency.BTC, currency.USDT)); er != nil {
		t.Errorf("%s GetCurrencyPairDetal() error %v", g.Name, er)
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	if _, er := g.GetTickers(context.Background(), currency.Pair{}, ""); er != nil {
		t.Errorf("%s GetTickers() error %v", g.Name, er)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	if _, er := g.GetTicker(context.Background(), currency.NewPair(currency.BTC, currency.USDT), UTC8TimeZone); er != nil {
		t.Errorf("%s GetTicker() error %v", g.Name, er)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	if _, er := g.GetOrderbook(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "0.1", 0, true); er != nil {
		t.Errorf("%s GetOrderbook() error %v", g.Name, er)
	}
}

func TestGetMarketTrades(t *testing.T) {
	t.Parallel()
	if _, er := g.GetMarketTrades(context.Background(), currency.NewPair(currency.BTC, currency.USDT), 0, "", true, time.Time{}, time.Time{}, 1); er != nil {
		t.Errorf("%s GetMarketTrades() error %v", g.Name, er)
	}
}

func TestGetCandlesticks(t *testing.T) {
	t.Parallel()
	if _, er := g.GetCandlesticks(context.Background(), currency.NewPair(currency.BTC, currency.USDT), 0, time.Time{}, time.Time{}, kline.OneDay); er != nil {
		t.Errorf("%s GetCandlesticks() error %v", g.Name, er)
	}
}
func TestGetTradingFeeRatio(t *testing.T) {
	t.Parallel()
	if _, er := g.GetTradingFeeRatio(context.Background(), currency.NewPair(currency.BTC, currency.USDT)); er != nil {
		t.Errorf("%s GetTradingFeeRatio() error %v", g.Name, er)
	}
}

func TestGetSpotAccounts(t *testing.T) {
	t.Parallel()
	if _, er := g.GetSpotAccounts(context.Background(), currency.BTC); er != nil {
		t.Errorf("%s GetSpotAccounts() error %v", g.Name, er)
	}
}

func TestCreateBatchOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	if _, er := g.CreateBatchOrders(context.Background(), []CreateOrderRequestData{
		{
			CurrencyPair: currency.NewPair(currency.BTC, currency.USDT),
			Side:         "sell",
			Amount:       1,
			Price:        1234567789,
			Account:      asset.Spot,
			Type:         "limit",
		},
		{
			CurrencyPair: currency.NewPair(currency.BTC, currency.USDT),
			Side:         "buy",
			Amount:       1,
			Price:        1234567789,
			Account:      asset.Spot,
			Type:         "limit",
		},
	}); er != nil {
		t.Errorf("%s CreateBatchOrders() error %v", g.Name, er)
	}
}

func TestGetSpotOpenOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := g.GetSpotOpenOrders(context.Background(), 0, 0, asset.Spot); er != nil {
		t.Errorf("%s GetSpotOpenOrders() error %v", g.Name, er)
	}
}

func TestSpotClosePositionWhenCrossCurrencyDisabled(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	if _, er := g.SpotClosePositionWhenCrossCurrencyDisabled(context.Background(), ClosePositionRequestParam{
		Amount:       0.1,
		Price:        1234567384,
		CurrencyPair: currency.NewPair(currency.BTC, currency.USDT),
	}); er != nil {
		t.Errorf("%s SpotClosePositionWhenCrossCurrencyDisabled() error %v", g.Name, er)
	}
}

func TestCreateSpotOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	if _, er := g.CreateSpotOrder(context.Background(), CreateOrderRequestData{
		CurrencyPair: currency.NewPair(currency.BTC, currency.USDT),
		Side:         "buy",
		Amount:       1,
		Price:        1234567789,
		Account:      asset.Spot,
		Type:         "limit",
	}); er != nil {
		t.Errorf("%s CreateSpotOrder() error %v", g.Name, er)
	}
}

func TestGetSpotOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := g.GetSpotOrders(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "open", 0, 0); er != nil {
		t.Errorf("%s GetSpotOrders() error %v", g.Name, er)
	}
}

func TestCancelAllOpenOrdersSpecifiedCurrencyPair(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := g.CancelAllOpenOrdersSpecifiedCurrencyPair(context.Background(), currency.NewPair(currency.BTC, currency.USDT), order.Sell, asset.Empty); er != nil {
		t.Errorf("%s CancelAllOpenOrdersSpecifiedCurrencyPair() error %v", g.Name, er)
	}
}

func TestCancelBatchOrdersWithIDList(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	if _, er := g.CancelBatchOrdersWithIDList(context.Background(), []CancelOrderByIDParam{
		{
			CurrencyPair: currency.NewPair(currency.BTC, currency.USDT),
			ID:           "1234567",
		},
		{
			CurrencyPair: currency.NewPair(currency.ETH, currency.USDT),
			ID:           "something",
		},
	}); er != nil {
		t.Errorf("%s CancelBatchOrderWithIDList() error %v", g.Name, er)
	}
}

var spotOrderJSON = `{"id": "12332324","text": "t-123456","create_time": "1548000000","update_time": "1548000100","create_time_ms": 1548000000123,"update_time_ms": 1548000100123,"currency_pair": "ETH_BTC","status": "cancelled","type": "limit","account": "spot","side": "buy","iceberg": "0","amount": "1","price": "5.00032","time_in_force": "gtc","left": "0.5","filled_total": "2.50016","fee": "0.005","fee_currency": "ETH","point_fee": "0","gt_fee": "0","gt_discount": false,"rebated_fee": "0","rebated_fee_currency": "BTC"}`

func TestGetSpotOrder(t *testing.T) {
	t.Parallel()
	var response SpotOrder
	if er := json.Unmarshal([]byte(spotOrderJSON), &response); er != nil {
		t.Errorf("%s error while deserializing to SpotOrder %v", g.Name, er)
	}
	if _, er := g.GetSpotOrder(context.Background(), "1234", currency.NewPair(currency.BTC, currency.USDT), asset.Spot); er != nil && !strings.Contains(er.Error(), "Order with ID 1234 not found") {
		t.Errorf("%s GetSpotOrder() error %v", g.Name, er)
	}
}
func TestCancelSingleSpotOrder(t *testing.T) {
	t.Parallel()
	if _, er := g.CancelSingleSpotOrder(context.Background(), "1234", currency.NewPair(currency.ETH, currency.USDT), asset.Empty); er != nil && !strings.Contains(er.Error(), "Order not found") {
		t.Errorf("%s CancelSingleSpotOrder() error %v", g.Name, er)
	}
}

var personalTradingHistoryJSON = `{"id": "1232893232","create_time": "1548000000","create_time_ms": "1548000000123.456","order_id": "4128442423","side": "buy","role": "maker","amount": "0.15","price": "0.03","fee": "0.0005","fee_currency": "ETH","point_fee": "0","gt_fee": "0"}`

func TestGetPersonalTradingHistory(t *testing.T) {
	t.Parallel()
	var response SpotPersonalTradeHistory
	if er := json.Unmarshal([]byte(personalTradingHistoryJSON), &response); er != nil {
		t.Errorf("%s error while deserializing to PersonalTrading History %v", g.Name, er)
	}
	if _, er := g.GetPersonalTradingHistory(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "", 0, 0, asset.Spot, time.Time{}, time.Time{}); er != nil {
		t.Errorf("%s GetPersonalTradingHistory() error %v", g.Name, er)
	}
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	if _, er := g.GetServerTime(context.Background()); er != nil {
		t.Errorf("%s GetServerTime() error %v", g.Name, er)
	}
}

func TestCountdownCancelorder(t *testing.T) {
	t.Parallel()
	if _, er := g.CountdownCancelorder(context.Background(), CountdownCancelOrderParam{
		Timeout:      10,
		CurrencyPair: currency.NewPair(currency.BTC, currency.ETH),
	}); er != nil {
		t.Errorf("%s CountdownCancelorder() error %v", g.Name, er)
	}
}

func TestCreatePriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	if _, er := g.CreatePriceTriggeredOrder(context.Background(), PriceTriggeredOrderParam{
		Trigger: TriggerPriceInfo{
			Price:      123,
			Rule:       ">=",
			Expiration: 3600,
		},
		Put: PutOrderData{
			Type:        "limit",
			Side:        "sell",
			Price:       2312312,
			Amount:      30,
			TimeInForce: "gtc",
		},
		Market: currency.NewPair(currency.GT, currency.USDT),
	}); er != nil {
		t.Errorf("%s CreatePriceTriggeredOrder() erro %v", g.Name, er)
	}
}

func TestGetPriceTriggeredOrderList(t *testing.T) {
	t.Parallel()
	if _, er := g.GetPriceTriggeredOrderList(context.Background(), "open", currency.EMPTYPAIR, asset.Empty, 0, 0); er != nil {
		t.Errorf("%s GetPriceTriggeredOrderList() error %v", g.Name, er)
	}
}

func TestCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	if _, er := g.CancelAllOpenOrders(context.Background(), currency.EMPTYPAIR, asset.CrossMargin); er != nil {
		t.Errorf("%s CancelAllOpenOrders() error %v", g.Name, er)
	}
}

var singlePriceTriggeredOrderJSON = `{"trigger": {"price": "100", "rule": ">=","expiration": 3600	},"put": {"type": "limit","side": "buy",	  "price": "2.15",	  "amount": "2.00000000",	  "account": "normal",	  "time_in_force": "gtc"	},	"id": 1283293,	"user": 1234,	"market": "GT_USDT",	"ctime": 1616397800,	"ftime": 1616397801,	"fired_order_id": 0,	"status": "",	"reason": ""}`

func TestGetSinglePriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	var response SpotPriceTriggeredOrder
	if err := json.Unmarshal([]byte(singlePriceTriggeredOrderJSON), &response); err != nil {
		t.Errorf("%s error while deserializing to SpotPriceTriggeredOrder %v", g.Name, err)
	}
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	if _, err := g.GetSinglePriceTriggeredOrder(context.Background(), "1234"); err != nil && !strings.Contains(err.Error(), "no order_id match") {
		t.Errorf("%s GetSinglePriceTriggeredOrder() error %v", g.Name, err)
	}
}

func TestCancelPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.CancelPriceTriggeredOrder(context.Background(), "1234"); err != nil &&
		!strings.Contains(err.Error(), "no order_id match") {
		t.Errorf("%s CancelPriceTriggeredOrder() error %v", g.Name, err)
	}
}

var singleMarginAccountJSON = `{"currency_pair": "ETH_BTC",  "locked": false,  "risk": "1.1",  "base": {    "currency": "ETH",    "available": "30.1",    "locked": "0",    "borrowed": "10.1",    "interest": "0"  },  "quote": {    "currency": "BTC",    "available": "10",    "locked": "0",    "borrowed": "1.5",    "interest": "0"  }}`

func TestGetMarginAccountList(t *testing.T) {
	t.Parallel()
	var response MarginAccountItem
	if err := json.Unmarshal([]byte(singleMarginAccountJSON), &response); err != nil {
		t.Errorf("%s deserializing to MarginAccountItem error %v", g.Name, err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.GetMarginAccountList(context.Background(), currency.EMPTYPAIR); err != nil {
		t.Errorf("%s GetMarginAccountList() error %v", g.Name, err)
	}
}

var marginAccountBalanceChangeHistoryJSON = `{  "id": "123456",  "time": "1547633726",  "time_ms": 1547633726123,  "currency": "BTC",  "currency_pair": "BTC_USDT",  "change": "1.03",  "balance": "4.59316525194"}`

func TestListMarginAccountBalanceChangeHistory(t *testing.T) {
	t.Parallel()
	var response MarginAccountBalanceChangeInfo
	if err := json.Unmarshal([]byte(marginAccountBalanceChangeHistoryJSON), &response); err != nil {
		t.Errorf("%s deserializes to MarginAccountBalanceChangeInfo error %v", g.Name, err)
	}
	if _, err := g.ListMarginAccountBalanceChangeHistory(context.Background(), currency.BTC, currency.NewPair(currency.BTC, currency.USDT), time.Time{}, time.Time{}, 0, 0); err != nil {
		t.Errorf("%s ListMarginAccountBalanceChangeHistory() error %v", g.Name, err)
	}
}

var getMarginFundingAccountListJSON = `{  "currency": "BTC",  "available": "1.238",  "locked": "0",  "lent": "3.32",  "total_lent": "3.32"}`

func TestGetMarginFundingAccountList(t *testing.T) {
	t.Parallel()
	var response MarginFundingAccountItem
	if err := json.Unmarshal([]byte(getMarginFundingAccountListJSON), &response); err != nil {
		t.Errorf("%s error while deserializing to MarginFundingAccountItem %v", g.Name, err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.GetMarginFundingAccountList(context.Background(), currency.BTC); err != nil {
		t.Errorf("%s GetMarginFundingAccountList %v", g.Name, err)
	}
}

var marginLoanJSON = `{"side":"borrow","currency":"BTC","rate":"0.002","amount":"1.5","days":10,"auto_renew": true,	"currency_pair": "ETH_BTC",	"fee_rate": "0.18",	"orig_id": "123424",	"text": "t-abc"}`

func TestMarginLoan(t *testing.T) {
	t.Parallel()
	var response MarginLoanResponse
	if err := json.Unmarshal([]byte(marginLoanJSON), &response); err != nil {
		t.Errorf("%s error while deserializing to MarginLoanResponse %v", g.Name, err)
	}
	if _, err := g.MarginLoan(context.Background(), MarginLoanRequestParam{
		Side:     "borrow",
		Amount:   1,
		Currency: currency.BTC,
		Days:     10,
	}); err != nil {
		t.Errorf("%s MarginLoan() error %v", g.Name, err)
	}
}

func TestGetMarginAllLoans(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.GetMarginAllLoans(context.Background(), "open", "", currency.USD, currency.NewPair(currency.BTC, currency.USDT), "", false, 0, 0); err != nil {
		t.Errorf("%s GetMarginAllLoans() error %v", g.Name, err)
	}
}

func TestMergeMultipleLendingLoans(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.MergeMultipleLendingLoans(context.Background(), currency.USDT, []string{"123", "23423"}); err != nil && !strings.Contains(err.Error(), "Orders which can be merged are not found") {
		t.Errorf("%s MergeMultipleLendingLoans() error %v", g.Name, err)
	}
}

func TestRetriveOneSingleLoanDetail(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.RetriveOneSingleLoanDetail(context.Background(), "borrow", "123"); err != nil && !strings.Contains(err.Error(), "Loan not found") {
		t.Errorf("%s RetriveOneSingleLoanDetail() error %v", g.Name, err)
	}
}

func TestModifyALoan(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.ModifyALoan(context.Background(), "1234", ModifyLoanRequestParam{
		Currency:  currency.BTC,
		Side:      "borrow",
		AutoRenew: false,
	}); err != nil && !strings.Contains(err.Error(), "Loan not found") {
		t.Errorf("%s ModifyALoan() error %v", g.Name, err)
	}
}

func TestCancelLendingLoan(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.CancelLendingLoan(context.Background(), currency.BTC, "1234"); err != nil && !strings.Contains(err.Error(), "Loan not found") {
		t.Errorf("%s CancelLendingLoan() error %v", g.Name, err)
	}
}

func TestRepayALoan(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.RepayALoan(context.Background(), "1234", RepayLoanRequestParam{
		CurrencyPair: currency.NewPair(currency.BTC, currency.USDT),
		Currency:     currency.BTC,
		Mode:         "all",
	}); err != nil && !strings.Contains(err.Error(), "Loan not found") {
		t.Errorf("%s RepayALoan() error %v", g.Name, err)
	}
}

var listLoanRepaymentRecordsJSON = `{"id": "12342323","create_time": "1578000000","principal": "100","interest": "2"}`

func TestListLoanRepaymentRecords(t *testing.T) {
	t.Parallel()
	var response LoanRepaymentRecord
	if err := json.Unmarshal([]byte(listLoanRepaymentRecordsJSON), &response); err != nil {
		t.Errorf("%s error while deserializing to LoanRepaymentRecord %v", g.Name, err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.ListLoanRepaymentRecords(context.Background(), "1234"); err != nil &&
		!strings.Contains(err.Error(), "Loan not found") {
		t.Errorf("%s LoanRepaymentRecord() error %v", g.Name, err)
	}
}

var loanRecordJSON = `{  "id": "122342323",  "loan_id": "12840282",  "create_time": "1548000000",  "expire_time": "1548100000",  "status": "loaned",  "borrow_user_id": "******12",  "currency": "BTC",  "rate": "0.002",  "amount": "1.5",  "days": 10,  "auto_renew": false,  "repaid": "0",  "paid_interest": "0",  "unpaid_interest": "0"}`

func TestListRepaymentRecordsOfSpecificLoan(t *testing.T) {
	t.Parallel()
	var response LoanRecord
	if err := json.Unmarshal([]byte(listLoanRepaymentRecordsJSON), &response); err != nil {
		t.Errorf("%s error while deserializing to LoanRecord %v", g.Name, err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.ListRepaymentRecordsOfSpecificLoan(context.Background(), "1234", "", 0, 0); err != nil {
		t.Errorf("%s error while ListRepaymentRecordsOfSpecificLoan() %v", g.Name, err)
	}
}

func TestGetOneSingleloanRecord(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.GetOneSingleloanRecord(context.Background(), "1234", "123"); err != nil && !strings.Contains(err.Error(), "Loan record not found") {
		t.Errorf("%s error while GetOneSingleloanRecord() %v", g.Name, err)
	}
}

func TestModifyALoanRecord(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.ModifyALoanRecord(context.Background(), "1234", ModifyLoanRequestParam{
		Currency:     currency.USDT,
		CurrencyPair: currency.NewPair(currency.BTC, currency.USDT),
		Side:         "lend",
		AutoRenew:    true,
		LoanID:       "1234",
	}); err != nil && !strings.Contains(err.Error(), "Loan record not found") {
		t.Errorf("%s ModifyALoanRecord() error %v", g.Name, err)
	}
}

func TestUpdateUsersAutoRepaymentSetting(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.UpdateUsersAutoRepaymentSetting(context.Background(), "on"); err != nil {
		t.Errorf("%s UpdateUsersAutoRepaymentSetting() error %v", g.Name, err)
	}
}

func TestGetUserAutoRepaymentSetting(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.GetUserAutoRepaymentSetting(context.Background()); err != nil {
		t.Errorf("%s GetUserAutoRepaymentSetting() error %v", g.Name, err)
	}
}

func TestGetMaxTransferableAmountForSpecificMarginCurrency(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.GetMaxTransferableAmountForSpecificMarginCurrency(context.Background(), currency.BTC, currency.EMPTYPAIR); err != nil {
		t.Errorf("%s GetMaxTransferableAmountForSpecificMarginCurrency() error %v", g.Name, err)
	}
}

func TestGetMaxBorrowableAmountForSpecificMarginCurrency(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.GetMaxBorrowableAmountForSpecificMarginCurrency(context.Background(), currency.BTC, currency.EMPTYPAIR); err != nil && !strings.Contains(err.Error(), "No margin account or margin balance is not enough") {
		t.Errorf("%s GetMaxBorrowableAmountForSpecificMarginCurrency() error %v", g.Name, err)
	}
}

var currencySupportedByCrossMarginJSON = `{	"name": "BTC",	"rate": "0.0002",	"prec": "0.000001",	"discount": "1",	"min_borrow_amount": "0.01",	"user_max_borrow_amount": "1000000",	"total_max_borrow_amount": "10000000",	"price": "63015.5214",	"status": 1}`

func TestCurrencySupportedByCrossMargin(t *testing.T) {
	t.Parallel()
	var response CrossMarginCurrencies
	if err := json.Unmarshal([]byte(currencySupportedByCrossMarginJSON), &response); err != nil {
		t.Errorf("%s error while deserializing to CrossMarginCurrencies error %v", g.Name, err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.CurrencySupportedByCrossMargin(context.Background()); err != nil {
		t.Errorf("%s CurrencySupportedByCrossMargin() error %v", g.Name, err)
	}
}

func TestGetCrossMarginSupportedCurrencyDetail(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.GetCrossMarginSupportedCurrencyDetail(context.Background(), currency.BTC); err != nil {
		t.Errorf("%s GetCrossMarginSupportedCurrencyDetail() error %v", g.Name, err)
	}
}

var crossMarginAccountsResponseJSON = `{	"user_id": 10001,	"locked": false,	"balances": {	  "ETH": {		"available": "0",		"freeze": "0",		"borrowed": "0.075393666654",		"interest": "0.0000106807603333"	  },	  "POINT": {		"available": "9999999999.017023138734",		"freeze": "0",		"borrowed": "0",		"interest": "0"	  },	  "USDT": {		"available": "0.00000062023",		"freeze": "0",		"borrowed": "0",		"interest": "0"	  }	},	"total": "230.94621713",	"borrowed": "161.66395521",	"interest": "0.02290237",	"risk": "1.4284",	"total_initial_margin": "1025.0524665088",	"total_margin_balance": "3382495.944473949183",	"total_maintenance_margin": "205.01049330176",	"total_initial_margin_rate": "3299.827135672679",	"total_maintenance_margin_rate": "16499.135678363399",	"total_available_margin": "3381470.892007440383"}`

func TestGetCrossMarginAccounts(t *testing.T) {
	t.Parallel()
	var response CrossMarginAccount
	if err := json.Unmarshal([]byte(crossMarginAccountsResponseJSON), &response); err != nil {
		t.Errorf("%s error while deserializing to CrossMarginAccounts error %v", g.Name, err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.GetCrossMarginAccounts(context.Background()); err != nil {
		t.Errorf("%s GetCrossMarginAccounts() error %v", g.Name, err)
	}
}

var crossMarginAccountChangeHistoryJSON = `{"id": "123456","time": 1547633726123,  "currency": "BTC",  "change": "1.03",  "balance": "4.59316525194",  "type": "in"}`

func TestGetCrossMarginAccountChangeHistory(t *testing.T) {
	t.Parallel()
	var response CrossMarginAccountHistoryItem
	if err := json.Unmarshal([]byte(crossMarginAccountChangeHistoryJSON), &response); err != nil {
		t.Errorf("%s error while deserializing to CrossMarginAccountHistoryItem error %v", g.Name, err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.GetCrossMarginAccountChangeHistory(context.Background(), currency.BTC, time.Time{}, time.Time{}, 0, 6, "in"); err != nil {
		t.Errorf("%s GetCrossMarginAccountChangeHistory() error %v", g.Name, err)
	}
}

var createCrossMarginBorrowLoanJSON = `{	"id": "17",	"create_time": 1620381696159,	"update_time": 1620381696159,	"currency": "EOS",	"amount": "110.553635",	"text": "web",	"status": 2,	"repaid": "110.506649705159",	"repaid_interest": "0.046985294841",	"unpaid_interest": "0.0000074393366667"}`

func TestCreateCrossMarginBorrowLoan(t *testing.T) {
	t.Parallel()
	var response CrossMarginLoanResponse
	if err := json.Unmarshal([]byte(createCrossMarginBorrowLoanJSON), &response); err != nil {
		t.Errorf("%s error while deserializing to CrossMarginBorrowLoanResponse %v", g.Name, err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.CreateCrossMarginBorrowLoan(context.Background(), CrossMarginBorrowLoanParams{
		Currency: currency.BTC,
		Amount:   3,
	}); err != nil {
		t.Errorf("%s CreateCrossMarginBorrowLoan() error %v", g.Name, err)
	}
}

func TestGetCrossMarginBorrowHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.GetCrossMarginBorrowHistory(context.Background(), 1, currency.BTC, 0, 0, false); err != nil {
		t.Errorf("%s GetCrossMarginBorrowHistory() error %v", g.Name, err)
	}
}

func TestGetSingleBorrowLoanDetail(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.GetSingleBorrowLoanDetail(context.Background(), "1234"); err != nil {
		t.Errorf("%s GetSingleBorrowLoanDetail() error %v", g.Name, err)
	}
}

var executeRepayment = `{"id": "17","create_time": 1620381696159,  "update_time": 1620381696159,  "currency": "EOS",  "amount": "110.553635",  "text": "web",  "status": 2,  "repaid": "110.506649705159",  "repaid_interest": "0.046985294841",  "unpaid_interest": "0.0000074393366667"}`

func TestExecuteRepayment(t *testing.T) {
	t.Parallel()
	var response CrossMarginLoanResponse
	if err := json.Unmarshal([]byte(executeRepayment), &response); err != nil {
		t.Errorf("%s error while deserializing to CrossMarginLoanResponse error %v", g.Name, err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.ExecuteRepayment(context.Background(), CurrencyAndAmount{
		Currency: currency.USD,
		Amount:   1234.55,
	}); err != nil {
		t.Errorf("%s ExecuteRepayment() error %v", g.Name, err)
	}
}

var getCrossMarginRepaymentJSON = `{"id": "51","create_time": 1620696347990, "loan_id": "30",  "currency": "BTC",  "principal": "5.385542",  "interest": "0.000044879516"}`

func TestGetCrossMarginRepayments(t *testing.T) {
	t.Parallel()
	var response RepaymentHistoryItem
	if err := json.Unmarshal([]byte(getCrossMarginRepaymentJSON), &response); err != nil {
		t.Errorf("%s error while deserializing to RepaymentHistoryItem error %v", g.Name, err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.GetCrossMarginRepayments(context.Background(), currency.BTC, "123", 0, 0, false); err != nil {
		t.Errorf("%s GetCrossMarginRepayments() error %v", g.Name, err)
	}
}

func TestGetMaxTransferableAmountForSpecificCrossMarginCurrency(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.GetMaxTransferableAmountForSpecificCrossMarginCurrency(context.Background(), currency.BTC); err != nil {
		t.Errorf("%s GetMaxTransferableAmountForSpecificCrossMarginCurrency() error %v", g.Name, err)
	}
}

func TestGetMaxBorrowableAmountForSpecificCrossMarginCurrency(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.GetMaxBorrowableAmountForSpecificCrossMarginCurrency(context.Background(), currency.BTC); err != nil {
		t.Errorf("%s GetMaxBorrowableAmountForSpecificCrossMarginCurrency() error %v", g.Name, err)
	}
}

func TestListCurrencyChain(t *testing.T) {
	t.Parallel()
	if _, er := g.ListCurrencyChain(context.Background(), currency.BTC); er != nil {
		t.Errorf("%s ListCurrencyChain() error %v", g.Name, er)
	}
}

func TestGenerateCurrencyDepositAddress(t *testing.T) {
	t.Parallel()
	if _, er := g.GenerateCurrencyDepositAddress(context.Background(), currency.BTC); er != nil {
		t.Errorf("%s GenerateCurrencyDepositAddress() error %v", g.Name, er)
	}
}

func TestGetWithdrawalRecords(t *testing.T) {
	t.Parallel()
	if _, er := g.GetWithdrawalRecords(context.Background(), currency.BTC, time.Time{}, time.Time{}, 0, 0); er != nil {
		t.Errorf("%s GetWithdrawalRecords() error %v", g.Name, er)
	}
}

func TestGetDepositRecords(t *testing.T) {
	t.Parallel()
	if _, er := g.GetDepositRecords(context.Background(), currency.BTC, time.Time{}, time.Time{}, 0, 0); er != nil {
		t.Errorf("%s GetDepositRecords() error %v", g.Name, er)
	}
}

func TestTransferCurrency(t *testing.T) {
	t.Parallel()
	if _, er := g.TransferCurrency(context.Background(), TransferCurrencyParam{
		Currency:     currency.BTC,
		From:         asset.Spot,
		To:           asset.Margin,
		Amount:       1202.000,
		CurrencyPair: currency.NewPair(currency.BTC, currency.USDT),
	}); er != nil && !strings.Contains(er.Error(), "BALANCE_NOT_ENOUGH") {
		t.Errorf("%s TransferCurrency() error %v", g.Name, er)
	}
}

func TestSubAccountTransfer(t *testing.T) {
	t.Parallel()
	if er := g.SubAccountTransfer(context.Background(), SubAccountTransferParam{
		Currency:   currency.BTC,
		SubAccount: "12222",
		Direction:  "to",
		Amount:     1,
	}); er != nil && !strings.Contains(er.Error(), "invalid account") {
		t.Errorf("%s SubAccountTransfer() error %v", g.Name, er)
	}
}

var subAccountTransferHistoryJSON = `{"uid": "10001","timest": "1592809000","source": "web","currency": "BTC","sub_account": "10002","direction": "to","amount": "1","sub_account_type": "spot"}`

func TestGetSubAccountTransferHistory(t *testing.T) {
	t.Parallel()
	var response SubAccountTransferResponse
	if er := json.Unmarshal([]byte(subAccountTransferHistoryJSON), &response); er != nil {
		t.Errorf("%s deserializing to SubAccountTransferResponse error %v", g.Name, er)
	}
	if _, er := g.GetSubAccountTransferHistory(context.Background(), "", time.Time{}, time.Time{}, 0, 0); er != nil {
		t.Errorf("%s GetSubAccountTransferHistory() error %v", g.Name, er)
	}
}

var withdrawalStatusJSON = `{"currency": "GT","name": "GateToken","name_cn": "GateToken","deposit": "0","withdraw_percent": "0%","withdraw_fix": "0.01","withdraw_day_limit": "20000","withdraw_day_limit_remain": "20000","withdraw_amount_mini": "0.11","withdraw_eachtime_limit": "20000","withdraw_fix_on_chains": {  "BTC": "20",  "ETH": "15",  "TRX": "0",  "EOS": "2.5"}}`

func TestGetWithdrawalStatus(t *testing.T) {
	t.Parallel()
	var response WithdrawalStatus
	if er := json.Unmarshal([]byte(withdrawalStatusJSON), &response); er != nil {
		t.Errorf("%s error while deserializing to WithdrawalStatus %v", g.Name, er)
	}
	if _, er := g.GetWithdrawalStatus(context.Background(), currency.NewCode("")); er != nil {
		t.Errorf("%s GetWithdrawalStatus() error %v", g.Name, er)
	}
}

var subAccountBalanceJSON = `{"uid": "10003","available": {  "BTC": "0.1",  "GT": "2000",  "USDT": "10"}}`

func TestGetSubAccountBalances(t *testing.T) {
	t.Parallel()
	var response SubAccountBalance
	if er := json.Unmarshal([]byte(subAccountBalanceJSON), &response); er != nil {
		t.Errorf("%s deserializes to SubAccountBalance error %v", g.Name, er)
	}
	if _, er := g.GetSubAccountBalances(context.Background(), ""); er != nil {
		t.Errorf("%s GetSubAccountBalances() error %v", g.Name, er)
	}
}

var subAccountMarginBalance = `{"uid": "10000","available": [  {    "locked": false,    "currency_pair": "BTC_USDT",    "risk": "9999.99",    "base": {      "available": "0.1",      "borrowed": "0",      "interest": "0",      "currency": "BTC",      "locked": "0"    },    "quote": {      "available": "0",      "borrowed": "0",      "interest": "0",      "currency": "USDT",      "locked": "0"    }  }]}`

func TestGetSubAccountMarginBalances(t *testing.T) {
	t.Parallel()
	var response SubAccountMarginBalance
	if er := json.Unmarshal([]byte(subAccountMarginBalance), &response); er != nil {
		t.Errorf("%s error while deserializing to SubAccountMarginBalance %v", g.Name, er)
	}
	if _, er := g.GetSubAccountMarginBalances(context.Background(), ""); er != nil {
		t.Errorf("%s GetSubAccountMarginBalances() error %v", g.Name, er)
	}
}

func TestGetSubAccountFuturesBalances(t *testing.T) {
	t.Parallel()
	if _, er := g.GetSubAccountFuturesBalances(context.Background(), "", ""); er != nil {
		t.Errorf("%s GetSubAccountFuturesBalance() error %v", g.Name, er)
	}
}

var subAccountCrossMarginInfo = `{"uid": "100000","available": {  "user_id": 100003,  "locked": false,  "total": "20.000000",  "borrowed": "0.000000",  "interest": "0",  "borrowed_net": "0",  "net": "20",  "leverage": "3",  "risk": "9999.99",  "total_initial_margin": "0.00",  "total_margin_balance": "20.00",  "total_maintenance_margin": "0.00",  "total_initial_margin_rate": "9999.9900",  "total_maintenance_margin_rate": "9999.9900",  "total_available_margin": "20.00",  "balances": {    "USDT": {      "available": "20.000000",      "freeze": "0.000000",      "borrowed": "0.000000",      "interest": "0.000000"    }  }}}`

func TestGetSubAccountCrossMarginBalances(t *testing.T) {
	t.Parallel()
	var response SubAccountCrossMarginInfo
	if er := json.Unmarshal([]byte(subAccountCrossMarginInfo), &response); er != nil {
		t.Errorf("%s error while deserializing to SubAccountCrossMarginInfo %v", g.Name, er)
	}
	if _, er := g.GetSubAccountCrossMarginBalances(context.Background(), ""); er != nil {
		t.Errorf("%s GetSubAccountCrossMarginBalances() error %v", g.Name, er)
	}
}

var savedAddressJSON = `{"currency": "usdt","chain": "TRX","address": "TWYirLzw2RARB2jfeFcfRPmeuU3rC7rakT","name": "gate","tag": "","verified": "1"}`

func TestGetSavedAddresses(t *testing.T) {
	t.Parallel()
	var response WalletSavedAddress
	if er := json.Unmarshal([]byte(savedAddressJSON), &response); er != nil {
		t.Errorf("%s error while deserializing to WalletSavedAddress %v", g.Name, er)
	}
	if _, er := g.GetSavedAddresses(context.Background(), currency.BTC, "", 0); er != nil {
		t.Errorf("%s GetSavedAddresses() error %v", g.Name, er)
	}
}

var personalTradingFeeJSON = `{"user_id": 10001,"taker_fee": "0.002","maker_fee": "0.002","futures_taker_fee": "-0.00025","futures_maker_fee": "0.00075","gt_discount": false,"gt_taker_fee": "0","gt_maker_fee": "0","loan_fee": "0.18","point_type": "1"}`

func TestGetPersonalTradingFee(t *testing.T) {
	t.Parallel()
	var response PersonalTradingFee
	if er := json.Unmarshal([]byte(personalTradingFeeJSON), &response); er != nil {
		t.Errorf("%s GetPersonalTradingFee() error %v", g.Name, er)
	}
	if _, er := g.GetPersonalTradingFee(context.Background(), currency.NewPair(currency.BTC, currency.USDT)); er != nil {
		t.Errorf("%s GetPersonalTradingFee() error %v", g.Name, er)
	}
}

var usersTotalBalanceJSON = `{"details": {"cross_margin": {"amount": "0","currency": "USDT"},"spot": {"currency": "USDT","amount": "42264489969935775.5160259954878034182418"},"finance": {"amount": "662714381.70310327810191647181","currency": "USDT"},"margin": {"amount": "1259175.664137668554329559","currency": "USDT"},"quant": {"amount": "591702859674467879.6488202650892478553852","currency": "USDT"},"futures": {"amount": "2384175.5606114082065","currency": "USDT"},"delivery": {	"currency": "USDT",	"amount": "1519804.9756702"},"warrant": {"amount": "0","currency": "USDT"},"cbbc": {"currency": "USDT","amount": "0"}},"total": {"currency": "USDT","amount": "633967350312281193.068368815439797304437"}}`

func TestGetUsersTotalBalance(t *testing.T) {
	t.Parallel()
	var response UsersAllAccountBalance
	if er := json.Unmarshal([]byte(usersTotalBalanceJSON), &response); er != nil {
		t.Errorf("%s error while deserializing to UsersAllAccountBalance %v", g.Name, er)
	}
	if _, er := g.GetUsersTotalBalance(context.Background(), currency.BTC); er != nil {
		t.Errorf("%s GetUsersTotalBalance() error %v", g.Name, er)
	}
}

func TestGetMarginSupportedCurrencyPairs(t *testing.T) {
	t.Parallel()
	if _, er := g.GetMarginSupportedCurrencyPairs(context.Background()); er != nil {
		t.Errorf("%s GetMarginSupportedCurrencyPair() error %v", g.Name, er)
	}
}

func TestGetMarginSupportedCurrencyPair(t *testing.T) {
	t.Parallel()
	if _, er := g.GetMarginSupportedCurrencyPair(context.Background(), currency.NewPair(currency.BTC, currency.USDT)); er != nil {
		t.Errorf("%s GetMarginSupportedCurrencyPair() error %v", g.Name, er)
	}
}

func TestGetOrderbookOfLendingLoans(t *testing.T) {
	t.Parallel()
	if _, er := g.GetOrderbookOfLendingLoans(context.Background(), currency.BTC); er != nil {
		t.Errorf("%s GetOrderbookOfLendingLoans() error %v", g.Name, er)
	}
}

func TestGetAllFutureContracts(t *testing.T) {
	t.Parallel()
	if _, er := g.GetAllFutureContracts(context.Background(), "USDT"); er != nil {
		t.Errorf("%s GetAllFutureContracts() error %v", g.Name, er)
	}
}
func TestGetSingleContract(t *testing.T) {
	t.Parallel()
	if _, er := g.GetSingleContract(context.Background(), "usdt", currency.NewPair(currency.BTC, currency.USDT)); er != nil {
		t.Errorf("%s GetSingleContract() error %s", g.Name, er)
	}
}

func TestGetFuturesOrderbook(t *testing.T) {
	t.Parallel()
	if _, er := g.GetFuturesOrderbook(context.Background(), "usdt", currency.NewPair(currency.BTC, currency.USDT), "0.1", 0, true); er != nil {
		t.Errorf("%s GetFuturesOrderbook() error %v", g.Name, er)
	}
}
func TestGetFuturesTradingHistory(t *testing.T) {
	t.Parallel()
	if _, er := g.GetFuturesTradingHistory(context.Background(), "usdt", currency.NewPair(currency.BTC, currency.USDT), 0, 0, "", time.Time{}, time.Time{}); er != nil {
		t.Errorf("%s GetFuturesTradingHistory() error %v", g.Name, er)
	}
}

func TestGetFuturesCandlesticks(t *testing.T) {
	t.Parallel()
	if _, er := g.GetFuturesCandlesticks(context.Background(), "usdt", currency.NewPair(currency.BTC, currency.USDT), time.Time{}, time.Time{}, 0, kline.OneWeek); er != nil {
		t.Errorf("%s GetFuturesCandlesticks() error %v", g.Name, er)
	}
}

func TestGetFutureTickers(t *testing.T) {
	t.Parallel()
	if _, er := g.GetFutureTickers(context.Background(), "usdt", currency.EMPTYPAIR); er != nil {
		t.Errorf("%s GetFutureTickers() error %v", g.Name, er)
	}
}

func TestGetFutureFundingRates(t *testing.T) {
	t.Parallel()
	if _, er := g.GetFutureFundingRates(context.Background(), "usdt", currency.NewPair(currency.BTC, currency.USDT), 0); er != nil {
		t.Errorf("%s GetFutureFundingRates() error %v", g.Name, er)
	}
}

func TestGetFuturesInsuranceBalanceHistory(t *testing.T) {
	t.Parallel()
	if _, er := g.GetFuturesInsuranceBalanceHistory(context.Background(), "usdt", 0); er != nil {
		t.Errorf("%s GetFuturesInsuranceBalanceHistory() error %v", g.Name, er)
	}
}

func TestGetFutureStats(t *testing.T) {
	t.Parallel()
	if _, er := g.GetFutureStats(context.Background(), "usdt", currency.NewPair(currency.BTC, currency.USDT), time.Time{}, kline.OneHour, 0); er != nil {
		t.Errorf("%s GetFutureStats() error %v", g.Name, er)
	}
}

func TestGetIndexConstituent(t *testing.T) {
	t.Parallel()
	if _, er := g.GetIndexConstituent(context.Background(), "usdt", currency.NewPair(currency.BTC, currency.USDT)); er != nil {
		t.Errorf("%s GetIndexConstituent() error %v", g.Name, er)
	}
}

func TestGetLiquidationHistory(t *testing.T) {
	t.Parallel()
	if _, er := g.GetLiquidationHistory(context.Background(), "usdt", currency.NewPair(currency.BTC, currency.USDT), time.Time{}, time.Time{}, 0); er != nil {
		t.Errorf("%s GetLiquidationHistory() error %v", g.Name, er)
	}
}
func TestQueryFuturesAccount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.QueryFuturesAccount(context.Background(), "usdt"); err != nil {
		t.Errorf("%s QueryFuturesAccount() error %v", g.Name, err)
	}
}

var getFuturesAccountBooksJSON = `{"time": 1547633726,  "change": "0.000010152188",  "balance": "4.59316525194",  "text": "ETH_USD:6086261",  "type": "fee"}`

func TestGetFuturesAccountBooks(t *testing.T) {
	t.Parallel()
	var response FuturesAccountBookItem
	if err := json.Unmarshal([]byte(getFuturesAccountBooksJSON), &response); err != nil {
		t.Errorf("%s error while deserializing FuturesAccountBookItem: %v", g.Name, err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.GetFuturesAccountBooks(context.Background(), "usdt", 0, time.Time{}, time.Time{}, "dnw"); err != nil {
		t.Errorf("%s GetFuturesAccountBooks() error %v", g.Name, err)
	}
}

var futuresPositionJSON = `{"user": 10000,"contract": "BTC_USDT","size": -9440,"leverage": "0","risk_limit": "100","leverage_max": "100","maintenance_rate": "0.005","value": "2.497143098997","margin": "4.431548146258","entry_price": "3779.55","liq_price": "99999999","mark_price": "3780.32","unrealised_pnl": "-0.000507486844","realised_pnl": "0.045543982432","history_pnl": "0","last_close_pnl": "0","realised_point": "0","history_point": "0","adl_ranking": 5,"pending_orders": 16,"close_order": {  "id": 232323,  "price": "3779",  "is_liq": false},"mode": "single","cross_leverage_limit": "0"}`

func TestGetAllPositionsOfUsers(t *testing.T) {
	t.Parallel()
	var response FuturesPosition
	if err := json.Unmarshal([]byte(futuresPositionJSON), &response); err != nil {
		t.Errorf("%s error while deserializing FuturesPosition: %v", g.Name, err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.GetAllPositionsOfUsers(context.Background(), "usdt"); err != nil {
		t.Errorf("%s GetAllPositionsOfUsers() error %v", g.Name, err)
	}
}

func TestGetSinglePosition(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.GetSinglePosition(context.Background(), "usdt", currency.Pair{Quote: currency.BTC, Base: currency.USDT}); err != nil {
		t.Errorf("%s GetSinglePosition() error %v", g.Name, err)
	}
}

func TestUpdatePositionMargin(t *testing.T) {
	t.Parallel()
	var response FuturesPosition
	if err := json.Unmarshal([]byte(""), &response); err != nil {
		t.Errorf("%s error while deserializing FuturesPosition: %v", g.Name, err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.UpdatePositionMargin(context.Background(), "usdt", 0.01, currency.NewPair(currency.ETH, currency.USD)); err != nil {
		t.Errorf("%s UpdatePositionMargin() error %v", g.Name, err)
	}
}

func TestUpdatePositionLeverage(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.UpdatePositionLeverage(context.Background(), "usdt", currency.Pair{Base: currency.BTC, Quote: currency.USDT}, 1, 0); err != nil {
		t.Errorf("%s UpdatePositionLeverage() error %v", g.Name, err)
	}
}

func TestUpdatePositionRiskLimit(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.UpdatePositionRiskLimit(context.Background(), "usdt", currency.Pair{Base: currency.BTC, Quote: currency.USDT}, 10); err != nil {
		t.Errorf("%s UpdatePositionRiskLimit() error %v", g.Name, err)
	}
}
func TestEnableOrDisableDualMode(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.EnableOrDisableDualMode(context.Background(), "btc", true); err != nil {
		t.Errorf("%s EnableOrDisableDualMode() error %v", g.Name, err)
	}
}

func TestRetrivePositionDetailInDualMode(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.RetrivePositionDetailInDualMode(context.Background(), "btc", currency.NewPair(currency.USDT, currency.BTC)); err != nil {
		t.Errorf("%s RetrivePositionDetailInDualMode() error %v", g.Name, err)
	}
}

func TestUpdatePositionMarginInDualMode(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.UpdatePositionMarginInDualMode(context.Background(), "btc", currency.NewPair(currency.USD, currency.USD), 0.001, "dual_long"); err != nil {
		t.Errorf("%s UpdatePositionMarginInDualMode() error %v", g.Name, err)
	}
}
func TestUpdatePositionLeverageInDualMode(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.UpdatePositionLeverageInDualMode(context.Background(), "usdt", currency.NewPair(currency.BTC, currency.USDT), 0.001, 0.001); err != nil {
		t.Errorf("%s UpdatePositionLeverageInDualMode() error %v", g.Name, err)
	}
}

func TestUpdatePositionRiskLimitinDualMode(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.UpdatePositionRiskLimitinDualMode(context.Background(), "usdt", currency.NewPair(currency.BTC, currency.USDT), 0.1); err != nil {
		t.Errorf("%s UpdatePositionRiskLimitinDualMode() error %v", g.Name, err)
	}
}

func TestCreateFuturesOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.CreateFuturesOrder(context.Background(), FuturesOrderCreateParams{
		Contract:    currency.NewPair(currency.BTC, currency.USDT),
		Size:        6024,
		Iceberg:     0,
		Price:       3765,
		TimeInForce: "gtc",
		Text:        "t-my-custom-id",
		Settle:      "btc",
	}); err != nil {
		t.Errorf("%s CreateFuturesOrder() error %v", g.Name, err)
	}
}

func TestGetAllDeliveryContracts(t *testing.T) {
	t.Parallel()
	if _, er := g.GetAllDeliveryContracts(context.Background(), "usdt"); er != nil {
		t.Errorf("%s GetAllDeliveryContracts() error %v", g.Name, er)
	}
}

func TestGetSingleDeliveryContracts(t *testing.T) {
	t.Parallel()
	if _, er := g.GetSingleDeliveryContracts(context.Background(), "usdt", "BTC_USDT_20220902"); er != nil {
		t.Errorf("%s GetSingleDeliveryContracts() error %v", g.Name, er)
	}
}

func TestGetDeliveryOrderbook(t *testing.T) {
	t.Parallel()
	if _, er := g.GetDeliveryOrderbook(context.Background(), "usdt", "BTC_USDT_20220902", "0", 0, false); er != nil {
		t.Errorf("%s GetDeliveryOrderbook() error %v", g.Name, er)
	}
}

func TestGetDeliveryTradingHistory(t *testing.T) {
	t.Parallel()
	if _, er := g.GetDeliveryTradingHistory(context.Background(), "usdt", "BTC_USDT_20220902", 0, "", time.Time{}, time.Time{}); er != nil {
		t.Errorf("%s GetDeliveryTradingHistory() error %v", g.Name, er)
	}
}
func TestGetDeliveryFuturesCandlesticks(t *testing.T) {
	t.Parallel()
	if _, er := g.GetDeliveryFuturesCandlesticks(context.Background(), "usdt", "BTC_USDT_20220902", time.Time{}, time.Time{}, 0, kline.OneWeek); er != nil {
		t.Errorf("%s GetFuturesCandlesticks() error %v", g.Name, er)
	}
}

func TestGetDeliveryFutureTickers(t *testing.T) {
	t.Parallel()
	if _, er := g.GetDeliveryFutureTickers(context.Background(), "usdt", "BTC_USDT_20220902"); er != nil {
		t.Errorf("%s GetDeliveryFutureTickers() error %v", g.Name, er)
	}
}

func TestGetDeliveryInsuranceBalanceHistory(t *testing.T) {
	t.Parallel()
	if _, er := g.GetDeliveryInsuranceBalanceHistory(context.Background(), "btc", 0); er != nil {
		t.Errorf("%s GetDeliveryInsuranceBalanceHistory() error %v", g.Name, er)
	}
}

func TestGetAllUnderlyings(t *testing.T) {
	t.Parallel()
	if _, er := g.GetAllUnderlyings(context.Background()); er != nil {
		t.Errorf("%s GetAllUnderlyings() error %v", g.Name, er)
	}
}

func TestGetExpirationTime(t *testing.T) {
	t.Parallel()
	if _, er := g.GetExpirationTime(context.Background(), "BTC_USDT"); er != nil {
		t.Errorf("%s GetExpirationTime() error %v", g.Name, er)
	}
}

func TestGetAllContractOfUnderlyingWithinExpiryDate(t *testing.T) {
	t.Parallel()
	if contr, er := g.GetAllContractOfUnderlyingWithinExpiryDate(context.Background(), "BTC_USDT", time.Time{}); er != nil {
		t.Errorf("%s GetAllContractOfUnderlyingWithinExpiryDate() error %v", g.Name, er)
	} else {
		for x := range contr {
			println(contr[x].Name)
		}
	}
}

func TestGetSpecifiedContractDetail(t *testing.T) {
	t.Parallel()
	if _, er := g.GetSpecifiedContractDetail(context.Background(), "BTC_USDT-20220826-35000-P"); er != nil {
		t.Errorf("%s GetSpecifiedContractDetail() error %v", g.Name, er)
	}
}

func TestGetSettlementHistory(t *testing.T) {
	t.Parallel()
	if _, er := g.GetSettlementHistory(context.Background(), "BTC_USDT", 0, 0, time.Time{}, time.Time{}); er != nil {
		t.Errorf("%s GetSettlementHistory() error %v", g.Name, er)
	}
}

func TestGetSpecifiedSettlementHistory(t *testing.T) {
	t.Parallel()
	if _, er := g.GetSpecifiedSettlementHistory(context.Background(), "BTC_USDT-20220819-26000-P", "BTC_USDT", 0); er != nil {
		t.Errorf("%s GetSpecifiedSettlementHistory() error %s", g.Name, er)
	}
}

func TestGetSupportedFlashSwapCurrencies(t *testing.T) {
	t.Parallel()
	if _, er := g.GetSupportedFlashSwapCurrencies(context.Background()); er != nil {
		t.Errorf("%s GetSupportedFlashSwapCurrencies() error %v", g.Name, er)
	}
}

var flashSwapOrderResponseJSON = `{"id": 54646,  "create_time": 1651116876378,  "update_time": 1651116876378,  "user_id": 11135567,  "sell_currency": "BTC",  "sell_amount": "0.01",  "buy_currency": "USDT",  "buy_amount": "10",  "price": "100",  "status": 1}`

func TestCreateFlashSwapOrder(t *testing.T) {
	t.Parallel()
	var response FlashSwapOrderResponse
	if err := json.Unmarshal([]byte(flashSwapOrderResponseJSON), &response); err != nil {
		t.Errorf("%s error while deserializing to FlashSwapOrderResponse %v", g.Name, err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.CreateFlashSwapOrder(context.Background(), FlashSwapOrderParams{
		PreviewID:    "1234",
		SellCurrency: currency.USDT,
		BuyCurrency:  currency.BTC,
		BuyAmount:    34234,
		SellAmount:   34234,
	}); err != nil && !strings.Contains(err.Error(), "The result of preview is expired") {
		t.Errorf("%s CreateFlashSwapOrder() error %v", g.Name, err)
	}
}

func TestGetAllFlashSwapOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.GetAllFlashSwapOrders(context.Background(), 1, currency.EMPTYCODE, currency.EMPTYCODE, true, 0, 0); err != nil {
		t.Errorf("%s GetAllFlashSwapOrders() error %v", g.Name, err)
	}
}

func TestGetSingleFlashSwapOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.GetSingleFlashSwapOrder(context.Background(), "1234"); err != nil {
		t.Errorf("%s GetSingleFlashSwapOrder() error %v", g.Name, err)
	}
}

func TestInitiateFlashSwapOrderReview(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := g.InitiateFlashSwapOrderReview(context.Background(), FlashSwapOrderParams{
		PreviewID:    "1234",
		SellCurrency: currency.USDT,
		BuyCurrency:  currency.BTC,
		SellAmount:   100,
	}); err != nil && !strings.Contains(err.Error(), "The result of preview is expired") {
		t.Errorf("%s InitiateFlashSwapOrderReview() error %v", g.Name, err)
	}
}

func TestGetMyOptionsSettlements(t *testing.T) {
	t.Parallel()
	if _, er := g.GetMyOptionsSettlements(context.Background(), "BTC_USDT", "", 0, 0, time.Time{}); er != nil {
		t.Errorf("%s GetMyOptionsSettlements() error %v", g.Name, er)
	}
}

func TestGetOptionAccounts(t *testing.T) {
	t.Parallel()
	if _, er := g.GetOptionAccounts(context.Background()); er != nil && !strings.Contains(er.Error(), "USER_NOT_FOUND") {
		t.Errorf("%s GetOptionAccounts() error %v", g.Name, er)
	} else if er != nil {
		t.Skipf("%s GetOptionAccounts() user has no futures account", g.Name)
	}
}

var accountChangingHistory = `{"time": 1636426005,"change": "-0.16","balance": "7378.189","text": "BTC_USDT-20211216-5000-P:25","type": "fee"}`

func TestGetAccountChangingHistory(t *testing.T) {
	t.Parallel()
	var accountBook AccountBook
	if er := json.Unmarshal([]byte(accountChangingHistory), &accountBook); er != nil {
		t.Errorf("%s error while deserializing to AccounBook %v", g.Name, er)
	}
	if _, er := g.GetAccountChangingHistory(context.Background(), 0, 0, time.Time{}, time.Time{}, ""); er != nil {
		t.Errorf("%s GetAccountChangingHistory() error %v", g.Name, er)
	}
}

var userUnderlyingPosition = `{"user": 11027586,"contract": "BTC_USDT-20211216-5000-P","size": 10,"entry_price": "1234","realised_pnl": "120","mark_price": "6000","unrealised_pnl": "-320","pending_orders": 1,"close_order": {  "id": 232323,  "price": "5779",  "is_liq": false}}`

func TestGetUsersPositionSpecifiedUnderlying(t *testing.T) {
	t.Parallel()
	var resp UsersPositionForUnderlying
	if er := json.Unmarshal([]byte(userUnderlyingPosition), &resp); er != nil {
		t.Errorf("%s error while decerializing to UsersPositionForUnderlying instance %v", g.Name, er)
	}
	if _, er := g.GetUsersPositionSpecifiedUnderlying(context.Background(), ""); er != nil {
		t.Errorf("%s GetUsersPositionSpecifiedUnderlying() error %v", g.Name, er)
	}
}

func TestGetSpecifiedContractPosition(t *testing.T) {
	t.Parallel()
	_, er := g.GetSpecifiedContractPosition(context.Background(), "")
	if er != nil && !errors.Is(er, errInvalidOrMissingContractParam) {
		t.Errorf("%s GetSpecifiedContractPosition() error expecting %v, but found %v", g.Name, errInvalidOrMissingContractParam, er)
	}
	_, er = g.GetSpecifiedContractPosition(context.Background(), "BTC_USDT-20220826-32000-C")
	if er != nil {
		t.Errorf("%s GetSpecifiedContractPosition() error expecting %v, but found %v", g.Name, errInvalidOrMissingContractParam, er)
	}
}

var optionsClosePositionData = `{"time": 1631764800,"pnl": "-42914.291","settle_size": "-10001","side": "short","contract": "BTC_USDT-20210916-5000-C","text": "settled"}`

func TestGetUsersLiquidationHistoryForSpecifiedUnderlying(t *testing.T) {
	t.Parallel()
	var response ContractClosePosition
	er := json.Unmarshal([]byte(optionsClosePositionData), &response)
	if er != nil {
		t.Errorf("%s error while deserializes ContractClosePosition %v", g.Name, er)
	}
	if _, er := g.GetUsersLiquidationHistoryForSpecifiedUnderlying(context.Background(), "BTC_USDT", ""); er != nil {
		t.Errorf("%s GetUsersLiquidationHistoryForSpecifiedUnderlying() error %v", g.Name, er)
	}
}

var optionOrderJSON = `{"status": "finished","size": -1,"id": 2,"iceberg": 0,"is_liq": false,"is_close": false,"contract": "BTC_USDT-20210916-5000-C","text": "-","fill_price": "100","finish_as": "filled","left": 0,"tif": "gtc","is_reduce_only": false,"create_time": 1631763361,"finish_time": 1631763397,"price": "100"}`

func TestPlaceOptionOrder(t *testing.T) {
	t.Parallel()
	var optionOrderResponse OptionOrderResponse
	er := json.Unmarshal([]byte(optionOrderJSON), &optionOrderResponse)
	if er != nil {
		t.Errorf("%s error while deserializing to OptionOrderResponse %v", g.Name, er)
	}
	if _, er = g.PlaceOptionOrder(context.Background(), OptionOrderParam{
		Contract:    "BTC_USDT-20220902-18000-P",
		OrderSize:   -1,
		Iceberg:     0,
		Text:        "-",
		TimeInForce: "gtc",
		Price:       100,
	}); er != nil {
		t.Errorf("%s PlaceOptionOrder() error %v", g.Name, er)
	}
}

func TestGetOptionFuturesOrders(t *testing.T) {
	t.Parallel()
	if _, er := g.GetOptionFuturesOrders(context.Background(), "", "", "", 0, 0, time.Time{}, time.Time{}); er != nil {
		t.Errorf("%s GetOptionFuturesOrders() error %v", g.Name, er)
	}
}

func TestCancelOptionOpenOrders(t *testing.T) {
	t.Parallel()
	if _, er := g.CancelOptionOpenOrders(context.Background(), "", "", ""); er != nil {
		t.Errorf("%s CancelOptionOpenOrders() error %v", g.Name, er)
	}
}
func TestGetSingleOptionorder(t *testing.T) {
	t.Parallel()
	if _, er := g.GetSingleOptionorder(context.Background(), ""); er != nil && !errors.Is(errInvalidOrderID, er) {
		t.Errorf("%s GetSingleOptionorder() expecting %v, but found %v", g.Name, errInvalidOrderID, er)
	}
	if _, er := g.GetSingleOptionorder(context.Background(), "1234"); er != nil {
		t.Errorf("%s GetSingleOptionOrder() error %v", g.Name, er)
	}
}

func TestCancelSingleOrder(t *testing.T) {
	t.Parallel()
	if _, er := g.CancelOptionSingleOrder(context.Background(), "1234"); er != nil && !strings.Contains(er.Error(), "ORDER_NOT_FOUND") {
		t.Errorf("%s CancelSingleOrder() error %v", g.Name, er)
	}
}

func TestGetOptionsPersonalTradingHistory(t *testing.T) {
	t.Parallel()
	if _, er := g.GetOptionsPersonalTradingHistory(context.Background(), "BTC_USDT", "", 0, 0, time.Time{}, time.Time{}); er != nil {
		t.Errorf("%s GetOptionPersonalTradingHistory() error %v", g.Name, er)
	}
}

func TestWithdrawCurrency(t *testing.T) {
	t.Parallel()
	_, er := g.WithdrawCurrency(context.Background(), WithdrawalRequestParam{})
	if er != nil && !errors.Is(er, errInvalidAmount) {
		t.Errorf("%s WithdrawCurrency() expecting error %v, but found %v", g.Name, errInvalidAmount, er)
	}
	_, er = g.WithdrawCurrency(context.Background(), WithdrawalRequestParam{
		Currency: currency.BTC,
		Amount:   0.00000001,
		Address:  "bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc",
	})
	if er != nil {
		t.Errorf("%s WithdrawCurrency() expecting error %v, but found %v", g.Name, errInvalidAmount, er)
	}
}

func TestCancelWithdrawalWithSpecifiedID(t *testing.T) {
	t.Parallel()
	if _, er := g.CancelWithdrawalWithSpecifiedID(context.Background(), "1234567"); er != nil {
		t.Errorf("%s CancelWithdrawalWithSpecifiedID() error %v", g.Name, er)
	}
}

func TestGetOptionsFuturesOrderbooks(t *testing.T) {
	t.Parallel()
	if _, er := g.GetOptionsFuturesOrderbooks(context.Background(), "BTC_USDT-20220826-32000-C", "0.1", 9, true); er != nil {
		t.Errorf("%s GetOptionsFuturesOrderbooks() error %v", g.Name, er)
	}
}

func TestGetOptionsTickers(t *testing.T) {
	t.Parallel()
	if _, er := g.GetOptionsTickers(context.Background(), "BTC_USDT"); er != nil {
		t.Errorf("%s GetOptionsTickers() error %v", g.Name, er)
	}
}

func TestGetOptionUnderlyingTickers(t *testing.T) {
	t.Parallel()
	if _, er := g.GetOptionUnderlyingTickers(context.Background(), "BTC_USDT"); er != nil {
		t.Errorf("%s GetOptionUnderlyingTickers() error %v", g.Name, er)
	}
}

func TestGetOptionFuturesCandlesticks(t *testing.T) {
	t.Parallel()
	if _, er := g.GetOptionFuturesCandlesticks(context.Background(), "BTC_USDT-20220826-32000-C", 0, time.Time{}, time.Time{}, kline.OneMonth); er != nil {
		t.Errorf("%s GetOptionFuturesCandlesticks() error %v", g.Name, er)
	}
}

func TestGetOptionFuturesMarkPriceCandlesticks(t *testing.T) {
	t.Parallel()
	if _, er := g.GetOptionFuturesMarkPriceCandlesticks(context.Background(), "BTC_USDT", 0, time.Time{}, time.Time{}, kline.OneMonth); er != nil {
		t.Errorf("%s GetOptionFuturesMarkPriceCandlesticks() error %v", g.Name, er)
	}
}

func TestGetOptionsTradeHistory(t *testing.T) {
	t.Parallel()
	if _, er := g.GetOptionsTradeHistory(context.Background(), "BTC_USDT-20220826-32000-C", "C", 0, 0, time.Time{}, time.Time{}); er != nil {
		t.Errorf("%s GetOptionsTradeHistory() error %v", g.Name, er)
	}
}
