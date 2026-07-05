import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Button, DotLoading, Toast } from 'antd-mobile'
import { useInfiniteQuery } from '@tanstack/react-query'
import { Gavel, RefreshCw, SendHorizontal, Smile } from 'lucide-react'
import { Swiper, SwiperSlide } from 'swiper/react'
import { Keyboard, Mousewheel } from 'swiper/modules'
import type { Swiper as SwiperInstance } from 'swiper'
import { useLocation, useNavigate } from 'react-router-dom'
import 'swiper/css'
import { AuctionPanel } from './AuctionPanel'
import { AuctionRecordsDrawer } from './AuctionRecordsDrawer'
import { LiveMessageList } from './LiveMessageList'
import { LiveRoomMeta } from './LiveRoomMeta'
import { LiveVideo } from './LiveVideo'
import { SearchOverlay } from './SearchOverlay'
import { mapIncomingLiveEvent } from './eventMapper'
import { cx, formatCentAmount, getErrorText } from '../../lib/format'
import {
  batchGetGoods,
  getGoods,
  getUserLiveAuctionRecords,
  getUserLiveAuctionSnapshot,
  getUserLiveEntry,
  getUserLiveFeed,
} from '../../services/liveApi'
import { openLiveSocket, type LiveSocketClient } from '../../services/liveSocket'
import { getUserProfile } from '../../services/userApi'
import { useAuthStore } from '../../stores/authStore'
import type {
  BidEventMessage,
  EntityID,
  UserProfile,
  UserLiveAuction,
  UserLiveAuctionRecord,
  UserLiveEntry,
  UserLiveFeedItem,
  UserLiveGoods,
  UserLiveRuntime,
  UserLiveStats,
} from '../../types/domain'

const pageSize = 8
const preloadThreshold = 2
const returnRoomStorageKey = 'livebid:return-room-id'

type RoomSession = {
  roomID: EntityID | null
  status: 'preview' | 'entering' | 'entered' | 'error'
  entry: UserLiveEntry | null
  runtime: UserLiveRuntime | null
  messages: BidEventMessage[]
  auctionRecords: UserLiveAuctionRecord[]
  error: string
}

type DealDialogState = {
  auctionID: EntityID
  role: 'winner' | 'viewer'
  goods?: UserLiveGoods | null
  title: string
  price: number
  winnerID?: EntityID
  winnerName?: string
  winnerAvatar?: string
  rounds: number
  purchaseExpireAt: number
} | null

type LocalLeadingBid = {
  auctionID: EntityID
  winnerID?: EntityID
}

type LiveHomeLocationState = {
  returnRoomID?: EntityID
} | null

