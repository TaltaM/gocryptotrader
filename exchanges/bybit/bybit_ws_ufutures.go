package bybit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// WsContractXConnect connects to a websocket feed
func (by *Bybit) WsContractXConnect(a asset.Item) error {
	if !by.Websocket.IsEnabled() || !by.IsEnabled() || !by.IsAssetWebsocketSupported(a) {
		return errors.New(stream.WebsocketNotEnabled)
	}
	assetWebsocket, err := by.Websocket.GetAssetWebsocket(a)
	if err != nil {
		return fmt.Errorf("%w asset type: %v", err, a)
	}
	var dialer websocket.Dialer
	err = assetWebsocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}

	pingMsg, err := json.Marshal(pingRequest)
	if err != nil {
		return err
	}
	assetWebsocket.Conn.SetupPingHandler(stream.PingHandler{
		Message:     pingMsg,
		MessageType: websocket.PingMessage,
		Delay:       bybitWebsocketTimer,
	})
	if by.Verbose {
		log.Debugf(log.ExchangeSys, "%s Connected to %v Websocket.\n", by.Name, a)
	}

	go by.wsContractReadData(assetWebsocket.Conn, a)
	by.Websocket.SetCanUseAuthenticatedEndpoints(true, a)
	if by.Websocket.CanUseAuthenticatedEndpoints() {
		err = by.WsContractXAuth(context.TODO(), &dialer, a)
		if err != nil {
			by.Websocket.DataHandler <- err
			by.Websocket.SetCanUseAuthenticatedEndpoints(false, a)
		}
	}
	return nil
}

// wsContractXReadData read coming messages thought the websocket connection and process the data.
func (by *Bybit) wsContractXReadData(wsConn stream.Connection, a asset.Item) {
	defer by.Websocket.Wg.Done()
	assetWebsocket, err := by.Websocket.GetAssetWebsocket(a)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%v asset type: %v", err, a)
		return
	}
	assetWebsocket.Wg.Add(1)
	defer assetWebsocket.Wg.Done()
	for {
		select {
		case <-assetWebsocket.ShutdownC:
			return
		default:
			resp := wsConn.ReadMessage()
			if resp.Raw == nil {
				return
			}

			err := by.wsContractXHandleData(resp.Raw, a)
			if err != nil {
				by.Websocket.DataHandler <- err
			}
		}
	}
}

// WsContractXAuth sends an authentication message to receive auth data
func (by *Bybit) WsContractXAuth(ctx context.Context, dialer *websocket.Dialer, a asset.Item) error {
	assetWebsocket, err := by.Websocket.GetAssetWebsocket(a)
	if err != nil {
		return fmt.Errorf("%w asset type: %v", err, a)
	}
	creds, err := by.GetCredentials(ctx)
	if err != nil {
		return err
	}

	err = assetWebsocket.AuthConn.Dial(dialer, http.Header{})
	if err != nil {
		return err
	}
	go by.wsContractReadData(assetWebsocket.AuthConn, a)
	intNonce := (time.Now().Unix() + 1) * 1000
	strNonce := strconv.FormatInt(intNonce, 10)
	hmac, err := crypto.GetHMAC(
		crypto.HashSHA256,
		[]byte("GET/realtime"+strNonce),
		[]byte(creds.Secret),
	)
	if err != nil {
		return err
	}
	sign := crypto.HexEncodeToString(hmac)
	req := Authenticate{
		Operation: "auth",
		Args:      []interface{}{creds.Key, intNonce, sign},
	}
	return assetWebsocket.AuthConn.SendJSONMessage(req)
}

