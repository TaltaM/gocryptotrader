package bittrex

import (
	"bytes"
	"compress/flate"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	bittrexAPIWSURL             = "wss://socket-v3.bittrex.com/signalr"
	bittrexAPIWSNegotiationsURL = "https://socket-v3.bittrex.com/signalr"

	bittrexWebsocketTimer = 13 * time.Second
	wsTicker              = "ticker"
	wsOrderbook           = "orderbook"
	wsMarketSummary       = "market_summary"
	wsOrders              = "order"
	wsHeartbeat           = "heartbeat"
	authenticate          = "Authenticate"
	subscribe             = "subscribe"
	unsubscribe           = "unsubscribe"
)

var defaultSpotSubscribedChannels = []string{
	wsHeartbeat,
	wsTicker,
	wsOrderbook,
	wsMarketSummary,
}

var defaultSpotSubscribedChannelsAuth = []string{
	wsOrders,
}

var invocationIDCounter int

// WsConnect connects to a websocket feed
func (b *Bittrex) WsConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}

	invocationIDCounter = 0
	var wsHandshakeData WsSignalRHandshakeData
	err := b.WsSignalRHandshake(&wsHandshakeData)
	if err != nil {
		return err
	}

	var dialer websocket.Dialer
	endpoint, err := b.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	params := url.Values{}
	params.Set("clientProtocol", "1.5")
	params.Set("transport", "webSockets")
	params.Set("connectionToken", wsHandshakeData.ConnectionToken)
	params.Set("connectionData", "[{name:\"c3\"}]")
	params.Set("tid", "10")

	path := common.EncodeURLValues("/connect", params)

	err = b.Websocket.SetWebsocketURL(endpoint+path, false, false)
	if err != nil {
		return err
	}

	err = b.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	// Can set up custom ping handler per websocket connection.
	b.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		MessageType: websocket.PingMessage,
		Delay:       bittrexWebsocketTimer,
	})

	// This reader routine is called prior to initiating a subscription for
	// efficient processing.
	go b.wsReadData()
	b.setupOrderbookManager()

	if b.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		err = b.WsAuth()
		if err != nil {
			b.Websocket.DataHandler <- err
			b.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	return nil
}

// WsSignalRHandshake requests the SignalR connection token over https
func (b *Bittrex) WsSignalRHandshake(result interface{}) error {
	endpoint, err := b.API.Endpoints.GetURL(exchange.WebsocketSpotSupplementary)
	if err != nil {
		return err
	}
	path := "/negotiate?connectionData=[{name:\"c3\"}]&clientProtocol=1.5"
	return b.SendPayload(context.Background(), &request.Item{
		Method:        http.MethodGet,
		Path:          endpoint + path,
		Result:        result,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
	})
}

