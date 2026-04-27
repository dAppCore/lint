package lint

import (
	"io/fs"
	"os"

	core "dappco.re/go/core"
)

// isNotExistError recognises missing-file errors from stdlib and core media.
func isNotExistError(err error) bool {
	return err != nil && (os.IsNotExist(err) || core.Is(err, fs.ErrNotExist))
}
