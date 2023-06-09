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

const (
	bybitWebsocketFuturesPublicV2 = "wss://stream.bybit.com/realtime"
)

var defaultFuturesSubscriptionChannels = []string{
	wsInstrument,
	wsOrder200,
	wsTrade,
	wsKlineV2,
}

// WsFuturesConnect connects to a Futures websocket feed
func (by *Bybit) WsFuturesConnect() error {
	if !by.Websocket.IsEnabled() || !by.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	futuresWebsocket, err := by.Websocket.GetAssetWebsocket(asset.Futures)
	if err != nil {
		return err
	}
	var dialer websocket.Dialer
	err = futuresWebsocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}

	pingMsg, err := json.Marshal(pingRequest)
	if err != nil {
		return err
	}
	futuresWebsocket.Conn.SetupPingHandler(stream.PingHandler{
		Message:     pingMsg,
		MessageType: websocket.PingMessage,
		Delay:       bybitWebsocketTimer,
	})
	if by.Verbose {
		log.Debugf(log.ExchangeSys, "%s Connected to Websocket.\n", by.Name)
	}

	go by.wsFuturesReadData()
	if by.IsWebsocketAuthenticationSupported() {
		err = by.WsFuturesAuth(context.TODO())
		if err != nil {
			by.Websocket.DataHandler <- err
			by.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	return nil
}

// GenerateFuturesDefaultSubscriptions returns channel subscriptions for futures instruments
func (by *Bybit) GenerateFuturesDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	channels := defaultFuturesSubscriptionChannels
	if by.Websocket.CanUseAuthenticatedEndpoints() {
		channels = append(channels,
			wsWallet,
			wsOrder,
			wsStopOrder)
	}
	subscriptions := []stream.ChannelSubscription{}
	futuresPairs, err := by.GetEnabledPairs(asset.Futures)
	if err != nil {
		return nil, err
	}
	futuresPairFormat, err := by.GetPairFormat(asset.Futures, true)
	if err != nil {
		return nil, err
	}
	futuresPairs = futuresPairs.Format(futuresPairFormat)
	for x := range channels {
		switch channels[x] {
		case wsTrade, wsInsurance, wsLiquidation, wsPosition,
			wsExecution, wsOrder, wsStopOrder, wsWallet:
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Asset:   asset.Futures,
				Channel: channels[x],
			})
		case wsOrder25:
			for p := range futuresPairs {
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Asset:    asset.Futures,
					Channel:  channels[x],
					Currency: futuresPairs[p],
				})
			}
		case wsKlineV2:
			for p := range futuresPairs {
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Asset:    asset.Futures,
					Channel:  channels[x],
					Currency: futuresPairs[p],
					Params: map[string]interface{}{
						"interval": "1",
					},
				})
			}
		case wsInstrument, wsOrder200:
			for p := range futuresPairs {
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Asset:    asset.Futures,
					Channel:  channels[x],
					Currency: futuresPairs[p],
					Params: map[string]interface{}{
						"frequency_interval": "100ms",
					},
				})
			}
		}
	}
	return subscriptions, nil
}

// WsFuturesAuth sends an authentication message to receive auth data
func (by *Bybit) WsFuturesAuth(ctx context.Context) error {
	futuresWebsocket, err := by.Websocket.GetAssetWebsocket(asset.Futures)
	if err != nil {
		return fmt.Errorf("%w asset type: %v", err, asset.Futures)
	}
	creds, err := by.GetCredentials(ctx)
	if err != nil {
		return err
	}

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
	return futuresWebsocket.Conn.SendJSONMessage(req)
}

