import { Button, DotLoading } from 'antd-mobile'
import { ChevronLeft, ShoppingBag } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useLocation, useNavigate, useParams } from 'react-router-dom'
import { formatCentAmount, getErrorText } from '../../lib/format'
import { getGoods } from '../../services/liveApi'
import type { UserLiveGoods } from '../../types/domain'

type GoodsDetailState = {
  startPrice?: number
  returnRoomID?: string
}

export function GoodsDetailPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const params = useParams<{ goodsId: string }>()
  const [goods, setGoods] = useState<UserLiveGoods | null>(null)
  const [status, setStatus] = useState<'loading' | 'ready' | 'error'>('loading')
  const [error, setError] = useState('')
  const detailState = location.state as GoodsDetailState | null
  const startPrice = typeof detailState?.startPrice === 'number' ? detailState.startPrice : undefined
  const returnRoomID = detailState?.returnRoomID

  function handleBack() {
    if (returnRoomID) {
      navigate('/', { replace: true, state: { returnRoomID } })
      return
    }

    navigate(-1)
  }

  useEffect(() => {
    const goodsId = params.goodsId?.trim()
    if (!goodsId) {
      setStatus('error')
      setError('商品不存在')
      return
    }

    const controller = new AbortController()
    setStatus('loading')
    setError('')

    void getGoods(goodsId, controller.signal)
      .then((data) => {
        setGoods(data)
        setStatus('ready')
      })
      .catch((err) => {
        setGoods(null)
        setStatus('error')
        setError(getErrorText(err, '商品详情加载失败'))
      })

    return () => controller.abort()
  }, [params.goodsId])

  return (
    <section className="goods-detail-page">
      <header className="goods-detail-page__header">
        <button type="button" aria-label="返回" onClick={handleBack}>
          <ChevronLeft size={20} aria-hidden="true" />
        </button>
        <div>
          <h1>商品详情</h1>
          <p>竞拍商品信息</p>
        </div>
      </header>

      {status === 'loading' ? (
        <div className="goods-detail-page__state">
          <DotLoading />
          <p>正在加载商品</p>
        </div>
      ) : status === 'error' ? (
        <div className="goods-detail-page__state">
          <h2>加载失败</h2>
          <p>{error}</p>
          <Button color="danger" size="small" onClick={handleBack}>
            返回
          </Button>
        </div>
      ) : goods ? (
        <article className="goods-detail-page__card">
          <div className="goods-detail-page__cover">
            {goods.cover_url ? <img src={goods.cover_url} alt="" /> : <ShoppingBag size={36} aria-hidden="true" />}
          </div>
          <h2>{goods.title}</h2>
          {startPrice !== undefined ? (
            <div className="goods-detail-page__start-price">
              <span>起拍价</span>
              <strong>{formatCentAmount(startPrice)}</strong>
            </div>
          ) : null}
          {goods.description ? <p className="goods-detail-page__description">{goods.description}</p> : null}
        </article>
      ) : null}
    </section>
  )
}
