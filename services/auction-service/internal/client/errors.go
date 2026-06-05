package client

import "errors"

var (
	ErrGoodsShopMismatch    = errors.New("goods shop mismatch")
	ErrLiveRoomShopMismatch = errors.New("live room shop mismatch")
)