// GenerateWsContractXDefaultSubscriptions returns channel subscriptions for futures instruments
func (by *Bybit) GenerateWsContractXDefaultSubscriptions(a asset.Item) ([]stream.ChannelSubscription, error) {
	channels := defaultSubscriptionChannels[a]
	if by.Websocket.CanUseAuthenticatedEndpoints() {
		channels = append(channels,
			wsWallet,
			wsOrder,
			wsStopOrder)
	}
	subscriptions := []stream.ChannelSubscription{}
	contractPairs, err := by.GetEnabledPairs(a)
	if err != nil {
		return nil, err
	}
	contractPairFormat, err := by.GetPairFormat(a, true)
	if err != nil {
		return nil, err
	}
	contractPairs = contractPairs.Format(contractPairFormat)
	for x := range channels {
		switch channels[x] {
		case wsInsurance, wsLiquidation, wsPosition,
			wsExecution, wsOrder, wsStopOrder, wsWallet:
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Asset:   a,
				Channel: channels[x],
			})
		case wsOrder25, wsTrade:
			for p := range contractPairs {
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Asset:    a,
					Channel:  channels[x],
					Currency: contractPairs[p],
				})
			}
		case wsKlineV2, wsUSDTKline:
			for p := range contractPairs {
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Asset:    a,
					Channel:  channels[x],
					Currency: contractPairs[p],
					Params: map[string]interface{}{
						"interval": "1",
					},
				})
			}
		case wsInstrument, wsOrder200:
			for p := range contractPairs {
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Asset:    a,
					Channel:  channels[x],
					Currency: contractPairs[p],
					Params: map[string]interface{}{
						"frequency_interval": "100ms",
					},
				})
			}
		}
	}
	return subscriptions, nil
}

// SubscribeWsContractX sends a websocket message to receive data from the channel
func (by *Bybit) SubscribeWsContractX(channelsToSubscribe []stream.ChannelSubscription, a asset.Item) error {
	assetWebsocket, err := by.Websocket.GetAssetWebsocket(a)
	if err != nil {
		return fmt.Errorf("%w asset type: %v", err, a)
	}
	var errs error
	for i := range channelsToSubscribe {
		var sub WsFuturesReq
		sub.Topic = wsSubscribe

		argStr := formatArgs(channelsToSubscribe[i].Channel, channelsToSubscribe[i].Params)
		switch channelsToSubscribe[i].Channel {
		case wsOrder25, wsKlineV2, wsUSDTKline, wsInstrument, wsOrder200, wsTrade:
			var formattedPair currency.Pair
			formattedPair, err = by.FormatExchangeCurrency(channelsToSubscribe[i].Currency, channelsToSubscribe[i].Asset)
			if err != nil {
				errs = common.AppendError(errs, err)
				continue
			}
			argStr += dot + formattedPair.String()
		}
		sub.Args = append(sub.Args, argStr)

		err = assetWebsocket.Conn.SendJSONMessage(sub)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		assetWebsocket.AddSuccessfulSubscriptions(channelsToSubscribe[i])
	}
	return errs
}

// UnsubscribeWsContractX sends a websocket message to stop receiving data from the channel
func (by *Bybit) UnsubscribeWsContractX(channelsToUnsubscribe []stream.ChannelSubscription, a asset.Item) error {
	assetWebsocket, err := by.Websocket.GetAssetWebsocket(a)
	if err != nil {
		return fmt.Errorf("%w asset type: %v", err, a)
	}
	var errs error
	for i := range channelsToUnsubscribe {
		var unSub WsFuturesReq
		unSub.Topic = wsUnsubscribe

		formattedPair, err := by.FormatExchangeCurrency(channelsToUnsubscribe[i].Currency, a)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		unSub.Args = append(unSub.Args, channelsToUnsubscribe[i].Channel+dot+formattedPair.String())
		err = assetWebsocket.Conn.SendJSONMessage(unSub)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		assetWebsocket.RemoveSuccessfulUnsubscriptions(channelsToUnsubscribe[i])
	}
	return errs
}

