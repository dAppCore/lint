package lint

import (
	"io/fs"

	core "github.com/dappcore/go"
)

// isNotExistError recognises missing-file errors from stdlib and core media.
func isNotExistError(err error) bool {
	return err != nil && (core.IsNotExist(err) || core.Is(err, fs.ErrNotExist))
}
