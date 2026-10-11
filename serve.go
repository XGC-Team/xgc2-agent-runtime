package agentruntime

import "context"

// OpenOptions requires explicit storage capabilities. The host owns the
// broker's transport: it calls the broker directly or mounts Handler on its
// edge. This library neither discovers storage nor starts its provider.
type OpenOptions struct {
	Storage     *Store
	Settings    *Store
	NativeFiles *NativeFiles
	Prepare     Prepare
	Factory     Factory
}

func Open(options OpenOptions) (*Broker, error) {
	if options.Storage == nil || options.Settings == nil {
		return nil, ErrUnavailable
	}
	profiles, _, err := options.Settings.profiles(context.Background())
	if err != nil {
		return nil, err
	}
	broker, err := NewBroker(options.Storage, profiles, options.Prepare, options.Factory, options.NativeFiles)
	if err != nil {
		return nil, err
	}
	if err = ConfigureBroker(broker, BrokerOptions{Settings: options.Settings}); err != nil {
		broker.Close()
		return nil, err
	}
	return broker, nil
}
