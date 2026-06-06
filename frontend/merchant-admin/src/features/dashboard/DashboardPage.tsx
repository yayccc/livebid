import { useEffect, useState } from 'react'
import { listShopAuctions } from '../../services/auctionApi'
import { listShopGoods } from '../../services/goodsApi'
import type { Auction, Goods } from '../../types/domain'
import { formatCentAmount, formatCount, getErrorText } from '../../lib/format'
import { StateBlock } from '../../components/common/StateBlock'
import { StatusTag } from '../../components/common/StatusTag'
import { auctionStatusLabel, auctionStatusTone } from '../../lib/status'

type DashboardPageProps = {
  token: string
  shopName: string
  onOpenGoods: () => void
  onOpenAuctions: () => void
}

export function DashboardPage({
  token,
  shopName,
  onOpenGoods,
  onOpenAuctions,
}: DashboardPageProps) {
  const [goods, setGoods] = useState<Goods[]>([])
  const [auctions, setAuctions] = useState<Auction[]>([])
  const [error, setError] = useState('')
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    let isActive = true

    Promise.all([
      listShopGoods(token, { page: 1, pageSize: 50 }),
      listShopAuctions(token, { page: 1, pageSize: 50 }),
    ])
      .then(([goodsResp, auctionResp]) => {
        if (!isActive) {
          return
        }
        setGoods(goodsResp.list)
        setAuctions(auctionResp.list)
      })
      .catch((err) => {
        if (isActive) {
          setError(getErrorText(err, '工作台数据加载失败'))
        }
      })
      .finally(() => {
        if (isActive) {
          setIsLoading(false)
        }
      })

    return () => {
      isActive = false
    }
  }, [token])

  const runningAuctions = auctions.filter((auction) => auction.status === 1)
  const pendingAuctions = auctions.filter((auction) => auction.status === 0)
  const dealAmount = auctions
    .filter((auction) => auction.status === 2)
    .reduce((sum, auction) => sum + (auction.deal_price || auction.current_price || 0), 0)

  return (
    <>
      <div className="page-header">
        <div>
          <h1>{shopName || 'LiveBid 经营工作台'}</h1>
          <p>聚合商品、竞拍状态和商家侧关键动作。统计基于当前已加载分页数据。</p>
        </div>
        <button type="button" className="primary-inline" onClick={onOpenAuctions}>
          新建竞拍
        </button>
      </div>

      {error && (
        <StateBlock title="读取失败" description={error} actionLabel="重试" onAction={() => location.reload()} />
      )}

      <div className="metric-grid" aria-busy={isLoading}>
        <Metric label="商品总数" value={formatCount(goods.length)} hint="当前页已加载" />
        <Metric label="竞拍中" value={formatCount(runningAuctions.length)} hint="需要重点盯盘" tone="hot" />
        <Metric label="成交额" value={formatCentAmount(dealAmount)} hint="已成交活动估算" tone="gold" />
        <Metric label="待开始" value={formatCount(pendingAuctions.length)} hint="可手动开始" />
      </div>

      <div className="content-grid">
        <section className="panel">
          <div className="panel-header">
            <h2>竞拍活动</h2>
            <button type="button" className="ghost-button" onClick={onOpenAuctions}>
              查看全部
            </button>
          </div>
          <div className="compact-list">
            {auctions.slice(0, 6).map((auction) => (
              <button
                key={auction.id}
                type="button"
                className="compact-row"
                onClick={onOpenAuctions}
              >
                <span>#{auction.id}</span>
                <strong>{formatCentAmount(auction.current_price)}</strong>
                <StatusTag
                  label={auctionStatusLabel(auction.status)}
                  tone={auctionStatusTone(auction.status)}
                />
              </button>
            ))}
            {!isLoading && auctions.length === 0 && (
              <StateBlock title="暂无竞拍" description="创建竞拍后，这里会展示最近活动。" />
            )}
          </div>
        </section>

        <section className="panel">
          <div className="panel-header">
            <h2>商品管理</h2>
            <button type="button" className="ghost-button" onClick={onOpenGoods}>
              管理商品
            </button>
          </div>
          <div className="compact-list">
            {goods.slice(0, 6).map((item) => (
              <button key={item.id} type="button" className="compact-row" onClick={onOpenGoods}>
                <span>{item.title}</span>
                <strong>#{item.id}</strong>
              </button>
            ))}
            {!isLoading && goods.length === 0 && (
              <StateBlock title="暂无商品" description="先创建商品，再配置竞拍活动。" />
            )}
          </div>
        </section>
      </div>
    </>
  )
}

function Metric({
  label,
  value,
  hint,
  tone,
}: {
  label: string
  value: string
  hint: string
  tone?: 'hot' | 'gold'
}) {
  return (
    <section className={`metric-card ${tone || ''}`}>
      <span>{label}</span>
      <strong>{value}</strong>
      <small>{hint}</small>
    </section>
  )
}
