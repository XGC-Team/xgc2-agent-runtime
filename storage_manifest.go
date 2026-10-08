package agentruntime

import (
	_ "embed"
	"encoding/json"

	"github.com/XGC-Team/xgc2-storage/api"
)

//go:embed storage-manifest.json
var manifestJSON []byte

// DeploymentManifest is a reviewed input for explicit storage initialization.
// Loading a broker never registers schemas or initializes a database.
func DeploymentManifest() api.Manifest {
	var manifest api.Manifest
	if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
		panic(err)
	}
	return manifest
}
