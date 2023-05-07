package stream

import (
	"errors"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
)

var defaultWebsocketWrapperSetup = &WrapperWebsocket{
	AssetTypeWebsockets: map[asset.Item]*Websocket{
		asset.Spot: {
			defaultURL:   "testDefaultURL",
			runningURL:   "wss://testRunningURL",
			connector:    func() error { return nil },
			Subscriber:   func(_ []ChannelSubscription) error { return nil },
			Unsubscriber: func(_ []ChannelSubscription) error { return nil },
			GenerateSubs: func() ([]ChannelSubscription, error) {
				return []ChannelSubscription{
					{Channel: "TestSub"},
					{Channel: "TestSub2"},
					{Channel: "TestSub3"},
					{Channel: "TestSub4"},
				}, nil
			},
		},
	},
}

var defaultWrapperSetup = &WebsocketWrapperSetup{
	ExchangeConfig: &config.Exchange{
		Features: &config.FeaturesConfig{
			Enabled: config.FeaturesEnabledConfig{Websocket: true},
		},
		API: config.APIConfig{
			AuthenticatedWebsocketSupport: true,
		},
		WebsocketTrafficTimeout: time.Second * 5,
		Name:                    "exchangeName",
	},
	Features: &protocol.Features{
		Subscribe:   true,
		Unsubscribe: true,
	},
}

