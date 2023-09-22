package log

import (
	"go.elara.ws/logger"
	"go.elara.ws/lure/internal/log"
)

// SetLogger sets LURE's global logger, which is disabled by default
func SetLogger(l logger.Logger) {
	log.Logger = l
}