// WsAuth sends an authentication message to receive auth data
// Authentications expire after 10 minutes
func (b *Bittrex) WsAuth() error {
	// [apiKey, timestamp in ms, random uuid, signed payload]
	apiKey := b.API.Credentials.Key
	randomContent, err := uuid.NewV4()
	if err != nil {
		return err
	}
	timestamp := strconv.FormatInt(time.Now().UnixNano()/1000000, 10)
	hmac := crypto.GetHMAC(
		crypto.HashSHA512,
		[]byte(timestamp+randomContent.String()),
		[]byte(b.API.Credentials.Secret),
	)
	signature := crypto.HexEncodeToString(hmac)

	invocationIDCounter++
	req := WsEventRequest{
		Hub:          "c3",
		Method:       authenticate,
		InvocationID: invocationIDCounter,
	}

	arguments := make([]string, 0)
	arguments = append(arguments, apiKey, timestamp, randomContent.String(), signature)
	req.Arguments = arguments

	requestString, err := json.Marshal(req)
	if err != nil {
		return err
	}
	if b.Verbose {
		log.Debugf(log.WebsocketMgr, "%s Sending JSON message - %s\n", b.Name, requestString)
	}
	err = b.Websocket.Conn.SendJSONMessage(req)
	if err != nil {
		return err
	}

	b.WsPendingRequests[req.InvocationID] = WsPendingRequest{
		WsEventRequest: req,
	}
	return nil
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be
// handled by ManageSubscriptions()
func (b *Bittrex) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	var subscriptions []stream.ChannelSubscription
	pairs, err := b.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}

	channels := defaultSpotSubscribedChannels
	if b.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		channels = append(channels, defaultSpotSubscribedChannelsAuth...)
	}

	for i := range pairs {
		pair, err := b.FormatExchangeCurrency(pairs[i], asset.Spot)
		if err != nil {
			return nil, err
		}
		for y := range channels {
			var channel string
			switch channels[y] {
			case wsOrderbook:
				channel = channels[y] + "_" + pair.String() + "_" + strconv.FormatInt(orderbookDepth, 10)
			case wsTicker:
				channel = channels[y] + "_" + pair.String()
			case wsMarketSummary:
				channel = channels[y] + "_" + pair.String()
			default:
				channel = channels[y]
			}
			subscriptions = append(subscriptions,
				stream.ChannelSubscription{
					Channel:  channel,
					Currency: pair,
					Asset:    asset.Spot,
				})
		}
	}

	return subscriptions, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (b *Bittrex) Subscribe(channelsToSubscribe []stream.ChannelSubscription) error {
	invocationIDCounter++
	req := WsEventRequest{
		Hub:          "c3",
		Method:       subscribe,
		InvocationID: invocationIDCounter,
	}

	var channels []string
	for i := range channelsToSubscribe {
		channels = append(channels, channelsToSubscribe[i].Channel)
	}
	arguments := make([][]string, 0)
	arguments = append(arguments, channels)
	req.Arguments = arguments

	requestString, err := json.Marshal(req)
	if err != nil {
		return err
	}
	if b.Verbose {
		log.Debugf(log.WebsocketMgr, "%s Sending JSON message - %s\n", b.Name, requestString)
	}
	err = b.Websocket.Conn.SendJSONMessage(req)
	if err != nil {
		return err
	}

	b.WsPendingRequests[req.InvocationID] = WsPendingRequest{
		req,
		&channelsToSubscribe,
	}

	return nil
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (b *Bittrex) Unsubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	invocationIDCounter++
	req := WsEventRequest{
		Hub:          "c3",
		Method:       unsubscribe,
		InvocationID: invocationIDCounter,
	}

	var channels []string
	for i := range channelsToUnsubscribe {
		channels = append(channels, channelsToUnsubscribe[i].Channel)
	}
	arguments := make([][]string, 0)
	arguments = append(arguments, channels)
	req.Arguments = arguments

	requestString, err := json.Marshal(req)
	if err != nil {
		return err
	}
	if b.Verbose {
		log.Debugf(log.WebsocketMgr, "%s Sending JSON message - %s\n", b.Name, requestString)
	}
	err = b.Websocket.Conn.SendJSONMessage(req)
	if err != nil {
		return err
	}

	b.Websocket.RemoveSuccessfulUnsubscriptions(channelsToUnsubscribe...)
	return nil
}

// wsReadData gets and passes on websocket messages for processing
func (b *Bittrex) wsReadData() {
	b.Websocket.Wg.Add(1)
	defer b.Websocket.Wg.Done()

	for {
		select {
		case <-b.Websocket.ShutdownC:
			return
		default:
			resp := b.Websocket.Conn.ReadMessage()
			if resp.Raw == nil {
				log.Warnf(log.WebsocketMgr, "%s Received empty message\n", b.Name)
				return
			}

			err := b.wsHandleData(resp.Raw)
			if err != nil {
				b.Websocket.DataHandler <- err
			}
		}
	}
}

func (b *Bittrex) wsDecodeMessage(encodedMessage string, v interface{}) error {
	raw, err := crypto.Base64Decode(encodedMessage)
	if err != nil {
		return err
	}
	reader := flate.NewReader(bytes.NewBuffer(raw))
	message, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}
	return json.Unmarshal(message, v)
}

