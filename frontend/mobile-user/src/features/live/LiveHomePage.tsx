import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Button, DotLoading, Toast } from 'antd-mobile'
import { useInfiniteQuery } from '@tanstack/react-query'
import { RefreshCw } from 'lucide-react'
import { Swiper, SwiperSlide } from 'swiper/react'
import { Keyboard, Mousewheel } from 'swiper/modules'
import type { Swiper as SwiperInstance } from 'swiper'
import 'swiper/css'
import { AuctionPanel } from './AuctionPanel'
import { GoodsCard } from './GoodsCard'
import { GoodsDrawer } from './GoodsDrawer'
import { LiveMessageList } from './LiveMessageList'
import { LiveRoomMeta } from './LiveRoomMeta'
import { LiveVideo } from './LiveVideo'
import { SearchOverlay } from './SearchOverlay'
import { mapIncomingLiveEvent } from './eventMapper'
import { cx, getErrorText } from '../../lib/format'
import { getUserLiveAuctionSnapshot, getUserLiveEntry, getUserLiveFeed } from '../../services/liveApi'
import { openLiveSocket, type LiveSocketClient } from '../../services/liveSocket'
import { useAuthStore } from '../../stores/authStore'
import type {
  BidEventMessage,
  EntityID,
  UserLiveEntry,
  UserLiveFeedItem,
  UserLiveRuntime,
  UserLiveStats,
} from '../../types/domain'

const pageSize = 8
const preloadThreshold = 2

type RoomSession = {
  roomID: EntityID | null
  status: 'preview' | 'entering' | 'entered' | 'error'
  entry: UserLiveEntry | null
  runtime: UserLiveRuntime | null
  messages: BidEventMessage[]
  error: string
}

