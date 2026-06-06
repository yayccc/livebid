package client

import "errors"

var (
	ErrGoodsShopMismatch    = errors.New("商品不存在或不属于当前商铺")
	ErrLiveRoomShopMismatch = errors.New("直播间不存在或不属于当前商铺")
)