// SubscribeFutures sends a websocket message to receive data from the channel
func (by *Bybit) SubscribeFutures(channelsToSubscribe []stream.ChannelSubscription) error {
	futuresWebsocket, err := by.Websocket.GetAssetWebsocket(asset.Futures)
	if err != nil {
		return fmt.Errorf("%w asset type: %v", err, asset.Futures)
	}
	var errs error
	for i := range channelsToSubscribe {
		var sub WsFuturesReq
		sub.Topic = subscribe

		sub.Args = append(sub.Args, formatArgs(channelsToSubscribe[i].Channel, channelsToSubscribe[i].Params))
		switch channelsToSubscribe[i].Channel {
		case wsOrder25, wsKlineV2, wsInstrument, wsOrder200:
			sub.Args[0] += dot + channelsToSubscribe[i].Currency.String()
		}
		err := futuresWebsocket.Conn.SendJSONMessage(sub)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		futuresWebsocket.AddSuccessfulSubscriptions(channelsToSubscribe[i])
	}
	return errs
}

// UnsubscribeFutures sends a websocket message to stop receiving data from the channel
func (by *Bybit) UnsubscribeFutures(channelsToUnsubscribe []stream.ChannelSubscription) error {
	futuresWebsocket, err := by.Websocket.GetAssetWebsocket(asset.Futures)
	if err != nil {
		return fmt.Errorf("%w asset type: %v", err, asset.Futures)
	}
	var errs error
	for i := range channelsToUnsubscribe {
		var unSub WsFuturesReq
		unSub.Topic = unsubscribe

		formattedPair, err := by.FormatExchangeCurrency(channelsToUnsubscribe[i].Currency, asset.Futures)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		unSub.Args = append(unSub.Args, channelsToUnsubscribe[i].Channel+dot+formattedPair.String())
		err = futuresWebsocket.Conn.SendJSONMessage(unSub)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		futuresWebsocket.RemoveSuccessfulUnsubscriptions(channelsToUnsubscribe[i])
	}
	return errs
}

// wsFuturesReadData gets and passes on websocket messages for processing
func (by *Bybit) wsFuturesReadData() {
	futuresWebsocket, err := by.Websocket.GetAssetWebsocket(asset.Futures)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%v asset type: %v", err, asset.Futures)
		return
	}
	by.Websocket.Wg.Add(1)
	defer by.Websocket.Wg.Done()

	for {
		select {
		case <-futuresWebsocket.ShutdownC:
			return
		default:
			resp := futuresWebsocket.Conn.ReadMessage()
			if resp.Raw == nil {
				return
			}

			err := by.wsFuturesHandleData(resp.Raw)
			if err != nil {
				by.Websocket.DataHandler <- err
			}
		}
	}
}

