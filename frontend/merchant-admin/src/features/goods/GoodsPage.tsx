import type React from 'react'
import { useEffect, useState } from 'react'
import { StateBlock } from '../../components/common/StateBlock'
import { StatusTag } from '../../components/common/StatusTag'
import {
  createGoods,
  deleteGoods,
  listShopGoods,
  putGoodsOffSale,
  putGoodsOnSale,
  updateGoods,
  uploadGoodsCover,
} from '../../services/goodsApi'
import type { Goods, GoodsDraft, PageResult } from '../../types/domain'
import { getErrorText } from '../../lib/format'
import { goodsStatusLabel, goodsStatusTone } from '../../lib/status'

const emptyGoodsDraft: GoodsDraft = {
  title: '',
  cover_url: '',
  description: '',
}

type GoodsPageProps = {
  token: string
}

export function GoodsPage({ token }: GoodsPageProps) {
  const [result, setResult] = useState<PageResult<Goods>>({
    total: 0,
    page: 1,
    page_size: 10,
    list: [],
  })
  const [keyword, setKeyword] = useState('')
  const [draft, setDraft] = useState<GoodsDraft>(emptyGoodsDraft)
  const [editingGoods, setEditingGoods] = useState<Goods | null>(null)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [isLoading, setIsLoading] = useState(true)
  const [isSubmitting, setIsSubmitting] = useState(false)

  function loadGoods(nextPage = result.page) {
    setIsLoading(true)
    setError('')
    listShopGoods(token, { page: nextPage, pageSize: result.page_size, keyword })
      .then(setResult)
      .catch((err) => setError(getErrorText(err, '商品列表加载失败')))
      .finally(() => setIsLoading(false))
  }

  useEffect(() => {
    let isActive = true
    listShopGoods(token, { page: 1, pageSize: result.page_size, keyword: '' })
      .then((resp) => {
        if (isActive) {
          setResult(resp)
        }
      })
      .catch((err) => {
        if (isActive) {
          setError(getErrorText(err, '商品列表加载失败'))
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

  function startEdit(goods: Goods) {
    setEditingGoods(goods)
    setDraft({
      title: goods.title,
      cover_url: goods.cover_url || '',
      description: goods.description || '',
    })
    setNotice('')
    setError('')
  }

  function resetForm() {
    setEditingGoods(null)
    setDraft(emptyGoodsDraft)
  }

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const title = draft.title.trim()
    if (!title) {
      setError('商品标题不能为空')
      return
    }

    setIsSubmitting(true)
    setError('')
    setNotice('')
    try {
      if (editingGoods) {
        await updateGoods(token, editingGoods.id, { ...draft, title })
        setNotice('商品已更新')
      } else {
        await createGoods(token, { ...draft, title })
        setNotice('商品已创建')
      }
      resetForm()
      loadGoods(1)
    } catch (err) {
      setError(getErrorText(err, '保存商品失败'))
    } finally {
      setIsSubmitting(false)
    }
  }

  async function handleUpload(file: File | null) {
    if (!file) {
      return
    }

    setIsSubmitting(true)
    setError('')
    try {
      const coverURL = await uploadGoodsCover(token, file)
      setDraft((current) => ({ ...current, cover_url: coverURL }))
      setNotice('封面已上传')
    } catch (err) {
      setError(getErrorText(err, '封面上传失败'))
    } finally {
      setIsSubmitting(false)
    }
  }

  async function runGoodsAction(action: () => Promise<void>, successMessage: string) {
    setError('')
    setNotice('')
    try {
      await action()
      setNotice(successMessage)
      loadGoods(result.page)
    } catch (err) {
      setError(getErrorText(err, '商品操作失败'))
    }
  }

  return (
    <>
      <div className="page-header">
        <div>
          <h1>商品管理</h1>
          <p>维护商铺商品基础资料，商品创建后默认下架，封面由网关上传接口返回 URL。</p>
        </div>
        <button type="button" className="primary-inline" onClick={resetForm}>
          新建商品
        </button>
      </div>

      <div className="content-grid goods-layout">
        <section className="panel">
          <div className="panel-header">
            <h2>商品列表</h2>
            <span className="muted">共 {result.total} 条</span>
          </div>

          <form
            className="toolbar"
            onSubmit={(event) => {
              event.preventDefault()
              loadGoods(1)
            }}
          >
            <input
              className="text-input"
              value={keyword}
              placeholder="商品关键词"
              onChange={(event) => setKeyword(event.target.value)}
            />
            <button type="submit" className="ghost-button">
              搜索
            </button>
            <button type="button" className="ghost-button" onClick={() => loadGoods(result.page)}>
              刷新
            </button>
          </form>

          {error && <div className="notice danger">{error}</div>}
          {notice && <div className="notice success">{notice}</div>}

          <div className="table-wrap" aria-busy={isLoading}>
            <table className="data-table">
              <thead>
                <tr>
                  <th>商品</th>
                  <th>状态</th>
                  <th>更新时间</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                {result.list.map((goods) => (
                  <tr key={goods.id}>
                    <td>
                      <div className="goods-cell">
                        {goods.cover_url ? (
                          <img src={goods.cover_url} alt="" />
                        ) : (
                          <span className="thumb-placeholder">图</span>
                        )}
                        <div>
                          <strong>{goods.title}</strong>
                          <span>#{goods.id}</span>
                        </div>
                      </div>
                    </td>
                    <td>
                      <StatusTag
                        label={goodsStatusLabel(goods.status)}
                        tone={goodsStatusTone(goods.status)}
                      />
                    </td>
                    <td>{goods.updated_at || '-'}</td>
                    <td>
                      <div className="row-actions">
                        <button type="button" onClick={() => startEdit(goods)}>
                          编辑
                        </button>
                        <button
                          type="button"
                          onClick={() =>
                            runGoodsAction(() => putGoodsOnSale(token, goods.id), '商品已上架')
                          }
                        >
                          上架
                        </button>
                        <button
                          type="button"
                          onClick={() =>
                            runGoodsAction(() => putGoodsOffSale(token, goods.id), '商品已下架')
                          }
                        >
                          下架
                        </button>
                        <button
                          type="button"
                          className="danger"
                          onClick={() => {
                            if (confirm('删除后商品将从默认列表隐藏，确认删除吗？')) {
                              runGoodsAction(() => deleteGoods(token, goods.id), '商品已删除')
                            }
                          }}
                        >
                          删除
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {!isLoading && result.list.length === 0 && (
              <StateBlock title="暂无商品" description="创建商品后，可继续配置竞拍活动。" />
            )}
          </div>
        </section>

        <section className="panel side-editor">
          <div className="panel-header">
            <h2>{editingGoods ? '编辑商品' : '新建商品'}</h2>
            {editingGoods && (
              <button type="button" className="ghost-button" onClick={resetForm}>
                取消编辑
              </button>
            )}
          </div>

          <form className="stack-form" onSubmit={handleSubmit}>
            <label className="field">
              <span className="field-label">商品标题 *</span>
              <input
                className="text-input"
                value={draft.title}
                maxLength={255}
                placeholder="例如：高冰翡翠手镯"
                onChange={(event) =>
                  setDraft((current) => ({ ...current, title: event.target.value }))
                }
              />
            </label>
            <label className="field">
              <span className="field-label">封面</span>
              <input
                className="text-input"
                value={draft.cover_url}
                placeholder="封面 URL"
                onChange={(event) =>
                  setDraft((current) => ({ ...current, cover_url: event.target.value }))
                }
              />
            </label>
            <label className="file-picker wide">
              <input
                type="file"
                accept="image/png,image/jpeg,image/webp"
                onChange={(event) => handleUpload(event.target.files?.[0] || null)}
              />
              <span>上传封面</span>
            </label>
            <label className="field">
              <span className="field-label">商品描述</span>
              <textarea
                value={draft.description}
                placeholder="商品材质、品相、直播说明"
                onChange={(event) =>
                  setDraft((current) => ({ ...current, description: event.target.value }))
                }
              />
            </label>
            <button type="submit" className="primary-action" disabled={isSubmitting}>
              {isSubmitting ? '保存中...' : '保存商品'}
            </button>
          </form>
        </section>
      </div>
    </>
  )
}