func TestWebsocketWrapperSetup(t *testing.T) {
	t.Parallel()
	var websocketWrapper *WrapperWebsocket
	err := websocketWrapper.Setup(defaultWrapperSetup)
	if !errors.Is(err, errWebsocketWrapperIsNil) {
		t.Fatalf("found %v, but expected %v", err, errWebsocketWrapperIsNil)
	}
	websocketWrapper = NewWrapper()
	err = websocketWrapper.Setup(nil)
	if !errors.Is(err, errWebsocketSetupIsNil) {
		t.Fatalf("found %v, but expected %v", err, errWebsocketSetupIsNil)
	}
	var wsSetup WebsocketWrapperSetup
	err = websocketWrapper.Setup(&wsSetup)
	if !errors.Is(err, errExchangeConfigIsNil) {
		t.Errorf("found %v, but expected %v", err, errExchangeConfigIsNil)
	}
	wsSetup.ExchangeConfig = &config.Exchange{}
	err = websocketWrapper.Setup(&wsSetup)
	if !errors.Is(err, errExchangeConfigNameUnset) {
		t.Errorf("found %v, but expected %v", err, errExchangeConfigIsNil)
	}
	wsSetup.ExchangeConfig = &config.Exchange{Name: "test_exchange_name"}
	err = websocketWrapper.Setup(&wsSetup)
	if !errors.Is(err, errWebsocketFeaturesIsUnset) {
		t.Errorf("found %v, but expected %v", err, errWebsocketFeaturesIsUnset)
	}
	wsSetup.Features = &protocol.Features{}
	err = websocketWrapper.Setup(&wsSetup)
	if !errors.Is(err, errConfigFeaturesIsNil) {
		t.Errorf("found %v, but expected %v", err, errConfigFeaturesIsNil)
	}
	wsSetup.ExchangeConfig.Features = &config.FeaturesConfig{
		Enabled: config.FeaturesEnabledConfig{
			SaveTradeData: true,
			FillsFeed:     true,
		},
	}
	err = websocketWrapper.Setup(&wsSetup)
	if !errors.Is(err, errInvalidTrafficTimeout) {
		t.Errorf("found %v, but expected %v", err, errInvalidTrafficTimeout)
	}
	wsSetup.ExchangeConfig.WebsocketTrafficTimeout = time.Second * 10
	err = websocketWrapper.Setup(&wsSetup)
	if err != nil {
		t.Error(err)
	}
	err = websocketWrapper.Setup(defaultWrapperSetup)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAssetWebsocket(t *testing.T) {
	websocketWrapper := NewWrapper()
	err := websocketWrapper.Setup(defaultWrapperSetup)
	if err != nil {
		t.Fatal(err)
	}
	_, err = websocketWrapper.AddWebsocket(&WebsocketSetup{
		DefaultURL:   "testDefaultURL",
		RunningURL:   "wss://testRunningURL",
		Connector:    func() error { return nil },
		Subscriber:   func(_ []ChannelSubscription) error { return nil },
		Unsubscriber: func(_ []ChannelSubscription) error { return nil },
		GenerateSubscriptions: func() ([]ChannelSubscription, error) {
			return []ChannelSubscription{
				{Channel: "TestSub"},
			}, nil
		},
		AssetType: asset.Spot,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := websocketWrapper.GetAssetWebsocket(asset.Empty); !errors.Is(err, ErrAssetWebsocketNotFound) {
		t.Errorf("found %v, but expected %v", err, ErrAssetWebsocketNotFound)
	}
	if _, err := websocketWrapper.GetAssetWebsocket(asset.Spot); err != nil {
		t.Error(err)
	}
}

func TestWebsocketWrapper(t *testing.T) {
	t.Parallel()
	wsInit := WrapperWebsocket{}
	err := wsInit.Setup(&WebsocketWrapperSetup{
		ExchangeConfig: &config.Exchange{
			Features: &config.FeaturesConfig{
				Enabled: config.FeaturesEnabledConfig{Websocket: true},
			},
			Name: "test",
		},
	})
	if !errors.Is(err, errWebsocketAlreadyInitialised) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errWebsocketAlreadyInitialised)
	}

	ws := *NewWrapper()
	err = ws.SetProxyAddress("garbagio")
	if err == nil {
		t.Error("error cannot be nil")
	}

	_, err = ws.AddWebsocket(&WebsocketSetup{
		DefaultURL:   "testDefaultURL",
		RunningURL:   "wss://testRunningURL",
		Connector:    func() error { return nil },
		Subscriber:   func(_ []ChannelSubscription) error { return nil },
		Unsubscriber: func(_ []ChannelSubscription) error { return nil },
		GenerateSubscriptions: func() ([]ChannelSubscription, error) {
			return []ChannelSubscription{
				{Channel: "TestSub"},
			}, nil
		},
		AssetType: asset.Spot,
	})
	if err != nil {
		t.Fatal(err)
	}

	// removing proxy
	err = ws.SetProxyAddress("")
	if err != nil {
		t.Error(err)
	}
	// reinstate proxy
	err = ws.SetProxyAddress("http://localhost:1337")
	if err != nil {
		t.Error(err)
	}
	// conflict proxy
	err = ws.SetProxyAddress("http://localhost:1337")
	if err == nil {
		t.Error("error cannot be nil")
	}
	err = ws.Setup(defaultWrapperSetup)
	if err != nil {
		t.Fatal(err)
	}
	if ws.GetName() != "exchangeName" {
		t.Error("WebsocketSetup")
	}
	if ws.GetProxyAddress() != "http://localhost:1337" {
		t.Error("WebsocketSetup")
	}

	if ws.trafficTimeout != time.Second*5 {
		t.Error("WebsocketSetup")
	}
	// -- Not connected shutdown
	err = ws.Shutdown()
	if err == nil {
		t.Fatal("should not be connected to able to shut down")
	}

	err = ws.SetWebsocketURL("", false, false)
	if err != nil {
		t.Fatal(err)
	}
	err = ws.SetWebsocketURL("ws://demos.kaazing.com/echo", false, false)
	if err != nil {
		t.Fatal(err)
	}
	err = ws.SetWebsocketURL("", true, false)
	if err != nil {
		t.Fatal(err)
	}
	err = ws.SetWebsocketURL("ws://demos.kaazing.com/echo", true, false)
	if err != nil {
		t.Fatal(err)
	}
	// Attempt reconnect
	err = ws.SetWebsocketURL("ws://demos.kaazing.com/echo", true, true)
	if err != nil {
		t.Fatal(err)
	}
	// -- initiate the reconnect which is usually handled by connection monitor
	err = ws.Connect()
	if err == nil {
		t.Fatal("should already be connected")
	}
	// -- Normal shutdown
	ws.Wg.Wait()
}