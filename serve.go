package agentruntime

import (
	"context"
	"net/http"
)

// OpenOptions requires explicit storage capabilities. The host owns transport
// lifetime; this library neither discovers storage nor starts its provider.
type OpenOptions struct {
	Storage           *Store
	Settings          *Store
	BasePath          string
	Prepare           Prepare
	Factory           Factory
	EvaluateDecisions bool
}

func Open(mux *http.ServeMux, options OpenOptions) (*Broker, error) {
	if options.Storage == nil || options.Settings == nil {
		return nil, ErrUnavailable
	}
	profiles, _, err := options.Settings.profiles(context.Background())
	if err != nil {
		return nil, err
	}
	broker, err := NewBroker(options.Storage, profiles, options.Prepare, options.Factory)
	if err != nil {
		return nil, err
	}
	if err = ConfigureBroker(broker, BrokerOptions{Settings: options.Settings}); err != nil {
		broker.Close()
		return nil, err
	}
	if options.EvaluateDecisions {
		if err = broker.SetDecisionEvaluator(broker.EvaluateDriverDecision); err != nil {
			broker.Close()
			return nil, err
		}
	}
	if err = RegisterRoutes(mux, broker, options.BasePath); err != nil {
		broker.Close()
		return nil, err
	}
	return broker, nil
}