func (by *Bybit) wsFuturesHandleData(respRaw []byte) error {
	var multiStreamData map[string]interface{}
	err := json.Unmarshal(respRaw, &multiStreamData)
	if err != nil {
		return err
	}

	t, ok := multiStreamData["topic"].(string)
	if !ok {
		log.Errorf(log.ExchangeSys, "%s Received unhandle message on websocket: %v\n", by.Name, multiStreamData)
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
				p, err = currency.NewPairFromString(response.OBData[0].Symbol)
				if err != nil {
					return err
				}
				var format currency.PairFormat
				format, err = by.GetPairFormat(asset.Futures, false)
				if err != nil {
					return err
				}
				p = p.Format(format)
				if err != nil {
					return err
				}
				err = by.processOrderbook(response.OBData,
					response.Type,
					p,
					asset.Futures)
				if err != nil {
					return err
				}
			case wsOperationDelta:
				var response WsCoinDeltaOrderbook
				err = json.Unmarshal(respRaw, &response)
				if err != nil {
					return err
				}

				if len(response.OBData.Delete) > 0 {
					var p currency.Pair
					p, err = currency.NewPairFromString(response.OBData.Delete[0].Symbol)
					if err != nil {
						return err
					}
					err = by.processOrderbook(response.OBData.Delete,
						wsOrderbookActionDelete,
						p,
						asset.Futures)
					if err != nil {
						return err
					}
				}

				if len(response.OBData.Update) > 0 {
					var p currency.Pair
					p, err = currency.NewPairFromString(response.OBData.Update[0].Symbol)
					if err != nil {
						return err
					}

					err = by.processOrderbook(response.OBData.Update,
						wsOrderbookActionUpdate,
						p,
						asset.Futures)
					if err != nil {
						return err
					}
				}

				if len(response.OBData.Insert) > 0 {
					var p currency.Pair
					p, err = currency.NewPairFromString(response.OBData.Insert[0].Symbol)
					if err != nil {
						return err
					}

					err = by.processOrderbook(response.OBData.Insert,
						wsOrderbookActionInsert,
						p,
						asset.Futures)
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
			p, err = by.extractCurrencyPair(response.TradeData[0].Symbol, asset.Futures)
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
			}

			trades[counter] = trade.Data{
				TID:          response.TradeData[i].ID,
				Exchange:     by.Name,
				CurrencyPair: p,
				AssetType:    asset.Futures,
				Side:         oSide,
				Price:        response.TradeData[i].Price,
				Amount:       response.TradeData[i].Size,
				Timestamp:    response.TradeData[i].Time,
			}
			counter++
		}
		return by.AddTradesToBuffer(trades...)

	case wsKlineV2:
		var response WsFuturesKline
		err = json.Unmarshal(respRaw, &response)
		if err != nil {
			return err
		}

		var p currency.Pair
		p, err = currency.NewPairFromString(topics[len(topics)-1])
		if err != nil {
			return err
		}

		for i := range response.KlineData {
			by.Websocket.DataHandler <- stream.KlineData{
				Pair:       p,
				AssetType:  asset.Futures,
				Exchange:   by.Name,
				OpenPrice:  response.KlineData[i].Open,
				HighPrice:  response.KlineData[i].High,
				LowPrice:   response.KlineData[i].Low,
				ClosePrice: response.KlineData[i].Close,
				Volume:     response.KlineData[i].Volume.Float64(),
				Timestamp:  response.KlineData[i].Timestamp.Time(),
			}
		}

	case wsInstrument:
		if wsType, ok := multiStreamData["type"].(string); ok {
			switch wsType {
			case wsOperationSnapshot:
				var response WsFuturesTicker
				err = json.Unmarshal(respRaw, &response)
				if err != nil {
					return err
				}
				var p currency.Pair
				p, err = currency.NewPairFromString(response.Ticker.Symbol)
				if err != nil {
					return err
				}
				by.Websocket.DataHandler <- &ticker.Price{
					ExchangeName: by.Name,
					Last:         response.Ticker.LastPrice,
					High:         response.Ticker.HighPrice24h,
					Low:          response.Ticker.LowPrice24h,
					Bid:          response.Ticker.BidPrice,
					Ask:          response.Ticker.AskPrice,
					Volume:       response.Ticker.Volume24h,
					Close:        response.Ticker.PrevPrice24h,
					LastUpdated:  response.Ticker.UpdateAt,
					AssetType:    asset.Futures,
					Pair:         p,
				}

			case wsOperationDelta:
				var response WsDeltaFuturesTicker
				err = json.Unmarshal(respRaw, &response)
				if err != nil {
					return err
				}

				if len(response.Data.Delete) > 0 {
					for x := range response.Data.Delete {
						var p currency.Pair
						p, err = by.extractCurrencyPair(response.Data.Delete[x].Symbol, asset.Futures)
						if err != nil {
							return err
						}
						by.Websocket.DataHandler <- &ticker.Price{
							ExchangeName: by.Name,
							Last:         response.Data.Delete[x].LastPrice,
							High:         response.Data.Delete[x].HighPrice24h,
							Low:          response.Data.Delete[x].LowPrice24h,
							Bid:          response.Data.Delete[x].BidPrice,
							Ask:          response.Data.Delete[x].AskPrice,
							Volume:       response.Data.Delete[x].Volume24h,
							Close:        response.Data.Delete[x].PrevPrice24h,
							LastUpdated:  response.Data.Delete[x].UpdateAt,
							AssetType:    asset.Futures,
							Pair:         p,
						}
					}
				}

				if len(response.Data.Update) > 0 {
					for x := range response.Data.Update {
						var p currency.Pair
						p, err = by.extractCurrencyPair(response.Data.Update[x].Symbol, asset.Futures)
						if err != nil {
							return err
						}
						var tick *ticker.Price
						tick, err = by.FetchTicker(context.Background(), p, asset.Futures)
						if err != nil {
							return err
						}
						var changed bool
						if response.Data.Update[x].BidPrice != 0 && response.Data.Update[x].BidPrice != tick.Bid {
							changed = true
							tick.Bid = response.Data.Update[x].BidPrice
						}
						if response.Data.Update[x].AskPrice != 0 && response.Data.Update[x].AskPrice != tick.Ask {
							changed = true
							tick.Ask = response.Data.Update[x].AskPrice
						}
						if response.Data.Update[x].IndexPrice != 0 && response.Data.Update[x].IndexPrice != tick.Last {
							changed = true
							tick.Last = response.Data.Update[x].IndexPrice
						}
						tick.LastUpdated = response.Data.Update[x].UpdateAt
						if changed {
							by.Websocket.DataHandler <- tick
						}
					}
				}

				if len(response.Data.Insert) > 0 {
					for x := range response.Data.Insert {
						var p currency.Pair
						p, err = by.extractCurrencyPair(response.Data.Insert[x].Symbol, asset.Futures)
						if err != nil {
							return err
						}

						by.Websocket.DataHandler <- &ticker.Price{
							ExchangeName: by.Name,
							Last:         response.Data.Insert[x].LastPrice,
							High:         response.Data.Insert[x].HighPrice24h,
							Low:          response.Data.Insert[x].LowPrice24h,
							Bid:          response.Data.Insert[x].BidPrice,
							Ask:          response.Data.Insert[x].AskPrice,
							Volume:       response.Data.Insert[x].Volume24h,
							Close:        response.Data.Insert[x].PrevPrice24h,
							LastUpdated:  response.Data.Insert[x].UpdateAt,
							AssetType:    asset.Futures,
							Pair:         p,
						}
					}
				}

			default:
				by.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: by.Name + stream.UnhandledMessage + "unsupported ticker operation"}
			}
		}

	case wsInsurance:
		var response WsInsurance
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
			p, err = currency.NewPairFromString(response.Data[i].Symbol)
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
				AssetType: asset.Futures,
				Pair:      p,
				Price:     response.Data[i].Price,
				Amount:    response.Data[i].OrderQty,
				Side:      oSide,
				Status:    oStatus,
				Trades: []order.TradeHistory{
					{
						Price:     response.Data[i].Price,
						Amount:    response.Data[i].OrderQty,
						Exchange:  by.Name,
						Side:      oSide,
						Timestamp: response.Data[i].Time,
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
			p, err = currency.NewPairFromString(response.Data[x].Symbol)
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
				Price:     response.Data[x].Price,
				Amount:    response.Data[x].OrderQty,
				Exchange:  by.Name,
				OrderID:   response.Data[x].OrderID,
				Type:      oType,
				Side:      oSide,
				Status:    oStatus,
				AssetType: asset.Futures,
				Date:      response.Data[x].Time,
				Pair:      p,
				Trades: []order.TradeHistory{
					{
						Price:     response.Data[x].Price,
						Amount:    response.Data[x].OrderQty,
						Exchange:  by.Name,
						Side:      oSide,
						Timestamp: response.Data[x].Time,
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
			p, err = currency.NewPairFromString(response.Data[x].Symbol)
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
				Price:     response.Data[x].Price,
				Amount:    response.Data[x].OrderQty,
				Exchange:  by.Name,
				OrderID:   response.Data[x].OrderID,
				AccountID: strconv.FormatInt(response.Data[x].UserID, 10),
				Type:      oType,
				Side:      oSide,
				Status:    oStatus,
				AssetType: asset.Futures,
				Date:      response.Data[x].Time,
				Pair:      p,
				Trades: []order.TradeHistory{
					{
						Price:     response.Data[x].Price,
						Amount:    response.Data[x].OrderQty,
						Exchange:  by.Name,
						Side:      oSide,
						Timestamp: response.Data[x].Time,
					},
				},
			}
		}

	default:
		by.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: by.Name + stream.UnhandledMessage + string(respRaw)}
	}

	return nil
}
