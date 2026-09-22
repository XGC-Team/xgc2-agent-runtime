package agentruntime

import "net/http"

// OpenOptions is the shared host assembly. A product supplies Prepare and may
// mount its own routes beside the runtime. It does not supply a second broker,
// event log, or provider settings file.
type OpenOptions struct {
	JournalDir        string
	SettingsFile      string
	BasePath          string
	Prepare           Prepare
	Factory           Factory
	EvaluateDecisions bool
}

// Open loads provider settings, starts one broker, and mounts its HTTP API.
func Open(mux *http.ServeMux, options OpenOptions) (*Broker, error) {
	settings := options.SettingsFile
	if settings == "" {
		var err error
		settings, err = DefaultSettingsPath()
		if err != nil {
			return nil, err
		}
	}
	profiles, err := LoadConfig(settings)
	if err != nil {
		return nil, err
	}
	broker, err := NewBroker(options.JournalDir, profiles, options.Prepare, options.Factory)
	if err != nil {
		return nil, err
	}
	if err = ConfigureBroker(broker, BrokerOptions{SettingsFile: settings}); err != nil {
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
