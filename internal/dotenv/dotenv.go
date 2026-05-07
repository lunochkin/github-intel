package dotenv

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

// Load reads .env from the process working directory when the file exists.
// A missing file is ignored; malformed content returns an error.
func Load() error {
	switch _, err := os.Stat(".env"); {
	case errors.Is(err, os.ErrNotExist):
		return nil
	case err != nil:
		return err
	}

	return godotenv.Load(".env")
}
