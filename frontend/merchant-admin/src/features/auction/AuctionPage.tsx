import type React from 'react'
import { useEffect, useMemo, useState } from 'react'
import { StateBlock } from '../../components/common/StateBlock'
import { StatusTag } from '../../components/common/StatusTag'
import {
  cancelAuction,
  createAuction,
  deleteAuction,
  finishAuction,
  getAuction,
  getAuctionCreateOptions,
  listBidRecords,
  listShopAuctions,
  startAuction,
} from '../../services/auctionApi'
import { batchGetGoods } from '../../services/goodsApi'
import type { Auction, AuctionDraft, BidRecord, Goods, LiveRoom, PageResult } from '../../types/domain'
import { compactTime, formatCentAmount, getErrorText } from '../../lib/format'
import { auctionStatusLabel, auctionStatusTone, liveRoomStatusLabel } from '../../lib/status'

const emptyAuctionDraft: AuctionDraft = {
  goods_id: '',
  room_id: '',
  start_price: '',
  bid_increment: '',
  seal_price: '',
  start_time: '',
  end_time: '',
}

const defaultAuctionDurationMinutes = 30

type AuctionPageProps = {
  token: string
}

export function AuctionPage({ token }: AuctionPageProps) {
  const [result, setResult] = useState<PageResult<Auction>>({
    total: 0,
    page: 1,
    page_size: 10,
    list: [],
  })
  const [statusFilter, setStatusFilter] = useState('')
  const [goodsMap, setGoodsMap] = useState<Record<string, Goods>>({})
  const [goodsOptions, setGoodsOptions] = useState<Goods[]>([])
  const [roomOptions, setRoomOptions] = useState<LiveRoom[]>([])
  const [selectedID, setSelectedID] = useState<string | null>(null)
  const [selectedAuction, setSelectedAuction] = useState<Auction | null>(null)
  const [bidRecords, setBidRecords] = useState<BidRecord[]>([])
  const [draft, setDraft] = useState<AuctionDraft>(emptyAuctionDraft)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [isLoading, setIsLoading] = useState(true)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false)

  function loadAuctions(nextPage = result.page) {
    setIsLoading(true)
    setError('')
    listShopAuctions(token, {
      page: nextPage,
      pageSize: result.page_size,
      status: statusFilter,
    })
      .then((resp) => {
        setResult(resp)
        setSelectedID((current) => current || resp.list[0]?.id || null)
        return hydrateGoods(resp.list)
      })
      .catch((err) => setError(getErrorText(err, '竞拍列表加载失败')))
      .finally(() => setIsLoading(false))
  }

  async function hydrateGoods(auctions: Auction[]) {
    const goodsIDs = Array.from(new Set(auctions.map((auction) => auction.goods_id)))
    const goodsList = await batchGetGoods(goodsIDs)
    setGoodsMap((current) => ({
      ...current,
      ...Object.fromEntries(goodsList.map((goods) => [goods.id, goods])),
    }))
  }

  async function refreshCreateOptions() {
    const options = await getAuctionCreateOptions(token)
    setGoodsOptions(options.goods)
    setRoomOptions(options.live_rooms)
  }

  useEffect(() => {
    let isActive = true
    Promise.all([
      listShopAuctions(token, { page: 1, pageSize: result.page_size }),
      getAuctionCreateOptions(token),
    ])
      .then(([auctionResp, options]) => {
        if (!isActive) {
          return
        }
        setResult(auctionResp)
        setGoodsOptions(options.goods)
        setRoomOptions(options.live_rooms)
        setSelectedID(auctionResp.list[0]?.id || null)
        if (auctionResp.list.length === 0) {
          setSelectedAuction(null)
          setBidRecords([])
        }
        return hydrateGoods(auctionResp.list)
      })
      .catch((err) => {
        if (isActive) {
          setError(getErrorText(err, '竞拍列表加载失败'))
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
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token])

  useEffect(() => {
    if (!selectedID) {
      return
    }

    let isActive = true
    Promise.all([getAuction(selectedID), listBidRecords(selectedID, 1, 20)])
      .then(([auction, bids]) => {
        if (!isActive) {
          return
        }
        setSelectedAuction(auction)
        setBidRecords(bids.list)
        hydrateGoods([auction]).catch(() => undefined)
      })
      .catch((err) => {
        if (isActive) {
          setError(getErrorText(err, '竞拍详情加载失败'))
        }
      })

    return () => {
      isActive = false
    }
  }, [selectedID])

  const selectedGoods = selectedAuction ? goodsMap[selectedAuction.goods_id] : undefined
  const selectedRoom = selectedAuction
    ? roomOptions.find((room) => room.id === selectedAuction.room_id)
    : undefined
  const runningCount = useMemo(
    () => result.list.filter((auction) => auction.status === 1).length,
    [result.list],
  )

  function openCreateDialog() {
    setDraft(createDefaultAuctionDraft())
    refreshCreateOptions().catch(() => undefined)
    setIsCreateDialogOpen(true)
  }

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (
      !draft.goods_id ||
      !draft.room_id ||
      !draft.start_price ||
      !draft.bid_increment ||
      !draft.start_time ||
      !draft.end_time
    ) {
      setError('商品、直播间、起拍价、加价幅度、开始时间和结束时间不能为空')
      return
    }
    if (!validTimeRange(draft.start_time, draft.end_time)) {
      setError('结束时间必须晚于开始时间')
      return
    }

    setIsSubmitting(true)
    setError('')
    setNotice('')
    try {
      const auction = await createAuction(token, draft)
      setNotice('竞拍已创建')
      setDraft(emptyAuctionDraft)
      setSelectedID(auction.id)
      setIsCreateDialogOpen(false)
      loadAuctions(1)
      refreshCreateOptions().catch(() => undefined)
    } catch (err) {
      setError(getErrorText(err, '创建竞拍失败'))
    } finally {
      setIsSubmitting(false)
    }
  }

  async function runAuctionAction(action: () => Promise<unknown>, successMessage: string) {
    setError('')
    setNotice('')
    try {
      await action()
      setNotice(successMessage)
      loadAuctions(result.page)
      refreshCreateOptions().catch(() => undefined)
      if (selectedID) {
        const [auction, bids] = await Promise.all([
          getAuction(selectedID),
          listBidRecords(selectedID, 1, 20),
        ])
        setSelectedAuction(auction)
        setBidRecords(bids.list)
      }
    } catch (err) {
      setError(getErrorText(err, '竞拍操作失败'))
    }
  }

  return (
    <>
      <div className="page-header">
        <div>
          <h1>竞拍管理</h1>
          <p>创建竞拍规则，管理开始、结束、取消和删除，右侧查看实时详情与出价记录。</p>
        </div>
        <div className="header-stats">
          <span>竞拍中 {runningCount}</span>
          <span>当前页 {result.list.length}</span>
          <button
            type="button"
            className="primary-inline"
            onClick={openCreateDialog}
          >
            新建竞拍
          </button>
        </div>
      </div>

      {error && <div className="notice danger">{error}</div>}
      {notice && <div className="notice success">{notice}</div>}

      <div className="auction-layout">
        <section className="panel">
          <div className="panel-header">
            <h2>竞拍活动</h2>
            <span className="muted">共 {result.total} 条</span>
          </div>

          <form
            className="toolbar"
            onSubmit={(event) => {
              event.preventDefault()
              loadAuctions(1)
            }}
          >
            <select
              className="select-input"
              value={statusFilter}
              onChange={(event) => setStatusFilter(event.target.value)}
            >
              <option value="">全部状态</option>
              <option value="0">待开始</option>
              <option value="1">竞拍中</option>
              <option value="2">已成交</option>
              <option value="3">已流拍</option>
              <option value="4">已取消</option>
            </select>
            <button type="submit" className="ghost-button">
              筛选
            </button>
            <button type="button" className="ghost-button" onClick={() => loadAuctions(result.page)}>
              刷新
            </button>
          </form>

          <div className="table-wrap" aria-busy={isLoading}>
            <table className="data-table auction-table">
              <thead>
                <tr>
                  <th>商品</th>
                  <th>起拍价</th>
                  <th>当前价</th>
                  <th>出价</th>
                  <th>状态</th>
                  <th>时间</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                {result.list.map((auction) => {
                  const goods = goodsMap[auction.goods_id]
                  return (
                    <tr
                      key={auction.id}
                      className={selectedID === auction.id ? 'selected' : ''}
                      onClick={() => setSelectedID(auction.id)}
                    >
                      <td>
                        <div className="goods-cell">
                          {goods?.cover_url ? (
                            <img src={goods.cover_url} alt="" />
                          ) : (
                            <span className="thumb-placeholder">品</span>
                          )}
                          <div>
                            <strong>{goods?.title || `商品 #${auction.goods_id}`}</strong>
                            <span>
                              竞拍 #{auction.id} · 直播间 #{auction.room_id}
                            </span>
                          </div>
                        </div>
                      </td>
                      <td>{formatCentAmount(auction.start_price)}</td>
                      <td className="price-cell">{formatCentAmount(auction.current_price)}</td>
                      <td>{auction.bid_count}</td>
                      <td>
                        <StatusTag
                          label={auctionStatusLabel(auction.status)}
                          tone={auctionStatusTone(auction.status)}
                        />
                      </td>
                      <td>
                        <span className="time-stack">
                          <span>{compactTime(auction.start_time)}</span>
                          <span>{compactTime(auction.end_time)}</span>
                        </span>
                      </td>
                      <td>
                        <div className="row-actions" onClick={(event) => event.stopPropagation()}>
                          <button
                            type="button"
                            onClick={() =>
                              runAuctionAction(
                                () => startAuction(token, auction.id),
                                '竞拍已开始',
                              )
                            }
                          >
                            开始
                          </button>
                          <button
                            type="button"
                            onClick={() =>
                              runAuctionAction(
                                () => finishAuction(token, auction.id),
                                '竞拍已结束',
                              )
                            }
                          >
                            结束
                          </button>
                          <button
                            type="button"
                            onClick={() =>
                              runAuctionAction(
                                () => cancelAuction(token, auction.id),
                                '竞拍已取消',
                              )
                            }
                          >
                            取消
                          </button>
                          <button
                            type="button"
                            className="danger"
                            onClick={() => {
                              if (confirm('确认逻辑删除该竞拍活动吗？')) {
                                runAuctionAction(
                                  () => deleteAuction(token, auction.id),
                                  '竞拍已删除',
                                )
                              }
                            }}
                          >
                            删除
                          </button>
                        </div>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
            {!isLoading && result.list.length === 0 && (
              <StateBlock title="暂无竞拍" description="请先选择商品并创建竞拍活动。" />
            )}
          </div>
        </section>

        <aside className="detail-column">
          <section className="panel detail-panel">
            <div className="panel-header">
              <h2>实时竞拍详情</h2>
              {selectedAuction && (
                <StatusTag
                  label={auctionStatusLabel(selectedAuction.status)}
                  tone={auctionStatusTone(selectedAuction.status)}
                />
              )}
            </div>
            {selectedAuction ? (
              <>
                <div className="detail-title">
                  <strong>{selectedGoods?.title || `商品 #${selectedAuction.goods_id}`}</strong>
                  <span>
                    竞拍 #{selectedAuction.id} · 直播间 #{selectedAuction.room_id}
                  </span>
                </div>
                <div className="current-price">
                  <span>当前价</span>
                  <strong>{formatCentAmount(selectedAuction.current_price)}</strong>
                </div>
                <dl className="detail-grid">
                  <div>
                    <dt>起拍价</dt>
                    <dd>{formatCentAmount(selectedAuction.start_price)}</dd>
                  </div>
                  <div>
                    <dt>加价幅度</dt>
                    <dd>{formatCentAmount(selectedAuction.bid_increment)}</dd>
                  </div>
                  <div>
                    <dt>封顶价</dt>
                    <dd>{formatCentAmount(selectedAuction.seal_price)}</dd>
                  </div>
                  <div>
                    <dt>出价次数</dt>
                    <dd>{selectedAuction.bid_count}</dd>
                  </div>
                  <div>
                    <dt>直播间</dt>
                    <dd>{selectedRoom?.title || `#${selectedAuction.room_id}`}</dd>
                  </div>
                  <div>
                    <dt>领先用户</dt>
                    <dd>{selectedAuction.winner_user_id || '-'}</dd>
                  </div>
                  <div>
                    <dt>成交价</dt>
                    <dd>{formatCentAmount(selectedAuction.deal_price)}</dd>
                  </div>
                </dl>
              </>
            ) : (
              <StateBlock title="未选择竞拍" description="点击左侧列表中的一行查看详情。" />
            )}
          </section>

          <section className="panel">
            <div className="panel-header">
              <h2>出价记录</h2>
              <span className="muted">最新优先</span>
            </div>
            <div className="bid-list">
              {bidRecords.map((record) => (
                <div key={record.id} className="bid-row">
                  <span>用户 {record.user_id}</span>
                  <strong>{formatCentAmount(record.bid_price)}</strong>
                  <small>{record.bid_time || '-'}</small>
                </div>
              ))}
              {selectedAuction && bidRecords.length === 0 && (
                <StateBlock title="暂无出价" description="竞拍开始后，用户出价会在这里展示。" />
              )}
            </div>
          </section>

        </aside>
      </div>

      {isCreateDialogOpen && (
        <div
          className="modal-backdrop"
          role="presentation"
          onMouseDown={(event) => {
            if (event.target === event.currentTarget && !isSubmitting) {
              setIsCreateDialogOpen(false)
            }
          }}
        >
          <section className="modal-panel" role="dialog" aria-modal="true" aria-labelledby="create-auction-title">
            <div className="panel-header">
              <h2 id="create-auction-title">新建竞拍</h2>
              <button
                type="button"
                className="ghost-button"
                disabled={isSubmitting}
                onClick={() => setIsCreateDialogOpen(false)}
              >
                关闭
              </button>
            </div>
            <form className="stack-form" onSubmit={handleSubmit}>
              <label className="field">
                <span className="field-label">商品 *</span>
                <select
                  className="select-input"
                  value={draft.goods_id}
                  autoFocus
                  onChange={(event) =>
                    setDraft((current) => ({ ...current, goods_id: event.target.value }))
                  }
                >
                  <option value="">选择商品</option>
                  {goodsOptions.map((goods) => (
                    <option key={goods.id} value={goods.id}>
                      {goods.title}
                    </option>
                  ))}
                </select>
              </label>
              <label className="field">
                <span className="field-label">直播间 *</span>
                <select
                  className="select-input"
                  value={draft.room_id}
                  onChange={(event) =>
                    setDraft((current) => ({ ...current, room_id: event.target.value }))
                  }
                >
                  <option value="">选择直播间</option>
                  {roomOptions.map((room) => (
                    <option key={room.id} value={room.id}>
                      {room.title} · {liveRoomStatusLabel(room.status)}
                    </option>
                  ))}
                </select>
              </label>
              <div className="form-grid">
                <MoneyInput
                  label="起拍价 *"
                  value={draft.start_price}
                  onChange={(value) => setDraft((current) => ({ ...current, start_price: value }))}
                />
                <MoneyInput
                  label="加价幅度 *"
                  value={draft.bid_increment}
                  onChange={(value) =>
                    setDraft((current) => ({ ...current, bid_increment: value }))
                  }
                />
              </div>
              <MoneyInput
                label="封顶价"
                value={draft.seal_price}
                onChange={(value) => setDraft((current) => ({ ...current, seal_price: value }))}
              />
              <label className="field">
                <span className="field-label">开始时间 *</span>
                <input
                  className="text-input"
                  type="datetime-local"
                  value={draft.start_time}
                  onChange={(event) =>
                    setDraft((current) => ({ ...current, start_time: event.target.value }))
                  }
                />
              </label>
              <label className="field">
                <span className="field-label">结束时间 *</span>
                <input
                  className="text-input"
                  type="datetime-local"
                  value={draft.end_time}
                  onChange={(event) =>
                    setDraft((current) => ({ ...current, end_time: event.target.value }))
                  }
                />
              </label>
              <div className="modal-actions">
                <button
                  type="button"
                  className="ghost-button"
                  disabled={isSubmitting}
                  onClick={() => setIsCreateDialogOpen(false)}
                >
                  取消
                </button>
                <button type="submit" className="primary-action" disabled={isSubmitting}>
                  {isSubmitting ? '创建中...' : '创建竞拍'}
                </button>
              </div>
            </form>
          </section>
        </div>
      )}
    </>
  )
}

function createDefaultAuctionDraft(): AuctionDraft {
  const startTime = roundToNextMinutes(new Date(), 5)
  const endTime = new Date(startTime.getTime() + defaultAuctionDurationMinutes * 60 * 1000)
  return {
    ...emptyAuctionDraft,
    start_time: toDateTimeLocalValue(startTime),
    end_time: toDateTimeLocalValue(endTime),
  }
}

function roundToNextMinutes(date: Date, minutes: number) {
  const result = new Date(date)
  result.setSeconds(0, 0)
  const step = minutes * 60 * 1000
  return new Date(Math.ceil(result.getTime() / step) * step)
}

function toDateTimeLocalValue(date: Date) {
  const year = date.getFullYear()
  const month = pad2(date.getMonth() + 1)
  const day = pad2(date.getDate())
  const hour = pad2(date.getHours())
  const minute = pad2(date.getMinutes())
  return `${year}-${month}-${day}T${hour}:${minute}`
}

function validTimeRange(startTime: string, endTime: string) {
  const start = new Date(startTime).getTime()
  const end = new Date(endTime).getTime()
  return Number.isFinite(start) && Number.isFinite(end) && end > start
}

function pad2(value: number) {
  return String(value).padStart(2, '0')
}

function MoneyInput({
  label,
  value,
  onChange,
}: {
  label: string
  value: string
  onChange: (value: string) => void
}) {
  return (
    <label className="field">
      <span className="field-label">{label}</span>
      <input
        className="text-input"
        value={value}
        inputMode="decimal"
        placeholder="金额，单位元"
        onChange={(event) => onChange(event.target.value)}
      />
    </label>
  )
}
