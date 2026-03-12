package server

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

const defaultResetBackupDir = "output/cutover"

func startupPreflightMessage(cfg config.Config, commandName string) (string, error) {
	inspection, err := store.InspectDataDir(cfg.DataDir)
	if err != nil {
		return "", err
	}

	switch inspection.Status {
	case store.DataDirStatusLegacy:
		return fmt.Sprintf(`Balanced 1.0 requires a fresh data directory before startup.

Detected legacy persisted state at %s.

Recovery:
  1. Run %s reset-data
     This archives the current data directory to %s and recreates %s.
  2. Start the server again with %s server

Alternative:
  Set DATA_DIR to a different empty directory before starting the server.`, cfg.DataDir, commandName, defaultResetBackupDir, cfg.DataDir, commandName), nil
	case store.DataDirStatusUnsupported:
		return fmt.Sprintf(`Balanced 1.0 cannot open the data directory at %s.

Detected unsupported storage schema %q.

Recovery:
  1. Run %s reset-data
     This archives the current data directory to %s and recreates %s.
  2. Start the server again with %s server`, cfg.DataDir, inspection.SchemaVersion, commandName, defaultResetBackupDir, cfg.DataDir, commandName), nil
	default:
		return "", nil
	}
}

func currentCommandName() string {
	if base := filepath.Base(os.Args[0]); base != "" && base != "." {
		if os.Args[0] == base {
			return base
		}
		return os.Args[0]
	}
	return "spartan"
}