export function LiveHomePage() {
  const navigate = useNavigate()
  const location = useLocation()
  const token = useAuthStore((state) => state.token)
  const userID = useAuthStore((state) => state.userID)
  const profile = useAuthStore((state) => state.profile)
  const isLoggedIn = Boolean(token)
  const [activeIndex, setActiveIndex] = useState(0)
  const [session, setSession] = useState<RoomSession>({
    roomID: null,
    status: 'preview',
    entry: null,
    runtime: null,
    messages: [],
    auctionRecords: [],
    error: '',
  })
  const [statsByRoom, setStatsByRoom] = useState<Record<EntityID, UserLiveStats>>({})
  const [isAuctionRecordsOpen, setIsAuctionRecordsOpen] = useState(false)
  const [dismissedAuctionCardByRoom, setDismissedAuctionCardByRoom] = useState<Record<EntityID, string>>({})
  const [isBidding, setIsBidding] = useState(false)
  const [isSendingDanmaku, setIsSendingDanmaku] = useState(false)
  const [dealDialog, setDealDialog] = useState<DealDialogState>(null)
  const [dealNow, setDealNow] = useState(Date.now())
  const [chatDraft, setChatDraft] = useState('')
  const bidTimeoutRef = useRef<number | null>(null)
  const danmakuTimeoutRef = useRef<number | null>(null)
  const pendingBidRef = useRef<LocalLeadingBid | null>(null)
  const localLeadingBidRef = useRef<LocalLeadingBid | null>(null)
  const publicUserCacheRef = useRef<Map<EntityID, UserProfile>>(new Map())
  const socketRef = useRef<LiveSocketClient | null>(null)
  const activeAuctionRef = useRef<UserLiveAuction | null>(null)
  const swiperRef = useRef<SwiperInstance | null>(null)
  const restoringRoomRef = useRef<EntityID | null>(null)
  const returnRoomID = (location.state as LiveHomeLocationState)?.returnRoomID || getStoredReturnRoomID()

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
  const isLiving = isLiveRoom(activeEntry?.room.status, activeEntry?.room.status_text)
  const isAuctionRunning = isAuctionActive(activeRuntime?.status ?? activeAuction?.status)
  const canSubmitBid = Boolean(isEntered && isLiving && isAuctionRunning && canBid)

  useEffect(() => {
    activeAuctionRef.current = activeAuction
  }, [activeAuction])

  useEffect(() => {
    if (dealDialog?.role !== 'winner') {
      return undefined
    }

    setDealNow(Date.now())
    const timer = window.setInterval(() => {
      setDealNow(Date.now())
    }, 1000)

    return () => window.clearInterval(timer)
  }, [dealDialog?.role, dealDialog?.auctionID])

  useEffect(() => {
    return () => {
      if (bidTimeoutRef.current !== null) {
        window.clearTimeout(bidTimeoutRef.current)
        bidTimeoutRef.current = null
      }
      if (danmakuTimeoutRef.current !== null) {
        window.clearTimeout(danmakuTimeoutRef.current)
        danmakuTimeoutRef.current = null
      }
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
    if (bidTimeoutRef.current !== null) {
      window.clearTimeout(bidTimeoutRef.current)
      bidTimeoutRef.current = null
    }
    if (danmakuTimeoutRef.current !== null) {
      window.clearTimeout(danmakuTimeoutRef.current)
      danmakuTimeoutRef.current = null
    }
    pendingBidRef.current = null
    localLeadingBidRef.current = null
    setIsBidding(false)
    setIsSendingDanmaku(false)
    setChatDraft('')
    setSession({
      roomID: null,
      status: 'preview',
      entry: null,
      runtime: null,
      messages: [],
      auctionRecords: [],
      error: '',
    })
    setIsAuctionRecordsOpen(false)
  }, [])

  async function handleEnterRoom(targetRoomID = activeRoom?.room.id) {
    if (!targetRoomID || session.status === 'entering') {
      return
    }

    const roomID = targetRoomID
    socketRef.current?.close()
    socketRef.current = null
    setSession({
      roomID,
      status: 'entering',
      entry: null,
      runtime: null,
      messages: [],
      auctionRecords: [],
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
        auctionRecords: buildAuctionRecordsFromEntry(entry),
        error: '',
      })
      void refreshAuctionRecords(roomID, entry.goods)

      if (entry.ws?.url) {
        socketRef.current = openLiveSocket(entry.ws.url, entry.ws.room_id || roomID, token, {
          onOpen: () => {
            appendMessage(createMessage('system', '互动连接已建立', 'ws_open'), roomID)
          },
          onError: () => {
            appendMessage(createMessage('error', '互动连接异常，价格以页面展示为准', 'ws_error'), roomID)
          },
          onClose: () => {
            setIsBidding(false)
            setIsSendingDanmaku(false)
            clearDanmakuTimeout()
          },
          onEvent: (event) => {
            const eventType = event.type || event.event_type
            const liveAuction = activeAuctionRef.current || auction
            const patch = mapIncomingLiveEvent(event, liveAuction)
            if (isAuctionRecordEvent(eventType)) {
              void refreshAuctionSnapshot(roomID)
              void refreshAuctionRecords(roomID)
            }
            if (patch.bidResolved) {
              clearBidTimeout()
              setIsBidding(false)
              if (patch.bidError) {
                pendingBidRef.current = null
              } else {
                updateLocalLeadingBidFromBidResponse(event, liveAuction)
              }
              void refreshAuctionSnapshot(roomID)
              void refreshAuctionRecords(roomID)
            }
            if (patch.bidError) {
              Toast.show(patch.bidError)
            }
            if (patch.danmakuResolved) {
              clearDanmakuTimeout()
              setIsSendingDanmaku(false)
              if (patch.danmakuError) {
                Toast.show(patch.danmakuError)
              } else {
                setChatDraft('')
              }
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
            if (patch.messages) {
              appendMessages(patch.messages, roomID)
            }
            if (patch.message) {
              appendMessage(patch.message, roomID)
            }
            if (patch.auctionRecord) {
              updateLocalLeadingBidFromAuctionRecord(patch.auctionRecord)
              appendAuctionRecord(patch.auctionRecord)
            }
            if (isDealRecordEvent(eventType)) {
              const dealRecord = patch.auctionRecord || buildDealRecordFromEvent(event, liveAuction)
              if (dealRecord) {
                openDealDialog(dealRecord, liveAuction, entry.goods)
              }
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
        auctionRecords: [],
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

  async function refreshAuctionRecords(roomID: EntityID, currentGoods = activeEntry?.goods) {
    try {
      const page = await getUserLiveAuctionRecords(roomID, token)
      const [goodsByID, winnerNamesByID] = await Promise.all([
        fetchMissingAuctionGoods(page.list),
        fetchMissingAuctionWinnerNames(page.list, publicUserCacheRef.current),
      ])
      setSession((current) => {
        if (current.roomID !== roomID || current.status !== 'entered') {
          return current
        }
        const records = mergeAuctionRecordsWinners(mergeAuctionRecordsGoods(page.list, goodsByID), winnerNamesByID)
        const currentAuction = current.entry?.current_auction || current.entry?.auction || null
        const currentRecord = currentAuction ? records.find((record) => record.id === currentAuction.id) : null
        const nextAuction =
          current.entry && currentAuction && currentRecord
            ? {
                ...currentAuction,
                status: currentRecord.status,
                status_text: currentRecord.status_text || currentAuction.status_text,
                current_price: currentRecord.current_price,
                next_bid_price: currentRecord.next_bid_price,
                bid_count: currentRecord.bid_count,
                winner_user_id: currentRecord.winner_user_id,
                winner_display_name: currentRecord.winner_display_name,
                deal_price: currentRecord.deal_price,
                end_time: currentRecord.end_time || currentAuction.end_time,
                server_time: currentRecord.server_time,
                expire_at: currentRecord.expire_at,
                version: currentRecord.version,
                countdown_received_at: currentRecord.countdown_received_at,
              }
            : currentAuction

        return {
          ...current,
          entry: current.entry
            ? {
                ...current.entry,
                current_auction: nextAuction,
                auction: nextAuction,
              }
            : current.entry,
          auctionRecords: mergeCurrentGoodsIntoAuctionRecords(records, nextAuction, currentGoods || current.entry?.goods),
        }
      })
    } catch (err) {
      appendMessage(createMessage('error', getErrorText(err, '竞拍记录刷新失败'), 'auction_records_error'))
    }
  }

  function handleBid(bidPrice: number) {
    if (!isEntered) {
      Toast.show('请先进入直播间')
      return
    }
    if (!isLiving) {
      Toast.show('当前直播间还未开播')
      return
    }
    if (!activeAuction || !isAuctionRunning) {
      if (activeRoom?.room.id) {
        void refreshAuctionSnapshot(activeRoom.room.id)
      }
      Toast.show('当前还没有开始竞拍')
      return
    }
    if (!isLoggedIn) {
      Toast.show('请先登录后参与出价')
      return
    }
    if (!canSubmitBid) {
      Toast.show('当前账号暂不能出价')
      return
    }

    const currentPrice = activeRuntime?.current_price ?? activeAuction.current_price
    const minBidPrice = activeRuntime?.next_bid_price || activeAuction.next_bid_price || currentPrice + activeAuction.bid_increment
    if (!Number.isSafeInteger(bidPrice) || bidPrice < minBidPrice) {
      Toast.show(`出价不能低于 ${formatCentAmount(minBidPrice)}`)
      return
    }

    const requestID = socketRef.current?.sendBid(activeAuction.id, bidPrice)
    if (!requestID) {
      Toast.show('互动连接未就绪，请稍后重试')
      return
    }

    pendingBidRef.current = { auctionID: activeAuction.id }
    setIsBidding(true)
    clearBidTimeout()
    bidTimeoutRef.current = window.setTimeout(() => {
      setIsBidding(false)
      pendingBidRef.current = null
      Toast.show('出价响应超时，请刷新竞拍状态后重试')
      if (activeRoom?.room.id) {
        void refreshAuctionSnapshot(activeRoom.room.id)
      }
    }, 8000)
  }

  function handleSendChat() {
    if (!isEntered) {
      return
    }

    const text = chatDraft.trim()
    if (!text) {
      Toast.show('请输入弹幕内容')
      return
    }
    if (Array.from(text).length > 30) {
      Toast.show('弹幕最多 30 个字符')
      return
    }
    if (!isLoggedIn) {
      Toast.show('请先登录后发送弹幕')
      return
    }
    if (isSendingDanmaku) {
      return
    }

    const requestID = socketRef.current?.sendDanmaku(text)
    if (!requestID) {
      Toast.show('互动连接未就绪，请稍后重试')
      return
    }

    setIsSendingDanmaku(true)
    clearDanmakuTimeout()
    danmakuTimeoutRef.current = window.setTimeout(() => {
      setIsSendingDanmaku(false)
      Toast.show('弹幕发送超时，请稍后重试')
    }, 8000)
  }

  function clearBidTimeout() {
    if (bidTimeoutRef.current !== null) {
      window.clearTimeout(bidTimeoutRef.current)
      bidTimeoutRef.current = null
    }
  }

  function clearDanmakuTimeout() {
    if (danmakuTimeoutRef.current !== null) {
      window.clearTimeout(danmakuTimeoutRef.current)
      danmakuTimeoutRef.current = null
    }
  }

  function appendMessage(message: BidEventMessage, roomID?: EntityID) {
    appendMessages([message], roomID)
  }

  function appendMessages(messages: BidEventMessage[], roomID?: EntityID) {
    if (messages.length === 0) {
      return
    }

    setSession((current) => {
      if (roomID && (current.roomID !== roomID || current.status !== 'entered')) {
        return current
      }

      return {
        ...current,
        messages: mergeMessages(current.messages, messages).slice(-30),
      }
    })

    for (const message of messages) {
      if (message.type === 'bid' && message.userID && !message.displayName) {
        void hydrateBidMessageUser(message)
      }
    }
  }

  async function hydrateBidMessageUser(message: BidEventMessage) {
    if (!message.userID) {
      return
    }

    const cachedProfile = publicUserCacheRef.current.get(message.userID)
    const displayName = cachedProfile ? userDisplayName(cachedProfile) : await fetchPublicUserDisplayName(message.userID)
    if (!displayName) {
      return
    }

    setSession((current) => ({
      ...current,
      messages: current.messages.map((item) =>
        item.id === message.id
          ? {
              ...item,
              displayName,
              text: `${displayName} 出价 ${formatCentAmount(item.bidPrice || message.bidPrice || 0)}`,
            }
          : item,
      ),
    }))
  }

  async function fetchPublicUserDisplayName(userID: EntityID) {
    try {
      const profile = await getUserProfile(userID)
      publicUserCacheRef.current.set(userID, profile)
      return userDisplayName(profile)
    } catch {
      return ''
    }
  }

  function appendAuctionRecord(record: UserLiveAuctionRecord) {
    setSession((current) => ({
      ...current,
      auctionRecords: upsertAuctionRecord(current.auctionRecords, record),
    }))
  }

  function updateLocalLeadingBidFromBidResponse(
    event: { data?: unknown; request_id?: string; requestId?: string },
    auction: UserLiveAuction | null,
  ) {
    const pendingBid = pendingBidRef.current
    if (!pendingBid) {
      return
    }

    const data = event.data && typeof event.data === 'object' && !Array.isArray(event.data) ? event.data as Record<string, unknown> : {}
    const winnerID = idFromUnknown(data.winner_user_id) || idFromUnknown(data.user_id) || getCurrentViewerID()
    const auctionID = idFromUnknown(data.auction_id) || auction?.id || pendingBid.auctionID
    if (!auctionID || !isSameEntityID(auctionID, pendingBid.auctionID)) {
      return
    }

    pendingBidRef.current = null
    localLeadingBidRef.current = {
      auctionID,
      winnerID,
    }
  }

  function updateLocalLeadingBidFromAuctionRecord(record: UserLiveAuctionRecord) {
    const localLeadingBid = localLeadingBidRef.current
    if (!localLeadingBid || !isSameEntityID(record.id, localLeadingBid.auctionID)) {
      return
    }

    const winnerID = record.winner_user_id
    if (winnerID && !isSameEntityID(winnerID, localLeadingBid.winnerID, getCurrentViewerID())) {
      localLeadingBidRef.current = null
    }
  }

  function getCurrentViewerID() {
    return session.entry?.viewer?.user_id || activeEntry?.viewer?.user_id || userID || profile?.id
  }

  function openDealDialog(
    record: UserLiveAuctionRecord,
    auction: UserLiveAuction | null,
    goods?: UserLiveGoods | null,
  ) {
    const winnerID = record.winner_user_id || auction?.winner_user_id
    const price = record.deal_price || record.current_price || auction?.deal_price || auction?.current_price || 0
    const viewerUserID = getCurrentViewerID()
    const localLeadingBid = localLeadingBidRef.current
    const isLocalLeadingWinner =
      Boolean(localLeadingBid && isSameEntityID(record.id, localLeadingBid.auctionID)) &&
      (!winnerID || isSameEntityID(winnerID, localLeadingBid?.winnerID, viewerUserID))
    const isWinner = isSameEntityID(winnerID, viewerUserID, userID, profile?.id) || isLocalLeadingWinner
    const winnerProfile = winnerID ? publicUserCacheRef.current.get(winnerID) : undefined
    const dealGoods = record.goods || goods || null
    setDealDialog({
      auctionID: record.id,
      role: isWinner ? 'winner' : 'viewer',
      goods: dealGoods,
      title: dealGoods?.title || '竞拍商品',
      price,
      winnerID,
      winnerName: winnerProfile?.nickname || record.winner_display_name || auction?.winner_display_name,
      winnerAvatar: winnerProfile?.avatar,
      rounds: Math.max(record.bid_count || auction?.bid_count || 1, 1),
      purchaseExpireAt: Date.now() + 30 * 60 * 1000,
    })
    if (!isWinner && winnerID && !winnerProfile) {
      void hydrateDealWinnerProfile(record.id, winnerID)
    }
    if (!dealGoods?.cover_url && record.goods_id) {
      void hydrateDealGoods(record.id, record.goods_id)
    }
  }

  async function hydrateDealWinnerProfile(auctionID: EntityID, winnerID: EntityID) {
    try {
      const winnerProfile = await getUserProfile(winnerID)
      publicUserCacheRef.current.set(winnerID, winnerProfile)
      setDealDialog((current) => {
        if (!current || current.auctionID !== auctionID || current.winnerID !== winnerID || current.role !== 'viewer') {
          return current
        }

        return {
          ...current,
          winnerName: winnerProfile.nickname || current.winnerName,
          winnerAvatar: winnerProfile.avatar || current.winnerAvatar,
        }
      })
    } catch {
      // 成交弹窗可以继续展示脱敏兜底信息。
    }
  }

  async function hydrateDealGoods(auctionID: EntityID, goodsID: EntityID) {
    try {
      const goods = await getGoods(goodsID)
      if (!goods) {
        return
      }

      setDealDialog((current) => {
        if (!current || current.auctionID !== auctionID) {
          return current
        }

        return {
          ...current,
          goods: current.goods ? { ...goods, ...current.goods, cover_url: current.goods.cover_url || goods.cover_url } : goods,
          title: current.title === '竞拍商品' ? goods.title : current.title,
        }
      })
    } catch {
      // The deal dialog can still show the title and price if the goods image cannot be refreshed.
    }
  }

  function handleSlideChange(swiper: SwiperInstance) {
    if (swiper.activeIndex !== activeIndex) {
      const nextRoomID = rooms[swiper.activeIndex]?.room.id
      if (restoringRoomRef.current && restoringRoomRef.current === nextRoomID) {
        restoringRoomRef.current = null
      } else {
        leaveRoom()
      }
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

  function openGoodsDetail(goodsID: EntityID, context?: { startPrice?: number; returnRoomID?: EntityID }) {
    if (context?.returnRoomID) {
      storeReturnRoomID(context.returnRoomID)
    }
    navigate(`/goods/${goodsID}`, { state: context })
  }

  useEffect(() => {
    if (!returnRoomID || rooms.length === 0) {
      return
    }

    const roomIndex = rooms.findIndex((item) => item.room.id === returnRoomID)
    if (roomIndex < 0) {
      if (feedQuery.hasNextPage && !feedQuery.isFetchingNextPage) {
        void feedQuery.fetchNextPage().catch(() => undefined)
      }
      return
    }

    clearStoredReturnRoomID()
    navigate('/', { replace: true, state: null })
    restoringRoomRef.current = returnRoomID
    if (roomIndex !== activeIndex) {
      swiperRef.current?.slideTo(roomIndex, 0)
      setActiveIndex(roomIndex)
    } else {
      restoringRoomRef.current = null
    }

    if (session.roomID !== returnRoomID || session.status !== 'entered') {
      void handleEnterRoom(returnRoomID)
    }
  }, [activeIndex, feedQuery, navigate, returnRoomID, rooms, session.roomID, session.status])

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
      <Swiper
        className="live-swiper"
        direction="vertical"
        modules={[Keyboard, Mousewheel]}
        keyboard
        mousewheel
        onSwiper={(swiper) => {
          swiperRef.current = swiper
        }}
        onSlideChange={handleSlideChange}
      >
        {rooms.map((item) => {
          const isActive = item.room.id === activeRoom?.room.id
          const itemEntry = isActive ? activeEntry : null
          const displayItem = isActive && activeDisplayItem ? activeDisplayItem : item
          const auction = itemEntry?.current_auction || itemEntry?.auction || null
          const goods = itemEntry?.goods || null
          const isAuctionCardVisible = Boolean(auction && dismissedAuctionCardByRoom[item.room.id] !== auction.id)

          return (
            <SwiperSlide key={item.room.id}>
              <article className={cx('live-slide', isActive && isEntered && 'is-entered')}>
                <LiveVideo item={displayItem} stream={itemEntry?.stream} enabled={isActive} />

                {!isEntered ? (
                  <>
                    <SearchOverlay />
                    <LiveRoomMeta item={displayItem} entry={itemEntry} variant="preview" />
                    <div className="live-preview-action">
                      <button
                        type="button"
                        disabled={session.status === 'entering' && session.roomID === item.room.id}
                        onClick={() => void handleEnterRoom()}
                      >
                        {session.status === 'entering' && session.roomID === item.room.id ? (
                          <>
                            <DotLoading />
                            进入中
                          </>
                        ) : (
                          <>
                            <span className="live-preview-action__pulse" aria-hidden="true">
                              <i />
                              <i />
                              <i />
                            </span>
                            点击进入直播间
                          </>
                        )}
                      </button>
                      {session.status === 'error' && session.roomID === item.room.id && (
                        <p>{session.error}</p>
                      )}
                    </div>
                  </>
                ) : (
                  <div className="live-room-entered">
                    <div className="live-room-entered__head">
                      <LiveRoomMeta item={displayItem} entry={itemEntry} variant="entered" />
                    </div>

                    <div className="live-room-entered__body">
                      <div className={cx('live-room-entered__auction-row', !isAuctionCardVisible && 'is-expanded')}>
                        <LiveMessageList messages={session.messages} variant="dock" />

                        {isAuctionCardVisible ? (
                          <AuctionPanel
                            auction={auction}
                            goods={goods}
                            onOpenGoods={() => {
                              if (goods?.id) {
                                openGoodsDetail(goods.id, {
                                  startPrice: auction?.start_price,
                                  returnRoomID: item.room.id,
                                })
                              }
                            }}
                            onOpenRecords={() => openAuctionRecords()}
                            onClose={() => {
                              if (auction) {
                                setDismissedAuctionCardByRoom((current) => ({
                                  ...current,
                                  [item.room.id]: auction.id,
                                }))
                              }
                            }}
                            variant="dock"
                          />
                        ) : null}
                      </div>

                      <div className="live-chat-row">
                        <div className="live-chat-composer">
                          <textarea
                            className="live-chat-composer__textarea"
                            value={chatDraft}
                            rows={1}
                            placeholder="说点什么..."
                            onChange={(event) => setChatDraft(event.target.value)}
                            onKeyDown={(event) => {
                              if (event.key === 'Enter' && !event.shiftKey) {
                                event.preventDefault()
                                handleSendChat()
                              }
                            }}
                          />
                          <span className="live-chat-composer__placeholder" aria-hidden="true">
                            <Smile size={18} strokeWidth={2.2} aria-hidden="true" />
                          </span>
                          <button
                            type="button"
                            className="live-chat-composer__send"
                            disabled={isSendingDanmaku || chatDraft.trim().length === 0}
                            onClick={handleSendChat}
                            aria-label="发送弹幕"
                          >
                            {isSendingDanmaku ? <DotLoading /> : <SendHorizontal size={16} aria-hidden="true" />}
                          </button>
                        </div>

                        <div className="live-chat-actions">
                          <button
                            type="button"
                            className="live-chat-composer__auction"
                            onClick={() => openAuctionRecords()}
                            aria-label="查看竞拍记录"
                          >
                            <Gavel size={16} aria-hidden="true" />
                          </button>
                        </div>
                      </div>
                    </div>
                  </div>
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

      <AuctionRecordsDrawer
        visible={isAuctionRecordsOpen}
        roomTitle={activeEntry?.room.title || activeRoom?.room.title || '直播竞拍间'}
        currentAuction={activeAuction}
        runtime={activeRuntime}
        records={isEntered ? mergeCurrentGoodsIntoAuctionRecords(session.auctionRecords, activeAuction, activeEntry?.goods) : []}
        isBidding={isBidding}
        onBid={handleBid}
        onOpenGoods={(goodsID, context) =>
          openGoodsDetail(goodsID, {
            startPrice: context?.startPrice,
            returnRoomID: activeRoom?.room.id,
          })}
        onClose={() => setIsAuctionRecordsOpen(false)}
      />

      {dealDialog ? (
        <div className="auction-deal-dialog" role="dialog" aria-modal="true" aria-label="竞拍成交结果">
          <div className="auction-deal-dialog__backdrop" />
          <section className={`auction-deal-dialog__panel auction-deal-dialog__panel--${dealDialog.role}`}>
            <button
              type="button"
              className="auction-deal-dialog__close"
              aria-label="关闭成交结果"
              onClick={() => setDealDialog(null)}
            >
              ×
            </button>
            {dealDialog.role === 'winner' ? (
              <h2>恭喜竞拍成功</h2>
            ) : (
              <h2 className="auction-deal-dialog__viewer-title">
                <span>落槌定音</span>
                <span>恭喜成交！！</span>
              </h2>
            )}

            <div className="auction-deal-dialog__goods">
              <span className="auction-deal-dialog__cover">
                {dealDialog.goods?.cover_url ? <img src={dealDialog.goods.cover_url} alt="" /> : null}
              </span>
              <div>
                <h3>{dealDialog.title}</h3>
                <span className="auction-deal-dialog__price-label">成交价</span>
                <strong>{formatCentAmount(dealDialog.price)}</strong>
                {dealDialog.role === 'viewer' ? (
                  <span className="auction-deal-dialog__winner">
                    <span className="auction-deal-dialog__avatar" aria-hidden="true">
                      {dealDialog.winnerAvatar ? <img src={dealDialog.winnerAvatar} alt="" /> : getWinnerInitial(dealDialog.winnerName)}
                    </span>
                    <b>{displayWinnerName(dealDialog.winnerName)}</b>
                  </span>
                ) : null}
              </div>
            </div>

            {dealDialog.role === 'winner' ? (
              <>
                <button type="button" className="auction-deal-dialog__pay">
                  确认地址并支付
                </button>
                <div className="auction-deal-dialog__countdown">
                  <span>距购买失效</span>
                  <b>{formatPurchaseCountdown(Math.max(0, dealDialog.purchaseExpireAt - dealNow))}</b>
                </div>
              </>
            ) : (
              <p className="auction-deal-dialog__viewer-summary">经过{dealDialog.rounds}轮竞拍成交</p>
            )}
          </section>
        </div>
      ) : null}
    </section>
  )

  function openAuctionRecords() {
    setIsAuctionRecordsOpen(true)
    if (activeRoom?.room.id) {
      void refreshAuctionRecords(activeRoom.room.id)
    }
  }
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

function userDisplayName(profile: UserProfile) {
  return profile.nickname?.trim() || profile.username?.trim() || ''
}

function displayWinnerName(value?: string) {
  const name = value?.trim()
  if (!name) {
    return '用户***'
  }

  if (/^用户\d+$/.test(name)) {
    return '用户***'
  }

  return `${name.slice(0, 1)}***`
}

function getWinnerInitial(value?: string) {
  return value?.trim().slice(0, 1) || '成'
}

function formatPurchaseCountdown(milliseconds: number) {
  const totalSeconds = Math.max(0, Math.ceil(milliseconds / 1000))
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60

  return [hours, minutes, seconds].map((part) => String(part).padStart(2, '0')).join(':')
}

function isSameEntityID(target?: EntityID | null, ...candidates: Array<EntityID | null | undefined>) {
  const normalizedTarget = normalizeEntityID(target)
  if (!normalizedTarget) {
    return false
  }

  return candidates.some((candidate) => normalizeEntityID(candidate) === normalizedTarget)
}

function normalizeEntityID(value?: EntityID | null) {
  return value === undefined || value === null ? '' : String(value).trim()
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

function mergeMessages(current: BidEventMessage[], incoming: BidEventMessage[]) {
  const byID = new Map<string, BidEventMessage>()
  for (const message of [...current, ...incoming]) {
    byID.set(message.id, message)
  }

  return [...byID.values()].sort((left, right) => left.createdAt - right.createdAt)
}

function isLiveRoom(status?: string, statusText?: string) {
  const text = `${status || ''} ${statusText || ''}`.toLowerCase()
  return text.includes('living') || text.includes('直播中') || text.includes('正在直播')
}

function isAuctionActive(status?: number) {
  return status === 1
}

function isAuctionRecordEvent(type?: string) {
  return (
    type === 'auction_started' ||
    type === 'bid_accepted' ||
    type === 'auction_finished' ||
    type === 'auction_deal' ||
    type === 'auction_failed' ||
    type === 'auction_cancelled' ||
    type === 'auction_canceled'
  )
}

function isDealRecordEvent(type?: string) {
  return type === 'auction_finished' || type === 'auction_deal'
}

function buildDealRecordFromEvent(
  event: { data?: unknown; auction_id?: EntityID; room_id?: EntityID; server_time?: number; version?: number },
  auction: UserLiveAuction | null,
): UserLiveAuctionRecord | null {
  const data = event.data && typeof event.data === 'object' && !Array.isArray(event.data) ? event.data as Record<string, unknown> : {}
  const auctionID = idFromUnknown(data.auction_id) || idFromUnknown(event.auction_id) || auction?.id
  if (!auctionID) {
    return null
  }

  const currentPrice = numberFromUnknown(data.current_price) || numberFromUnknown(data.deal_price) || auction?.current_price || 0
  const status = numberFromUnknown(data.status) || 2
  const winnerUserID = idFromUnknown(data.winner_user_id) || idFromUnknown(data.user_id) || auction?.winner_user_id
  const winnerDisplayName = displayNameFromData(data) || auction?.winner_display_name

  return {
    id: auctionID,
    room_id: idFromUnknown(data.room_id) || idFromUnknown(event.room_id) || auction?.room_id,
    goods_id: idFromUnknown(data.goods_id) || auction?.goods_id || '',
    shop_id: idFromUnknown(data.shop_id) || auction?.shop_id,
    status,
    status_text: '已成交',
    start_price: numberFromUnknown(data.start_price) || auction?.start_price || currentPrice,
    bid_increment: numberFromUnknown(data.bid_increment) || auction?.bid_increment || 0,
    current_price: currentPrice,
    deal_price: numberFromUnknown(data.deal_price) || currentPrice,
    bid_count: numberFromUnknown(data.bid_count) || auction?.bid_count || 1,
    start_time: timeFromUnknown(data.start_time) || auction?.start_time,
    end_time: timeFromUnknown(data.end_time),
    server_time: numberFromUnknown(data.server_time) || event.server_time,
    version: numberFromUnknown(data.version) || event.version,
    winner_user_id: winnerUserID,
    winner_display_name: winnerDisplayName,
  }
}

function displayNameFromData(data: Record<string, unknown>) {
  return (
    stringFromUnknown(data.winner_display_name) ||
    stringFromUnknown(data.bidder_display_name) ||
    stringFromUnknown(data.display_name) ||
    stringFromUnknown(data.nickname) ||
    stringFromUnknown(data.username) ||
    stringFromUnknown(data.user_name) ||
    undefined
  )
}

function idFromUnknown(value: unknown) {
  if (typeof value === 'string') {
    const trimmed = value.trim()
    return trimmed && trimmed !== '0' ? trimmed : undefined
  }
  if (typeof value === 'number' && Number.isFinite(value) && value > 0) {
    return String(value)
  }

  return undefined
}

function numberFromUnknown(value: unknown) {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value
  }
  if (typeof value === 'string') {
    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : undefined
  }

  return undefined
}

function stringFromUnknown(value: unknown) {
  return typeof value === 'string' ? value.trim() : ''
}

function timeFromUnknown(value: unknown) {
  return typeof value === 'string' || typeof value === 'number' ? value : undefined
}

function buildAuctionRecordsFromEntry(entry: UserLiveEntry): UserLiveAuctionRecord[] {
  const currentAuction = entry.current_auction || entry.auction || null
  if (!currentAuction) {
    return []
  }

  return [
    {
      id: currentAuction.id,
      room_id: currentAuction.room_id,
      goods_id: currentAuction.goods_id,
      shop_id: currentAuction.shop_id,
      goods: entry.goods || null,
      status: currentAuction.status,
      status_text: currentAuction.status_text,
      start_price: currentAuction.start_price,
      bid_increment: currentAuction.bid_increment,
      current_price: currentAuction.current_price,
      deal_price: currentAuction.deal_price,
      bid_count: currentAuction.bid_count,
      start_time: currentAuction.start_time,
      end_time: currentAuction.end_time,
      winner_user_id: currentAuction.winner_user_id,
      winner_display_name: currentAuction.winner_display_name,
      created_at: currentAuction.start_time,
      updated_at: currentAuction.end_time,
    },
  ]
}

function upsertAuctionRecord(records: UserLiveAuctionRecord[], next: UserLiveAuctionRecord) {
  const index = records.findIndex((item) => item.id === next.id)
  if (index === -1) {
    return [next, ...records].slice(0, 20)
  }

  const nextRecords = [...records]
  nextRecords[index] = {
    ...nextRecords[index],
    ...next,
    goods: next.goods || nextRecords[index].goods,
  }
  return nextRecords
}

function mergeCurrentGoodsIntoAuctionRecords(
  records: UserLiveAuctionRecord[],
  currentAuction?: UserLiveEntry['auction'] | null,
  goods?: UserLiveEntry['goods'] | null,
) {
  if (!currentAuction || !goods) {
    return records
  }

  return records.map((record) => (record.id === currentAuction.id ? { ...record, goods } : record))
}

async function fetchMissingAuctionGoods(records: UserLiveAuctionRecord[]) {
  const ids = records.filter((record) => !record.goods?.cover_url).map((record) => record.goods_id)
  if (ids.length === 0) {
    return new Map<EntityID, UserLiveGoods>()
  }

  try {
    const goodsList = await batchGetGoods(ids)
    return new Map(goodsList.map((goods) => [goods.id, goods]))
  } catch {
    return new Map<EntityID, UserLiveGoods>()
  }
}

function mergeAuctionRecordsGoods(records: UserLiveAuctionRecord[], goodsByID: Map<EntityID, UserLiveGoods>) {
  if (goodsByID.size === 0) {
    return records
  }

  return records.map((record) => {
    const goods = goodsByID.get(record.goods_id)
    if (!goods) {
      return record
    }

    return {
      ...record,
      goods: record.goods ? { ...goods, ...record.goods, cover_url: record.goods.cover_url || goods.cover_url } : goods,
    }
  })
}

async function fetchMissingAuctionWinnerNames(records: UserLiveAuctionRecord[], userCache: Map<EntityID, UserProfile>) {
  const ids = Array.from(
    new Set(
      records
        .filter((record) => record.winner_user_id && shouldHydrateDisplayName(record.winner_display_name))
        .map((record) => record.winner_user_id as EntityID),
    ),
  )
  if (ids.length === 0) {
    return new Map<EntityID, string>()
  }

  const entries = await Promise.all(
    ids.map(async (id) => {
      const cachedProfile = userCache.get(id)
      if (cachedProfile) {
        return [id, userDisplayName(cachedProfile)] as const
      }

      try {
        const profile = await getUserProfile(id)
        userCache.set(id, profile)
        return [id, userDisplayName(profile)] as const
      } catch {
        return [id, ''] as const
      }
    }),
  )

  return new Map(entries.filter((entry) => Boolean(entry[1])))
}

function mergeAuctionRecordsWinners(records: UserLiveAuctionRecord[], winnerNamesByID: Map<EntityID, string>) {
  if (winnerNamesByID.size === 0) {
    return records
  }

  return records.map((record) => {
    const winnerName = record.winner_user_id ? winnerNamesByID.get(record.winner_user_id) : ''
    if (!winnerName) {
      return record
    }

    return {
      ...record,
      winner_display_name: winnerName,
    }
  })
}

function shouldHydrateDisplayName(value?: string) {
  const name = value?.trim()
  return !name || /^用户\d+$/.test(name)
}

function getStoredReturnRoomID() {
  try {
    return window.sessionStorage.getItem(returnRoomStorageKey) || undefined
  } catch {
    return undefined
  }
}

function storeReturnRoomID(roomID: EntityID) {
  try {
    window.sessionStorage.setItem(returnRoomStorageKey, roomID)
  } catch {
    // Ignore storage failures; route state still handles normal in-app back.
  }
}

function clearStoredReturnRoomID() {
  try {
    window.sessionStorage.removeItem(returnRoomStorageKey)
  } catch {
    // Ignore storage failures.
  }
}
