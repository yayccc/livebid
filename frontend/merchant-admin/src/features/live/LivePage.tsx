import type React from 'react'
import { useEffect, useMemo, useState } from 'react'
import { StateBlock } from '../../components/common/StateBlock'
import { StatusTag } from '../../components/common/StatusTag'
import {
  createLiveRoom,
  endLive,
  getLiveStreamInfo,
  listMerchantLiveRooms,
  startLive,
} from '../../services/liveApi'
import type { LiveRoom, LiveRoomDraft, LiveStreamInfo, PageResult } from '../../types/domain'
import { compactTime, getErrorText } from '../../lib/format'
import {
  liveRoomStatusLabel,
  liveRoomStatusTone,
  mediaStreamStatusLabel,
  mediaStreamStatusTone,
} from '../../lib/status'

const emptyLiveRoomDraft: LiveRoomDraft = {
  title: '',
  cover: '',
  description: '',
}

type LivePageProps = {
  token: string
}

type CreatedStreamNotice = {
  roomTitle: string
  initialStreamCode: string
  rtmpPushURL: string
  webrtcPlayURL: string
}

export function LivePage({ token }: LivePageProps) {
  const [result, setResult] = useState<PageResult<LiveRoom>>({
    total: 0,
    page: 1,
    page_size: 10,
    list: [],
  })
  const [statusFilter, setStatusFilter] = useState('')
  const [selectedID, setSelectedID] = useState<number | null>(null)
  const [streamInfo, setStreamInfo] = useState<LiveStreamInfo | null>(null)
  const [draft, setDraft] = useState<LiveRoomDraft>(emptyLiveRoomDraft)
  const [createdStream, setCreatedStream] = useState<CreatedStreamNotice | null>(null)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [isLoading, setIsLoading] = useState(true)
  const [isSubmitting, setIsSubmitting] = useState(false)

  const selectedRoom = useMemo(
    () => result.list.find((room) => room.id === selectedID) || null,
    [result.list, selectedID],
  )
  const livingCount = useMemo(
    () => result.list.filter((room) => room.status === 'living').length,
    [result.list],
  )
  const streamURL = streamInfo ? `${streamInfo.rtmp_push_url}?token=推流码` : ''

  function loadRooms(nextPage = result.page) {
    setIsLoading(true)
    setError('')
    listMerchantLiveRooms(token, {
      page: nextPage,
      pageSize: result.page_size,
      status: statusFilter,
    })
      .then((resp) => {
        setResult(resp)
        setSelectedID((current) => current || resp.list[0]?.id || null)
        if (resp.list.length === 0) {
          setStreamInfo(null)
        }
      })
      .catch((err) => setError(getErrorText(err, '直播间列表加载失败')))
      .finally(() => setIsLoading(false))
  }

  useEffect(() => {
    let isActive = true
    listMerchantLiveRooms(token, { page: 1, pageSize: result.page_size })
      .then((resp) => {
        if (!isActive) {
          return
        }
        setResult(resp)
        setSelectedID(resp.list[0]?.id || null)
      })
      .catch((err) => {
        if (isActive) {
          setError(getErrorText(err, '直播间列表加载失败'))
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
    getLiveStreamInfo(token, selectedID)
      .then((info) => {
        if (isActive) {
          setStreamInfo(info)
        }
      })
      .catch((err) => {
        if (isActive) {
          setError(getErrorText(err, '推流信息加载失败'))
          setStreamInfo(null)
        }
      })

    return () => {
      isActive = false
    }
  }, [selectedID, token])

  async function handleCreate(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const title = draft.title.trim()
    if (!title) {
      setError('直播间标题不能为空')
      return
    }

    setIsSubmitting(true)
    setError('')
    setNotice('')
    setCreatedStream(null)
    try {
      const resp = await createLiveRoom(token, { ...draft, title })
      setNotice('直播间已创建')
      setCreatedStream({
        roomTitle: resp.live_room.title,
        initialStreamCode: resp.initial_stream_code,
        rtmpPushURL: resp.rtmp_push_url,
        webrtcPlayURL: resp.webrtc_play_url,
      })
      setDraft(emptyLiveRoomDraft)
      setSelectedID(resp.live_room.id)
      loadRooms(1)
    } catch (err) {
      setError(getErrorText(err, '创建直播间失败'))
    } finally {
      setIsSubmitting(false)
    }
  }

  async function runLiveAction(action: () => Promise<LiveRoom>, successMessage: string) {
    setError('')
    setNotice('')
    try {
      const room = await action()
      setNotice(successMessage)
      setSelectedID(room.id)
      loadRooms(result.page)
    } catch (err) {
      setError(getErrorText(err, '直播间操作失败'))
    }
  }

  return (
    <>
      <div className="page-header">
        <div>
          <h1>直播管理</h1>
          <p>创建直播间，管理开播和关播，查看 OBS 推流地址与商家预览播放地址。</p>
        </div>
        <div className="header-stats">
          <span>直播中 {livingCount}</span>
          <span>当前页 {result.list.length}</span>
        </div>
      </div>

      {error && <div className="notice danger">{error}</div>}
      {notice && <div className="notice success">{notice}</div>}

      <div className="live-layout">
        <section className="panel">
          <div className="panel-header">
            <h2>直播间列表</h2>
            <span className="muted">共 {result.total} 条</span>
          </div>

          <form
            className="toolbar"
            onSubmit={(event) => {
              event.preventDefault()
              loadRooms(1)
            }}
          >
            <select
              className="select-input"
              value={statusFilter}
              onChange={(event) => setStatusFilter(event.target.value)}
            >
              <option value="">全部状态</option>
              <option value="not_live">未开播</option>
              <option value="living">直播中</option>
            </select>
            <button type="submit" className="ghost-button">
              筛选
            </button>
            <button type="button" className="ghost-button" onClick={() => loadRooms(result.page)}>
              刷新
            </button>
          </form>

          <div className="table-wrap" aria-busy={isLoading}>
            <table className="data-table live-table">
              <thead>
                <tr>
                  <th>直播间</th>
                  <th>直播状态</th>
                  <th>推流状态</th>
                  <th>时间</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                {result.list.map((room) => (
                  <tr
                    key={room.id}
                    className={selectedID === room.id ? 'selected' : ''}
                    onClick={() => setSelectedID(room.id)}
                  >
                    <td>
                      <div className="goods-cell">
                        {room.cover ? (
                          <img src={room.cover} alt="" />
                        ) : (
                          <span className="thumb-placeholder">播</span>
                        )}
                        <div>
                          <strong>{room.title}</strong>
                          <span>直播间 #{room.id}</span>
                        </div>
                      </div>
                    </td>
                    <td>
                      <StatusTag
                        label={liveRoomStatusLabel(room.status)}
                        tone={liveRoomStatusTone(room.status)}
                      />
                    </td>
                    <td>
                      <StatusTag
                        label={mediaStreamStatusLabel(room.media_stream_status)}
                        tone={mediaStreamStatusTone(room.media_stream_status)}
                      />
                    </td>
                    <td>
                      <span className="time-stack">
                        <span>{compactTime(room.actual_start_time)}</span>
                        <span>{compactTime(room.actual_end_time)}</span>
                      </span>
                    </td>
                    <td>
                      <div className="row-actions" onClick={(event) => event.stopPropagation()}>
                        <button
                          type="button"
                          onClick={() =>
                            runLiveAction(() => startLive(token, room.id), '直播间已开播')
                          }
                        >
                          开播
                        </button>
                        <button
                          type="button"
                          onClick={() =>
                            runLiveAction(() => endLive(token, room.id), '直播间已关播')
                          }
                        >
                          关播
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {!isLoading && result.list.length === 0 && (
              <StateBlock title="暂无直播间" description="先创建直播间，再在竞拍管理中选择它。" />
            )}
          </div>
        </section>

        <aside className="detail-column">
          <section className="panel detail-panel">
            <div className="panel-header">
              <h2>推流信息</h2>
              {selectedRoom && (
                <StatusTag
                  label={liveRoomStatusLabel(selectedRoom.status)}
                  tone={liveRoomStatusTone(selectedRoom.status)}
                />
              )}
            </div>

            {selectedRoom ? (
              <>
                <div className="detail-title">
                  <strong>{selectedRoom.title}</strong>
                  <span>直播间 #{selectedRoom.id}</span>
                </div>
                <dl className="detail-grid stream-detail-grid">
                  <div>
                    <dt>媒体流</dt>
                    <dd>{streamInfo?.stream_name || '-'}</dd>
                  </div>
                  <div>
                    <dt>推流状态</dt>
                    <dd>{mediaStreamStatusLabel(streamInfo?.media_stream_status)}</dd>
                  </div>
                  <div>
                    <dt>推流地址</dt>
                    <dd>{streamInfo?.rtmp_push_url || '-'}</dd>
                  </div>
                  <div>
                    <dt>OBS 完整地址</dt>
                    <dd>{streamURL || '-'}</dd>
                  </div>
                  <div>
                    <dt>预览地址</dt>
                    <dd>{streamInfo?.webrtc_play_url || '-'}</dd>
                  </div>
                </dl>
              </>
            ) : (
              <StateBlock title="未选择直播间" description="点击左侧列表中的一行查看推流信息。" />
            )}
          </section>

          {createdStream && (
            <section className="panel">
              <div className="panel-header">
                <h2>一次性推流码</h2>
                <StatusTag label="仅创建时返回" tone="accent" />
              </div>
              <dl className="detail-grid stream-detail-grid">
                <div>
                  <dt>直播间</dt>
                  <dd>{createdStream.roomTitle}</dd>
                </div>
                <div>
                  <dt>推流码</dt>
                  <dd>{createdStream.initialStreamCode}</dd>
                </div>
                <div>
                  <dt>推流地址</dt>
                  <dd>{createdStream.rtmpPushURL}</dd>
                </div>
                <div>
                  <dt>预览地址</dt>
                  <dd>{createdStream.webrtcPlayURL}</dd>
                </div>
              </dl>
            </section>
          )}

          <section className="panel">
            <div className="panel-header">
              <h2>新建直播间</h2>
            </div>
            <form className="stack-form" onSubmit={handleCreate}>
              <label className="field">
                <span className="field-label">标题 *</span>
                <input
                  className="text-input"
                  value={draft.title}
                  maxLength={128}
                  placeholder="例如：翡翠晚场专拍"
                  onChange={(event) =>
                    setDraft((current) => ({ ...current, title: event.target.value }))
                  }
                />
              </label>
              <label className="field">
                <span className="field-label">封面</span>
                <input
                  className="text-input"
                  value={draft.cover}
                  placeholder="直播间封面 URL"
                  onChange={(event) =>
                    setDraft((current) => ({ ...current, cover: event.target.value }))
                  }
                />
              </label>
              <label className="field">
                <span className="field-label">简介</span>
                <textarea
                  value={draft.description}
                  placeholder="本场直播主题、品类或开播说明"
                  onChange={(event) =>
                    setDraft((current) => ({ ...current, description: event.target.value }))
                  }
                />
              </label>
              <button type="submit" className="primary-action" disabled={isSubmitting}>
                {isSubmitting ? '创建中...' : '创建直播间'}
              </button>
            </form>
          </section>
        </aside>
      </div>
    </>
  )
}
