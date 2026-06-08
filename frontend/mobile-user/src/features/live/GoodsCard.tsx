import { ChevronRight, ShoppingBag } from 'lucide-react'
import type { UserLiveGoods } from '../../types/domain'

type GoodsCardProps = {
  goods?: UserLiveGoods | null
  onOpen: () => void
}

export function GoodsCard({ goods, onOpen }: GoodsCardProps) {
  if (!goods) {
    return null
  }

  return (
    <button type="button" className="goods-card" onClick={onOpen}>
      <span className="goods-card__cover">
        {goods.cover_url ? (
          <img src={goods.cover_url} alt="" />
        ) : (
          <ShoppingBag size={22} aria-hidden="true" />
        )}
      </span>
      <span className="goods-card__body">
        <span>{goods.title}</span>
        <small>查看竞拍商品</small>
      </span>
      <ChevronRight size={16} aria-hidden="true" />
    </button>
  )
}
