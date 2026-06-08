import { Popup } from 'antd-mobile'
import { X } from 'lucide-react'
import { formatCentAmount, formatTimeText } from '../../lib/format'
import type { UserLiveAuction, UserLiveGoods, UserLiveShop } from '../../types/domain'

type GoodsDrawerProps = {
  visible: boolean
  goods?: UserLiveGoods | null
  auction?: UserLiveAuction | null
  shop?: UserLiveShop | null
  onClose: () => void
}

export function GoodsDrawer({ visible, goods, auction, shop, onClose }: GoodsDrawerProps) {
  return (
    <Popup visible={visible} onMaskClick={onClose} bodyClassName="goods-drawer" position="bottom">
      <header className="goods-drawer__header">
        <h2>竞拍商品</h2>
        <button type="button" aria-label="关闭商品详情" onClick={onClose}>
          <X size={20} aria-hidden="true" />
        </button>
      </header>
      {goods ? (
        <div className="goods-drawer__body">
          {goods.cover_url && <img className="goods-drawer__cover" src={goods.cover_url} alt="" />}
          <h3>{goods.title}</h3>
          {shop?.name && <p className="goods-drawer__shop">{shop.name}</p>}
          {goods.description && <p className="goods-drawer__description">{goods.description}</p>}
          {auction && (
            <dl className="goods-drawer__facts">
              <div>
                <dt>起拍价</dt>
                <dd>{formatCentAmount(auction.start_price)}</dd>
              </div>
              <div>
                <dt>当前价</dt>
                <dd>{formatCentAmount(auction.current_price)}</dd>
              </div>
              <div>
                <dt>加价幅度</dt>
                <dd>{formatCentAmount(auction.bid_increment)}</dd>
              </div>
              <div>
                <dt>结束时间</dt>
                <dd>{formatTimeText(auction.end_time)}</dd>
              </div>
            </dl>
          )}
        </div>
      ) : (
        <p className="goods-drawer__empty">当前直播间暂未展示竞拍商品</p>
      )}
    </Popup>
  )
}
