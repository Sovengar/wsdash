package layout

import "errors"

var (
	ErrBadSplit = errors.New("invalid split direction")
	ErrBadRatio = errors.New("ratio out of bounds")
)