func (b *Bittrex) wsHandleResponseData(req WsPendingRequest, respRaw []byte) error {
	switch req.Method {
	case "Authenticate":
		var response WsAuthResponse
		err := json.Unmarshal(respRaw, &response)
		if err != nil {
			log.Warnf(log.WebsocketMgr, "%s - wsHandleResponseData - Cannot unmarshal into WsAuthResponse (%s)\n", b.Name, string(respRaw))
			return err
		}
		if !response.Response.Success {
			log.Warnf(log.WebsocketMgr, "%s - Unable to authenticate (%s)", b.Name, response.Response.ErrorCode)
			b.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
		return nil
	case "subscribe":
		var response WsSubscriptionResponse
		err := json.Unmarshal(respRaw, &response)
		if err != nil {
			log.Warnf(log.WebsocketMgr, "%s - wsHandleResponseData - Cannot unmarshal into WsSubscriptionResponse (%s)\n", b.Name, string(respRaw))
			return err
		}
		channels, ok := req.Arguments.([][]string)
		if !ok {
			log.Warnf(log.WebsocketMgr, "%s - wsHandleResponseData - Cannot get channel list\n", b.Name)
		}
		for i := range response.Response {
			if !response.Response[i].Success {
				log.Warnf(log.WebsocketMgr, "%s - Unable to subscribe to %s (%s)", b.Name, channels[0][i], response.Response[i].ErrorCode)
				continue
			}
			b.Websocket.AddSuccessfulSubscriptions((*req.ChannelsToSubscribe)[i])
		}
	case "unsubscribe":
		var response WsSubscriptionResponse
		err := json.Unmarshal(respRaw, &response)
		if err != nil {
			log.Warnf(log.WebsocketMgr, "%s - wsHandleResponseData - Cannot unmarshal into WsSubscriptionResponse (%s)\n", b.Name, string(respRaw))
			return err
		}
		channels, ok := req.Arguments.([][]string)
		if !ok {
			log.Warnf(log.WebsocketMgr, "%s - wsHandleResponseData - Cannot get channel list\n", b.Name)
		}
		for i := range response.Response {
			if !response.Response[i].Success {
				log.Warnf(log.WebsocketMgr, "%s - Unable to subscribe to %s (%s)", b.Name, channels[0][i], response.Response[i].ErrorCode)
				continue
			}
			b.Websocket.RemoveSuccessfulUnsubscriptions((*req.ChannelsToSubscribe)[i])
		}
	default:
		return errors.New("unrecognized response message")
	}
	return nil
}

func (b *Bittrex) wsHandleData(respRaw []byte) error {
	var response WsEventResponse
	err := json.Unmarshal(respRaw, &response)
	if err != nil {
		log.Warnf(log.WebsocketMgr, "%s Cannot unmarshal into eventResponse (%s)\n", b.Name, string(respRaw))
		return err
	}
	if response.Response != nil && response.InvocationID > 0 {
		req, hasRequest := b.WsPendingRequests[response.InvocationID]
		if !hasRequest {
			return errors.New("received response to unknown request")
		}
		delete(b.WsPendingRequests, req.InvocationID)

		return b.wsHandleResponseData(req, respRaw)
	}
	if response.Response == nil && len(response.Message) == 0 && response.C == "" {
		if b.Verbose {
			log.Warnf(log.WebsocketMgr, "%s Received keep-alive (%s)\n", b.Name, string(respRaw))
		}
		return nil
	}
	for i := range response.Message {
		switch response.Message[i].Method {
		case "orderBook":
			for j := range response.Message[i].Arguments {
				var orderbookUpdate OrderbookUpdateMessage
				err = b.wsDecodeMessage(response.Message[i].Arguments[j], &orderbookUpdate)
				if err != nil {
					return err
				}
				var init bool
				init, err = b.UpdateLocalOBBuffer(&orderbookUpdate)
				if err != nil {
					if init {
						return nil
					}
					return fmt.Errorf("%v - UpdateLocalCache error: %s",
						b.Name,
						err)
				}
			}
		case "ticker":
			for j := range response.Message[i].Arguments {
				var tickerUpdate TickerData
				err = b.wsDecodeMessage(response.Message[i].Arguments[j], &tickerUpdate)
				if err != nil {
					return err
				}
				err = b.WsProcessUpdateTicker(tickerUpdate)
				if err != nil {
					return err
				}
			}
		case "marketSummary":
			for j := range response.Message[i].Arguments {
				var marketSummaryUpdate MarketSummaryData
				err = b.wsDecodeMessage(response.Message[i].Arguments[j], &marketSummaryUpdate)
				if err != nil {
					return err
				}

				err = b.WsProcessUpdateMarketSummary(marketSummaryUpdate)
				if err != nil {
					return err
				}
			}
		case "heartbeat":
			if b.Verbose {
				log.Warnf(log.WebsocketMgr, "%s Received heartbeat\n", b.Name)
			}
		case "authenticationExpiring":
			if b.Verbose {
				log.Debugf(log.WebsocketMgr, "%s - Re-authenticating.\n", b.Name)
			}
			err = b.WsAuth()
			if err != nil {
				b.Websocket.DataHandler <- err
				b.Websocket.SetCanUseAuthenticatedEndpoints(false)
			}
		case "order":
			for j := range response.Message[i].Arguments {
				var orderUpdate OrderUpdateMessage
				err = b.wsDecodeMessage(response.Message[i].Arguments[j], &orderUpdate)
				if err != nil {
					return err
				}
				err = b.WsProcessUpdateOrder(&orderUpdate)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// WsProcessUpdateTicker processes an update on the ticker
func (b *Bittrex) WsProcessUpdateTicker(tickerData TickerData) error {
	pair, err := currency.NewPairFromString(tickerData.Symbol)
	if err != nil {
		return err
	}

	tickerPrice, err := ticker.GetTicker(b.Name, pair, asset.Spot)
	if err != nil {
		// Received partial data for a ticker: request the missing data through REST
		marketSummaryData, err := b.GetMarketSummary(tickerData.Symbol)
		if err != nil {
			return err
		}

		tickerPrice = b.constructTicker(tickerData, marketSummaryData, pair, asset.Spot)
		b.Websocket.DataHandler <- tickerPrice

		return nil
	}

	tickerPrice.Last = tickerData.LastTradeRate
	tickerPrice.Bid = tickerData.BidRate
	tickerPrice.Ask = tickerData.AskRate

	b.Websocket.DataHandler <- tickerPrice

	return nil
}

// WsProcessUpdateMarketSummary processes an update on the ticker
func (b *Bittrex) WsProcessUpdateMarketSummary(marketSummaryData MarketSummaryData) error {
	pair, err := currency.NewPairFromString(marketSummaryData.Symbol)
	if err != nil {
		return err
	}

	tickerPrice, err := ticker.GetTicker(b.Name, pair, asset.Spot)
	if err != nil {
		// Received partial data for a ticker: request the missing data through REST
		var tickerData TickerData
		tickerData, err = b.GetTicker(marketSummaryData.Symbol)
		if err != nil {
			return err
		}

		tickerPrice = b.constructTicker(tickerData, marketSummaryData, pair, asset.Spot)
		b.Websocket.DataHandler <- tickerPrice

		return nil
	}

	tickerPrice.High = marketSummaryData.High
	tickerPrice.Low = marketSummaryData.Low
	tickerPrice.Volume = marketSummaryData.Volume
	tickerPrice.QuoteVolume = marketSummaryData.QuoteVolume
	tickerPrice.LastUpdated = marketSummaryData.UpdatedAt

	b.Websocket.DataHandler <- tickerPrice

	return nil
}

// WsProcessUpdateOrder processes an update on the open orders
func (b *Bittrex) WsProcessUpdateOrder(data *OrderUpdateMessage) error {
	var orderSide order.Side
	var orderStatus order.Status
	var pair currency.Pair

	orderType, err := order.StringToOrderType(data.Delta.Type)
	if err != nil {
		b.Websocket.DataHandler <- order.ClassificationError{
			Exchange: b.Name,
			OrderID:  data.Delta.ID,
			Err:      err,
		}
	}
	orderSide, err = order.StringToOrderSide(data.Delta.Direction)
	if err != nil {
		b.Websocket.DataHandler <- order.ClassificationError{
			Exchange: b.Name,
			OrderID:  data.Delta.ID,
			Err:      err,
		}
	}
	orderStatus, err = order.StringToOrderStatus(data.Delta.Status)
	if err != nil {
		b.Websocket.DataHandler <- order.ClassificationError{
			Exchange: b.Name,
			OrderID:  data.Delta.ID,
			Err:      err,
		}
	}

	pair, err = currency.NewPairFromString(data.Delta.MarketSymbol)
	if err != nil {
		b.Websocket.DataHandler <- order.ClassificationError{
			Exchange: b.Name,
			OrderID:  data.Delta.ID,
			Err:      err,
		}
	}

	b.Websocket.DataHandler <- &order.Detail{
		ImmediateOrCancel: data.Delta.TimeInForce == string(ImmediateOrCancel),
		FillOrKill:        data.Delta.TimeInForce == string(GoodTilCancelled),
		PostOnly:          data.Delta.TimeInForce == string(PostOnlyGoodTilCancelled),
		Price:             data.Delta.Limit,
		Amount:            data.Delta.Quantity,
		RemainingAmount:   data.Delta.Quantity - data.Delta.FillQuantity,
		ExecutedAmount:    data.Delta.FillQuantity,
		Exchange:          b.Name,
		ID:                data.Delta.ID,
		Type:              orderType,
		Side:              orderSide,
		Status:            orderStatus,
		AssetType:         asset.Spot,
		Date:              data.Delta.CreatedAt,
		Pair:              pair,
	}
	return nil
}
