package gateio

import (
	"encoding/json"
	"math"
	"strconv"
	"time"
)

// func (a *ClosePositionRequestParam) UnmarshalJSON(data []byte) error {
// 	type Alias ClosePositionRequestParam
// 	chil := &struct {
// 		*Alias
// 	}{
// 		Alias: (*Alias)(a),
// 	}
// 	return nil
// }

// UnmarshalJSON decerializes json, and timestamp information.
func (a *SubAccountTransferResponse) UnmarshalJSON(data []byte) error {
	type Alias SubAccountTransferResponse
	chil := &struct {
		*Alias
		Timestamp float64 `json:"timest,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.Unix(int64(chil.Timestamp), 0)
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information.
func (a *WithdrawalResponse) UnmarshalJSON(data []byte) error {
	type Alias WithdrawalResponse
	chil := &struct {
		*Alias
		Timestamp float64 `json:"timestamp,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.Unix(int64(chil.Timestamp), 0)
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information.
func (a *OptionTradingHistory) UnmarshalJSON(data []byte) error {
	type Alias OptionTradingHistory
	chil := &struct {
		*Alias
		CreateTime float64 `json:"create_time"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.CreateTime = time.Unix(int64(chil.CreateTime), 0)
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information.
func (a *OptionOrderResponse) UnmarshalJSON(data []byte) error {
	type Alias OptionOrderResponse
	chil := &struct {
		*Alias
		CreateTime float64 `json:"create_time"`
		FinishTime float64 `json:"finish_time"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.CreateTime = time.Unix(int64(chil.CreateTime), 0)
	a.FinishTime = time.Unix(int64(chil.FinishTime), 0)
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information.
func (a *ContractClosePosition) UnmarshalJSON(data []byte) error {
	type Alias ContractClosePosition
	chil := &struct {
		*Alias
		PositionCloseTime float64 `json:"time"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.PositionCloseTime = time.Unix(int64(chil.PositionCloseTime), 0)
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information.
func (a *AccountBook) UnmarshalJSON(data []byte) error {
	type Alias AccountBook
	chil := &struct {
		*Alias
		ChangeTime float64 `json:"time"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.ChangeTime = time.Unix(int64(chil.ChangeTime), 0)
	return nil
}

func (a *CurrencyPairDetail) UnmarshalJSON(data []byte) error {
	type Alias CurrencyPairDetail
	chil := struct {
		*Alias
		Fee            string `json:"fee"`
		MinBaseAmount  string `json:"min_base_amount"`
		MinQuoteAmount string `json:"min_quote_amount"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, &chil); er != nil {
		return er
	}
	if val, er := strconv.ParseFloat(chil.Fee, 64); er == nil {
		a.Fee = val
	}
	if val, er := strconv.ParseFloat(chil.MinBaseAmount, 64); er == nil {
		a.MinBaseAmount = val
	}
	if val, er := strconv.ParseFloat(chil.MinQuoteAmount, 64); er == nil {
		a.MinQuoteAmount = val
	}
	return nil
}

func (a *Ticker) UnmarshalJSON(data []byte) error {
	type Alias Ticker
	child := &struct {
		*Alias
		BaseVolume  string `json:"base_volume"`
		QuoteVolume string `json:"quote_volume"`
		High24H     string `json:"high_24h"`
		Low24H      string `json:"low_24h"`

		LowestAsk       string `json:"lowest_ask"`
		HighestBid      string `json:"highest_bid"`
		EtfLeverage     string `json:"etf_leverage"`
		EtfPreTimestamp int64  `json:"etf_pre_timestamp"`
	}{
		Alias: (*Alias)(a),
	}
	var er error
	if er = json.Unmarshal(data, child); er != nil {
		return er
	}
	var val float64
	if val, er = strconv.ParseFloat(child.BaseVolume, 64); er == nil {
		a.BaseVolume = val
	}
	if val, er = strconv.ParseFloat(child.QuoteVolume, 64); er == nil {
		a.QuoteVolume = val
	}
	if val, er = strconv.ParseFloat(child.High24H, 64); er == nil {
		a.High24H = val
	}
	if val, er = strconv.ParseFloat(child.Low24H, 64); er == nil {
		a.Low24H = val
	}
	if val, er = strconv.ParseFloat(child.LowestAsk, 64); er == nil {
		a.LowestAsk = val
	}
	if val, er = strconv.ParseFloat(child.HighestBid, 64); er == nil {
		a.HighestBid = val
	}
	if val, er = strconv.ParseFloat(child.EtfLeverage, 64); er == nil {
		a.EtfLeverage = val
	}
	a.EtfPreTimestamp = time.Unix(child.EtfPreTimestamp, 0)
	return nil
}

// UnmarshalJSON to decerialize the timestamp information to golang time.Time instance
func (a *OrderbookData) UnmarshalJSON(data []byte) error {
	type Alias OrderbookData
	chil := &struct {
		*Alias
		Current float64 `json:"current"`
		Update  float64 `json:"update"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Current = time.Unix(int64(math.Round(chil.Current)), 0)
	a.Update = time.Unix(int64(math.Round(chil.Update)), 0)
	return nil
}

// UnmarshalJSON to decerialize timestamp information and create OrderbookItem instance from the list of asks and bids data.
func (a *Orderbook) UnmarshalJSON(data []byte) error {
	type Alias Orderbook
	type askorbid struct {
		Price string  `json:"p"`
		Size  float64 `json:"s"`
	}
	chil := &struct {
		*Alias
		Current float64    `json:"current"`
		Update  float64    `json:"update"`
		Bids    []askorbid `json:"asks"`
		Asks    []askorbid `json:"bids"`
	}{}
	if er := json.Unmarshal(data, &chil); er != nil {
		return er
	}
	a.Current = time.Unix(int64(chil.Current), 0)
	a.Update = time.Unix(int64(chil.Update), 0)
	asks := make([]OrderbookItem, len(chil.Asks))
	bids := make([]OrderbookItem, len(chil.Bids))
	for x := range chil.Asks {
		val, er := strconv.ParseFloat(chil.Asks[x].Price, 64)
		if er != nil {
			return er
		}
		asks[x] = OrderbookItem{
			Price:  val,
			Amount: float64(chil.Asks[x].Size),
		}
	}
	for x := range chil.Bids {
		val, er := strconv.ParseFloat(chil.Bids[x].Price, 64)
		if er != nil {
			return er
		}
		bids[x] = OrderbookItem{
			Price:  val,
			Amount: float64(chil.Bids[x].Size),
		}
	}
	a.Asks = asks
	a.Bids = bids
	return nil
}

// UnmarshalJSON decerializes the unix seconds, and milliseconds timestamp information to builtin time.Time.
func (a *Trade) UnmarshalJSON(data []byte) error {
	type Alias Trade
	chil := &struct {
		*Alias
		TradingTime  int64   `json:"create_time,string"`
		CreateTimeMs float64 `json:"create_time_ms,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.TradingTime = time.Unix(chil.TradingTime, 0)
	a.CreateTimeMs = time.UnixMilli(int64(math.Round(chil.CreateTimeMs)))
	return nil
}

// UnmarshalJSON to decerialize timestamp information to built-int golang time.Time instance.
func (a *FuturesContract) UnmarshalJSON(data []byte) error {
	type Alias FuturesContract
	chil := &struct {
		*Alias
		FundingNextApply int64 `json:"funding_next_apply"`
		ConfigChangeTime int64 `json:"config_change_time"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.FundingNextApply = time.Unix(chil.FundingNextApply, 0)
	a.ConfigChangeTime = time.Unix(chil.ConfigChangeTime, 0)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *TradingHistoryItem) UnmarshalJSON(data []byte) error {
	type Alias TradingHistoryItem
	chil := &struct {
		*Alias
		CreateTime float64 `json:"create_time"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.CreateTime = time.Unix(int64(chil.CreateTime), 0)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *FuturesCandlestick) UnmarshalJSON(data []byte) error {
	type Alias FuturesCandlestick
	chil := &struct {
		*Alias
		Timestamp float64 `json:"t"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.Unix(int64(chil.Timestamp), 10)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *FuturesFundingRate) UnmarshalJSON(data []byte) error {
	type Alias FuturesFundingRate
	chil := &struct {
		*Alias
		Timestamp float64 `json:"t"`
		Rate      string  `json:"r"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if val, er := strconv.ParseFloat(chil.Rate, 64); er == nil {
		a.Rate = val
	}
	a.Timestamp = time.Unix(int64(chil.Timestamp), 0)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *InsuranceBalance) UnmarshalJSON(data []byte) error {
	type Alias InsuranceBalance
	chil := &struct {
		*Alias
		Timestamp float64 `json:"t"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.Unix(int64(chil.Timestamp), 0)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *ContractStat) UnmarshalJSON(data []byte) error {
	type Alias ContractStat
	chil := &struct {
		*Alias
		Time float64 `json:"time"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Time = time.Unix(int64(chil.Time), 0)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp.
func (a *LiquidationHistory) UnmarshalJSON(data []byte) error {
	type Alias LiquidationHistory
	chil := &struct {
		*Alias
		Time float64 `json:"time"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Time = time.Unix(int64(chil.Time), 0)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *DeliveryContract) UnmarshalJSON(data []byte) error {
	type Alias DeliveryContract
	chil := &struct {
		*Alias
		ExpireTime       float64 `json:"expire_time"`
		ConfigChangeTime float64 `json:"config_change_time"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.ExpireTime = time.Unix(int64(chil.ExpireTime), 0)
	a.ConfigChangeTime = time.Unix(int64(chil.ConfigChangeTime), 0)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *OptionContract) UnmarshalJSON(data []byte) error {
	type Alias OptionContract
	chil := &struct {
		*Alias
		CreateTime     float64 `json:"create_time"`
		ExpirationTime float64 `json:"expiration_time"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.CreateTime = time.Unix(int64(chil.CreateTime), 0)
	a.ExpirationTime = time.Unix(int64(chil.ExpirationTime), 0)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *OptionSettlement) UnmarshalJSON(data []byte) error {
	type Alias OptionSettlement
	chil := &struct {
		*Alias
		Time float64 `json:"time"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Time = time.Unix(int64(chil.Time), 10)
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information.
func (a *SpotOrder) UnmarshalJSON(data []byte) error {
	type Alias SpotOrder
	chil := &struct {
		*Alias
		CreateTime   int64 `json:"create_time,string"`
		UpdateTime   int64 `json:"update_time,string"`
		CreateTimeMs int64 `json:"create_time_ms"`
		UpdateTimeMs int64 `json:"update_time_ms"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.CreateTime = time.Unix(chil.CreateTime, 0)
	a.UpdateTime = time.Unix(chil.UpdateTime, 0)
	a.CreateTimeMs = time.UnixMilli(chil.CreateTimeMs)
	a.UpdateTimeMs = time.UnixMilli(chil.UpdateTimeMs)
	return nil
}