func (by *Bybit) wsContractXHandleData(respRaw []byte, a asset.Item) error {
	var multiStreamData map[string]interface{}
	err := json.Unmarshal(respRaw, &multiStreamData)
	if err != nil {
		return err
	}

	t, ok := multiStreamData["topic"].(string)
	if !ok {
		if by.Verbose {
			log.Warnf(log.ExchangeSys, "%s Asset Type %v Received unhandled message on websocket: %v\n", by.Name, a, multiStreamData)
		}
		return nil
	}

	topics := strings.Split(t, dot)
	if len(topics) < 1 {
		return errors.New(by.Name + " - topic could not be extracted from response")
	}

	switch topics[0] {
	case wsOrder25, wsOrder200:
		if wsType, ok := multiStreamData["type"].(string); ok {
			switch wsType {
			case wsOperationSnapshot:
				var response WsFuturesOrderbook
				err = json.Unmarshal(respRaw, &response)
				if err != nil {
					return err
				}

				var p currency.Pair
				p, err = by.extractCurrencyPair(response.Data[0].Symbol, a)
				if err != nil {
					return err
				}

				err = by.processOrderbook(response.Data,
					wsOperationSnapshot,
					p,
					a)
				if err != nil {
					return err
				}

			case wsOperationDelta:
				var response WsFuturesDeltaOrderbook
				err = json.Unmarshal(respRaw, &response)
				if err != nil {
					return err
				}

				if len(response.OBData.Delete) > 0 {
					var p currency.Pair
					p, err = by.extractCurrencyPair(response.OBData.Delete[0].Symbol, a)
					if err != nil {
						return err
					}

					err = by.processOrderbook(response.OBData.Delete,
						wsOrderbookActionDelete,
						p,
						a)
					if err != nil {
						return err
					}
				}

				if len(response.OBData.Update) > 0 {
					var p currency.Pair
					p, err = by.extractCurrencyPair(response.OBData.Update[0].Symbol, a)
					if err != nil {
						return err
					}

					err = by.processOrderbook(response.OBData.Update,
						wsOrderbookActionUpdate,
						p,
						a)
					if err != nil {
						return err
					}
				}

				if len(response.OBData.Insert) > 0 {
					var p currency.Pair
					p, err = by.extractCurrencyPair(response.OBData.Insert[0].Symbol, a)
					if err != nil {
						return err
					}

					err = by.processOrderbook(response.OBData.Insert,
						wsOrderbookActionInsert,
						p,
						a)
					if err != nil {
						return err
					}
				}
			default:
				by.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: by.Name + stream.UnhandledMessage + "unsupported orderbook operation"}
			}
		}

	case wsTrades:
		if !by.IsSaveTradeDataEnabled() {
			return nil
		}
		var response WsFuturesTrade
		err = json.Unmarshal(respRaw, &response)
		if err != nil {
			return err
		}
		counter := 0
		trades := make([]trade.Data, len(response.TradeData))
		for i := range response.TradeData {
			var p currency.Pair
			p, err = by.extractCurrencyPair(response.TradeData[0].Symbol, a)
			if err != nil {
				return err
			}

			var oSide order.Side
			oSide, err = order.StringToOrderSide(response.TradeData[i].Side)
			if err != nil {
				by.Websocket.DataHandler <- order.ClassificationError{
					Exchange: by.Name,
					Err:      err,
				}
				continue
			}

			trades[counter] = trade.Data{
				TID:          response.TradeData[i].ID,
				Exchange:     by.Name,
				CurrencyPair: p,
				AssetType:    a,
				Side:         oSide,
				Price:        response.TradeData[i].Price.Float64(),
				Amount:       response.TradeData[i].Size,
				Timestamp:    response.TradeData[i].Time,
			}
			counter++
		}
		return by.AddTradesToBuffer(trades...)

	case wsKlineV2, wsUSDTKline:
		var response WsFuturesKline
		err = json.Unmarshal(respRaw, &response)
		if err != nil {
			return err
		}

		var p currency.Pair
		p, err = by.extractCurrencyPair(topics[len(topics)-1], a)
		if err != nil {
			return err
		}

		for i := range response.KlineData {
			by.Websocket.DataHandler <- stream.KlineData{
				Pair:       p,
				AssetType:  a,
				Exchange:   by.Name,
				OpenPrice:  response.KlineData[i].Open.Float64(),
				HighPrice:  response.KlineData[i].High.Float64(),
				LowPrice:   response.KlineData[i].Low.Float64(),
				ClosePrice: response.KlineData[i].Close.Float64(),
				Volume:     response.KlineData[i].Volume.Float64(),
				Timestamp:  response.KlineData[i].Timestamp.Time(),
			}
		}

	case wsInstrument:
		if wsType, ok := multiStreamData["type"].(string); ok {
			switch wsType {
			case wsOperationSnapshot:
				var response WsTicker
				err = json.Unmarshal(respRaw, &response)
				if err != nil {
					return err
				}

				var p currency.Pair
				p, err = by.extractCurrencyPair(response.Ticker.Symbol, a)
				if err != nil {
					return err
				}

				by.Websocket.DataHandler <- &ticker.Price{
					ExchangeName: by.Name,
					Last:         response.Ticker.LastPrice.Float64(),
					High:         response.Ticker.HighPrice24h.Float64(),
					Low:          response.Ticker.LowPrice24h.Float64(),
					Bid:          response.Ticker.BidPrice.Float64(),
					Ask:          response.Ticker.AskPrice.Float64(),
					Volume:       response.Ticker.Volume24h.Float64(),
					Close:        response.Ticker.PrevPrice24h.Float64(),
					LastUpdated:  response.Ticker.UpdateAt,
					AssetType:    a,
					Pair:         p,
				}

			case wsOperationDelta:
				var response WsDeltaTicker
				err = json.Unmarshal(respRaw, &response)
				if err != nil {
					return err
				}

				if len(response.Data.Delete) > 0 {
					for x := range response.Data.Delete {
						var p currency.Pair
						p, err = by.extractCurrencyPair(response.Data.Delete[x].Symbol, a)
						if err != nil {
							return err
						}
						var tickerData *ticker.Price
						tickerData, err = by.FetchTicker(context.Background(), p, a)
						if err != nil {
							return err
						}
						by.Websocket.DataHandler <- &ticker.Price{
							ExchangeName: by.Name,
							Last:         compareAndSet(tickerData.Last, response.Data.Delete[x].LastPrice.Float64()),
							High:         compareAndSet(tickerData.High, response.Data.Delete[x].HighPrice24h.Float64()),
							Low:          compareAndSet(tickerData.Low, response.Data.Delete[x].LowPrice24h.Float64()),
							Bid:          compareAndSet(tickerData.Bid, response.Data.Delete[x].BidPrice.Float64()),
							Ask:          compareAndSet(tickerData.Ask, response.Data.Delete[x].AskPrice.Float64()),
							Volume:       compareAndSet(tickerData.Volume, response.Data.Delete[x].Volume24h.Float64()),
							Close:        compareAndSet(tickerData.Close, response.Data.Delete[x].PrevPrice24h.Float64()),
							LastUpdated:  response.Data.Delete[x].UpdateAt,
							AssetType:    a,
							Pair:         p,
						}
					}
				}

				if len(response.Data.Update) > 0 {
					for x := range response.Data.Update {
						if response.Data.Update[x] == (WsTickerData{}) {
							continue
						}
						var p currency.Pair
						p, err = by.extractCurrencyPair(response.Data.Update[x].Symbol, a)
						if err != nil {
							return err
						}
						var tickerData *ticker.Price
						tickerData, err = by.FetchTicker(context.Background(), p, a)
						if err != nil {
							return err
						}
						by.Websocket.DataHandler <- &ticker.Price{
							ExchangeName: by.Name,
							Last:         compareAndSet(tickerData.Last, response.Data.Update[x].LastPrice.Float64()),
							High:         compareAndSet(tickerData.High, response.Data.Update[x].HighPrice24h.Float64()),
							Low:          compareAndSet(tickerData.Low, response.Data.Update[x].LowPrice24h.Float64()),
							Bid:          compareAndSet(tickerData.Bid, response.Data.Update[x].BidPrice.Float64()),
							Ask:          compareAndSet(tickerData.Ask, response.Data.Update[x].AskPrice.Float64()),
							Volume:       compareAndSet(tickerData.Volume, response.Data.Update[x].Volume24h.Float64()),
							Close:        compareAndSet(tickerData.Close, response.Data.Update[x].PrevPrice24h.Float64()),
							LastUpdated:  response.Data.Update[x].UpdateAt,
							AssetType:    a,
							Pair:         p,
						}
					}
				}

				if len(response.Data.Insert) > 0 {
					for x := range response.Data.Insert {
						var p currency.Pair
						p, err = by.extractCurrencyPair(response.Data.Insert[x].Symbol, a)
						if err != nil {
							return err
						}

						by.Websocket.DataHandler <- &ticker.Price{
							ExchangeName: by.Name,
							Last:         response.Data.Insert[x].LastPrice.Float64(),
							High:         response.Data.Insert[x].HighPrice24h.Float64(),
							Low:          response.Data.Insert[x].LowPrice24h.Float64(),
							Bid:          response.Data.Insert[x].BidPrice.Float64(),
							Ask:          response.Data.Insert[x].AskPrice.Float64(),
							Volume:       response.Data.Insert[x].Volume24h.Float64(),
							Close:        response.Data.Insert[x].PrevPrice24h.Float64(),
							LastUpdated:  response.Data.Insert[x].UpdateAt,
							AssetType:    a,
							Pair:         p,
						}
					}
				}

			default:
				by.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: by.Name + stream.UnhandledMessage + "unsupported ticker operation"}
			}
		}

	case wsLiquidation:
		var response WsFuturesLiquidation
		err = json.Unmarshal(respRaw, &response)
		if err != nil {
			return err
		}
		by.Websocket.DataHandler <- response.Data

	case wsPosition:
		var response WsFuturesPosition
		err = json.Unmarshal(respRaw, &response)
		if err != nil {
			return err
		}
		by.Websocket.DataHandler <- response.Data

	case wsExecution:
		var response WsFuturesExecution
		err = json.Unmarshal(respRaw, &response)
		if err != nil {
			return err
		}

		for i := range response.Data {
			var p currency.Pair
			p, err = by.extractCurrencyPair(response.Data[i].Symbol, a)
			if err != nil {
				return err
			}

			var oSide order.Side
			oSide, err = order.StringToOrderSide(response.Data[i].Side)
			if err != nil {
				by.Websocket.DataHandler <- order.ClassificationError{
					Exchange: by.Name,
					OrderID:  response.Data[i].OrderID,
					Err:      err,
				}
			}

			var oStatus order.Status
			oStatus, err = order.StringToOrderStatus(response.Data[i].ExecutionType)
			if err != nil {
				by.Websocket.DataHandler <- order.ClassificationError{
					Exchange: by.Name,
					OrderID:  response.Data[i].OrderID,
					Err:      err,
				}
			}

			by.Websocket.DataHandler <- &order.Detail{
				Exchange:  by.Name,
				OrderID:   response.Data[i].OrderID,
				AssetType: a,
				Pair:      p,
				Price:     response.Data[i].Price.Float64(),
				Amount:    response.Data[i].OrderQty,
				Side:      oSide,
				Status:    oStatus,
				Trades: []order.TradeHistory{
					{
						Price:     response.Data[i].Price.Float64(),
						Amount:    response.Data[i].OrderQty,
						Exchange:  by.Name,
						Side:      oSide,
						Timestamp: response.Data[i].Time,
						TID:       response.Data[i].ExecutionID,
						IsMaker:   response.Data[i].IsMaker,
					},
				},
			}
		}

	case wsOrder:
		var response WsOrder
		err = json.Unmarshal(respRaw, &response)
		if err != nil {
			return err
		}
		for x := range response.Data {
			var p currency.Pair
			p, err = by.extractCurrencyPair(response.Data[x].Symbol, a)
			if err != nil {
				return err
			}
			var oSide order.Side
			oSide, err = order.StringToOrderSide(response.Data[x].Side)
			if err != nil {
				by.Websocket.DataHandler <- order.ClassificationError{
					Exchange: by.Name,
					OrderID:  response.Data[x].OrderID,
					Err:      err,
				}
			}
			var oType order.Type
			oType, err = order.StringToOrderType(response.Data[x].OrderType)
			if err != nil {
				by.Websocket.DataHandler <- order.ClassificationError{
					Exchange: by.Name,
					OrderID:  response.Data[x].OrderID,
					Err:      err,
				}
			}
			var oStatus order.Status
			oStatus, err = order.StringToOrderStatus(response.Data[x].OrderStatus)
			if err != nil {
				by.Websocket.DataHandler <- order.ClassificationError{
					Exchange: by.Name,
					OrderID:  response.Data[x].OrderID,
					Err:      err,
				}
			}
			by.Websocket.DataHandler <- &order.Detail{
				Price:     response.Data[x].Price.Float64(),
				Amount:    response.Data[x].OrderQty,
				Exchange:  by.Name,
				OrderID:   response.Data[x].OrderID,
				Type:      oType,
				Side:      oSide,
				Status:    oStatus,
				AssetType: a,
				Date:      response.Data[x].GetTime(a),
				Pair:      p,
				Trades: []order.TradeHistory{
					{
						Price:     response.Data[x].Price.Float64(),
						Amount:    response.Data[x].OrderQty,
						Exchange:  by.Name,
						Side:      oSide,
						Timestamp: response.Data[x].GetTime(a),
					},
				},
			}
		}

	case wsStopOrder:
		var response WsFuturesStopOrder
		err = json.Unmarshal(respRaw, &response)
		if err != nil {
			return err
		}
		for x := range response.Data {
			var p currency.Pair
			p, err = by.extractCurrencyPair(response.Data[x].Symbol, a)
			if err != nil {
				return err
			}
			var oSide order.Side
			oSide, err = order.StringToOrderSide(response.Data[x].Side)
			if err != nil {
				by.Websocket.DataHandler <- order.ClassificationError{
					Exchange: by.Name,
					OrderID:  response.Data[x].OrderID,
					Err:      err,
				}
			}
			var oType order.Type
			oType, err = order.StringToOrderType(response.Data[x].OrderType)
			if err != nil {
				by.Websocket.DataHandler <- order.ClassificationError{
					Exchange: by.Name,
					OrderID:  response.Data[x].OrderID,
					Err:      err,
				}
			}
			var oStatus order.Status
			oStatus, err = order.StringToOrderStatus(response.Data[x].OrderStatus)
			if err != nil {
				by.Websocket.DataHandler <- order.ClassificationError{
					Exchange: by.Name,
					OrderID:  response.Data[x].OrderID,
					Err:      err,
				}
			}
			by.Websocket.DataHandler <- &order.Detail{
				Price:     response.Data[x].Price.Float64(),
				Amount:    response.Data[x].OrderQty,
				Exchange:  by.Name,
				OrderID:   response.Data[x].OrderID,
				AccountID: strconv.FormatInt(response.Data[x].UserID, 10),
				Type:      oType,
				Side:      oSide,
				Status:    oStatus,
				AssetType: a,
				Date:      response.Data[x].GetTime(a),
				Pair:      p,
				Trades: []order.TradeHistory{
					{
						Price:     response.Data[x].Price.Float64(),
						Amount:    response.Data[x].OrderQty,
						Exchange:  by.Name,
						Side:      oSide,
						Timestamp: response.Data[x].GetTime(a),
					},
				},
			}
		}

	case wsWallet:
		var response WsFuturesWallet
		err = json.Unmarshal(respRaw, &response)
		if err != nil {
			return err
		}
		by.Websocket.DataHandler <- response.Data

	default:
		by.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: by.Name + stream.UnhandledMessage + string(respRaw)}
	}

	return nil
}

func compareAndSet(prevVal, newVal float64) float64 {
	if newVal != 0 {
		return newVal
	}
	return prevVal
}