export function LiveHomePage() {
  const token = useAuthStore((state) => state.token)
  const isLoggedIn = Boolean(token)
  const [activeIndex, setActiveIndex] = useState(0)
  const [session, setSession] = useState<RoomSession>({
    roomID: null,
    status: 'preview',
    entry: null,
    runtime: null,
    messages: [],
    error: '',
  })
  const [statsByRoom, setStatsByRoom] = useState<Record<EntityID, UserLiveStats>>({})
  const [isGoodsOpen, setIsGoodsOpen] = useState(false)
  const [isBidding, setIsBidding] = useState(false)
  const socketRef = useRef<LiveSocketClient | null>(null)

  const feedQuery = useInfiniteQuery({
    queryKey: ['user-live-feed'],
    initialPageParam: 1,
    queryFn: ({ pageParam, signal }) => getUserLiveFeed(pageParam, pageSize, signal),
    getNextPageParam: (lastPage, allPages) => {
      const loadedCount = allPages.reduce((sum, pageData) => sum + pageData.list.length, 0)
      if (lastPage.list.length < pageSize || loadedCount >= lastPage.total) {
        return undefined
      }

      return lastPage.page + 1
    },
    retry: 1,
  })
  const rooms = useMemo(
    () => mergeRooms([], feedQuery.data?.pages.flatMap((pageData) => pageData.list) || []),
    [feedQuery.data],
  )
  const activeRoom = rooms[activeIndex]
  const isEntered = session.status === 'entered' && session.roomID === activeRoom?.room.id
  const activeEntry = isEntered ? session.entry : null
  const activeAuction = activeEntry?.current_auction || activeEntry?.auction || null
  const activeRuntime = session.runtime || activeEntry?.runtime || null
  const canBid = Boolean(isLoggedIn && activeEntry?.viewer?.can_bid !== false)

  useEffect(() => {
    return () => {
      socketRef.current?.close()
    }
  }, [])

  const activeDisplayItem = useMemo(() => {
    if (!activeRoom) {
      return null
    }

    const roomStats = statsByRoom[activeRoom.room.id]
    if (!roomStats) {
      return activeRoom
    }

    return {
      ...activeRoom,
      stats: {
        ...activeRoom.stats,
        ...roomStats,
      },
    }
  }, [activeRoom, statsByRoom])

  const leaveRoom = useCallback(() => {
    socketRef.current?.close()
    socketRef.current = null
    setIsBidding(false)
    setSession({
      roomID: null,
      status: 'preview',
      entry: null,
      runtime: null,
      messages: [],
      error: '',
    })
  }, [])

  async function handleEnterRoom() {
    if (!activeRoom || session.status === 'entering') {
      return
    }

    const roomID = activeRoom.room.id
    socketRef.current?.close()
    socketRef.current = null
    setSession({
      roomID,
      status: 'entering',
      entry: null,
      runtime: null,
      messages: [],
      error: '',
    })

    try {
      const entry = await getUserLiveEntry(roomID, token)
      const auction = entry.current_auction || entry.auction || null
      setSession({
        roomID,
        status: 'entered',
        entry,
        runtime: entry.runtime || null,
        messages: [createMessage('system', '已进入直播间', 'enter')],
        error: '',
      })

      if (entry.ws?.url) {
        socketRef.current = openLiveSocket(entry.ws.url, entry.ws.room_id || roomID, token, {
          onOpen: () => {
            appendMessage(createMessage('system', '互动连接已建立', 'ws_open'))
          },
          onError: () => {
            appendMessage(createMessage('error', '互动连接异常，价格以页面展示为准', 'ws_error'))
          },
          onClose: () => {
            setIsBidding(false)
          },
          onEvent: (event) => {
            const patch = mapIncomingLiveEvent(event, auction)
            if (event.type === 'auction_started') {
              void refreshAuctionSnapshot(roomID)
            }
            if (patch.bidResolved) {
              setIsBidding(false)
            }
            if (patch.bidError) {
              Toast.show(patch.bidError)
            }
            if (patch.runtime) {
              setSession((current) =>
                current.roomID === roomID
                  ? {
                      ...current,
                      runtime: {
                        ...current.runtime,
                        ...patch.runtime,
                      },
                    }
                  : current,
              )
            }
            if (patch.stats) {
              setStatsByRoom((current) => ({
                ...current,
                [roomID]: {
                  ...current[roomID],
                  ...patch.stats,
                },
              }))
            }
            if (patch.message) {
              appendMessage(patch.message)
            }
          },
        })
      }
    } catch (err) {
      setSession({
        roomID,
        status: 'error',
        entry: null,
        runtime: null,
        messages: [],
        error: getErrorText(err, '进入直播间失败，请稍后重试'),
      })
    }
  }

  async function refreshAuctionSnapshot(roomID: EntityID) {
    try {
      const snapshot = await getUserLiveAuctionSnapshot(roomID, token)
      setSession((current) => {
        if (current.roomID !== roomID || current.status !== 'entered' || !current.entry) {
          return current
        }

        return {
          ...current,
          entry: {
            ...current.entry,
            viewer: snapshot.viewer || current.entry.viewer,
            current_auction: snapshot.current_auction ?? snapshot.auction ?? null,
            auction: snapshot.auction ?? snapshot.current_auction ?? null,
            goods: snapshot.goods ?? null,
            runtime: snapshot.runtime ?? null,
          },
          runtime: snapshot.runtime ?? null,
        }
      })
    } catch (err) {
      appendMessage(createMessage('error', getErrorText(err, '竞拍快照刷新失败'), 'snapshot_error'))
    }
  }

  function handleBid() {
    if (!activeAuction) {
      return
    }
    if (!isLoggedIn) {
      Toast.show('请先登录后参与出价')
      return
    }
    if (!canBid) {
      Toast.show('当前账号暂不能出价')
      return
    }

    const currentPrice = activeRuntime?.current_price ?? activeAuction.current_price
    const nextBidPrice = activeRuntime?.next_bid_price || activeAuction.next_bid_price || currentPrice + activeAuction.bid_increment
    if (!socketRef.current?.sendBid(activeAuction.id, nextBidPrice)) {
      Toast.show('互动连接未就绪，请稍后重试')
      return
    }

    setIsBidding(true)
  }

  function appendMessage(message: BidEventMessage) {
    setSession((current) => ({
      ...current,
      messages: [...current.messages, message].slice(-30),
    }))
  }

  function handleSlideChange(swiper: SwiperInstance) {
    if (swiper.activeIndex !== activeIndex) {
      leaveRoom()
      setIsGoodsOpen(false)
    }
    setActiveIndex(swiper.activeIndex)
    if (
      rooms.length - swiper.activeIndex <= preloadThreshold &&
      feedQuery.hasNextPage &&
      !feedQuery.isFetchingNextPage
    ) {
      void feedQuery.fetchNextPage().catch((err: unknown) => {
        Toast.show(getErrorText(err, '加载更多直播间失败'))
      })
    }
  }

  if (feedQuery.isLoading) {
    return (
      <section className="live-home live-home--state">
        <DotLoading />
        <p>正在加载直播间</p>
      </section>
    )
  }

  if (feedQuery.isError) {
    const feedError = getErrorText(feedQuery.error, '直播间加载失败，请稍后重试')
    return (
      <section className="live-home live-home--state">
        <h1>直播间加载失败</h1>
        <p>{feedError}</p>
        <Button color="primary" size="small" onClick={() => void feedQuery.refetch()}>
          <RefreshCw size={16} aria-hidden="true" />
          重新加载
        </Button>
      </section>
    )
  }

  if (rooms.length === 0) {
    return (
      <section className="live-home live-home--state">
        <h1>暂无直播间</h1>
        <p>当前还没有正在直播的竞拍间</p>
        <Button color="primary" size="small" onClick={() => void feedQuery.refetch()}>
          重新加载
        </Button>
      </section>
    )
  }

  return (
    <section className="live-home">
      <SearchOverlay />
      <Swiper
        className="live-swiper"
        direction="vertical"
        modules={[Keyboard, Mousewheel]}
        keyboard
        mousewheel
        onSlideChange={handleSlideChange}
      >
        {rooms.map((item) => {
          const isActive = item.room.id === activeRoom?.room.id
          const itemEntry = isActive ? activeEntry : null
          const displayItem = isActive && activeDisplayItem ? activeDisplayItem : item
          const auction = itemEntry?.current_auction || itemEntry?.auction || null
          const goods = itemEntry?.goods || null

          return (
            <SwiperSlide key={item.room.id}>
              <article className={cx('live-slide', isActive && isEntered && 'is-entered')}>
                <LiveVideo item={displayItem} stream={itemEntry?.stream} enabled={isActive} />
                <LiveRoomMeta item={displayItem} entry={itemEntry} />

                {!isEntered && (
                  <div className="live-preview-action">
                    <button
                      type="button"
                      disabled={session.status === 'entering' && session.roomID === item.room.id}
                      onClick={handleEnterRoom}
                    >
                      {session.status === 'entering' && session.roomID === item.room.id ? (
                        <>
                          <DotLoading />
                          进入中
                        </>
                      ) : (
                        '进入直播间'
                      )}
                    </button>
                    {session.status === 'error' && session.roomID === item.room.id && (
                      <p>{session.error}</p>
                    )}
                  </div>
                )}

                {isEntered && (
                  <>
                    <LiveMessageList messages={session.messages} />
                    <GoodsCard goods={goods} onOpen={() => setIsGoodsOpen(true)} />
                    <AuctionPanel
                      auction={auction}
                      runtime={activeRuntime}
                      canBid={canBid}
                      isBidding={isBidding}
                      onBid={handleBid}
                    />
                  </>
                )}
              </article>
            </SwiperSlide>
          )
        })}
      </Swiper>

      {feedQuery.isFetchingNextPage && (
        <div className="live-load-more">
          <DotLoading />
        </div>
      )}

      <GoodsDrawer
        visible={isGoodsOpen}
        goods={activeEntry?.goods}
        auction={activeAuction}
        shop={activeEntry?.shop}
        onClose={() => setIsGoodsOpen(false)}
      />
    </section>
  )
}

function createMessage(type: BidEventMessage['type'], text: string, prefix: string): BidEventMessage {
  const createdAt = Date.now()
  return {
    id: `${prefix}_${createdAt}`,
    type,
    text,
    createdAt,
  }
}

function mergeRooms(current: UserLiveFeedItem[], incoming: UserLiveFeedItem[]) {
  const seen = new Set(current.map((item) => item.room.id))
  const next = [...current]
  for (const item of incoming) {
    if (!seen.has(item.room.id)) {
      seen.add(item.room.id)
      next.push(item)
    }
  }

  return next
}
