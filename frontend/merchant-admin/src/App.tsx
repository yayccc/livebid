import type React from 'react'
import { useEffect, useState } from 'react'
import './App.css'
import { AdminShell, type AdminPageKey } from './components/common/AdminShell'
import { AuctionPage } from './features/auction/AuctionPage'
import { DashboardPage } from './features/dashboard/DashboardPage'
import { GoodsPage } from './features/goods/GoodsPage'
import { LivePage } from './features/live/LivePage'
import {
  getCurrentShop,
  loginShop,
  registerShop,
  uploadShopLogo,
  type Shop,
} from './services/shopApi'
import { getErrorText } from './lib/format'

const tokenStorageKey = 'livebid_merchant_access_token'

type AuthMode = 'login' | 'register'
type FormStatus = 'idle' | 'submitting'

type LoginFormValues = {
  username: string
  password: string
}

type RegisterFormValues = {
  username: string
  password: string
  shopName: string
  logo: string
  description: string
  phone: string
  email: string
}

function App() {
  const [token, setToken] = useState(() => localStorage.getItem(tokenStorageKey))
  const [shop, setShop] = useState<Shop | null>(null)
  const [isLoadingShop, setIsLoadingShop] = useState(Boolean(token))
  const [authNotice, setAuthNotice] = useState('')
  const [activePage, setActivePage] = useState<AdminPageKey>('dashboard')

  useEffect(() => {
    if (!token) {
      return
    }

    let isActive = true

    getCurrentShop(token)
      .then((nextShop) => {
        if (isActive) {
          setShop(nextShop)
        }
      })
      .catch(() => {
        if (isActive) {
          clearSession()
          setAuthNotice('登录状态已失效，请重新登录')
        }
      })
      .finally(() => {
        if (isActive) {
          setIsLoadingShop(false)
        }
      })

    return () => {
      isActive = false
    }
  }, [token])

  function handleLogin(nextToken: string, nextShop: Shop | null) {
    localStorage.setItem(tokenStorageKey, nextToken)
    setAuthNotice('')
    setShop(nextShop)
    setIsLoadingShop(!nextShop)
    setToken(nextToken)
    setActivePage('dashboard')
  }

  function clearSession() {
    localStorage.removeItem(tokenStorageKey)
    setToken(null)
    setShop(null)
    setIsLoadingShop(false)
  }

  function handleRefreshShop() {
    if (!token) {
      return
    }
    setIsLoadingShop(true)
    getCurrentShop(token)
      .then(setShop)
      .catch(() => {
        clearSession()
        setAuthNotice('登录状态已失效，请重新登录')
      })
      .finally(() => setIsLoadingShop(false))
  }

  if (!token) {
    return <AuthPage initialNotice={authNotice} onLogin={handleLogin} />
  }

  return (
    <AdminShell
      shop={shop}
      activePage={activePage}
      onNavigate={setActivePage}
      onRefreshShop={handleRefreshShop}
      onLogout={clearSession}
    >
      {isLoadingShop && <div className="notice success">正在读取商铺信息...</div>}
      {activePage === 'dashboard' && (
        <DashboardPage
          token={token}
          shopName={shop?.shopName || ''}
          onOpenGoods={() => setActivePage('goods')}
          onOpenAuctions={() => setActivePage('auctions')}
        />
      )}
      {activePage === 'goods' && <GoodsPage token={token} />}
      {activePage === 'auctions' && <AuctionPage token={token} />}
      {activePage === 'live' && <LivePage />}
      {activePage === 'shop' && (
        <ShopPage shop={shop} isLoadingShop={isLoadingShop} onRefreshShop={handleRefreshShop} />
      )}
    </AdminShell>
  )
}

function AuthPage({
  initialNotice,
  onLogin,
}: {
  initialNotice: string
  onLogin: (token: string, shop: Shop | null) => void
}) {
  const [mode, setMode] = useState<AuthMode>('login')
  const [notice, setNotice] = useState(initialNotice)
  const [loginValues, setLoginValues] = useState<LoginFormValues>({ username: '', password: '' })
  const [registerValues, setRegisterValues] = useState<RegisterFormValues>({
    username: '',
    password: '',
    shopName: '',
    logo: '',
    description: '',
    phone: '',
    email: '',
  })
  const [logoFile, setLogoFile] = useState<File | null>(null)
  const [error, setError] = useState('')
  const [status, setStatus] = useState<FormStatus>('idle')

  function switchMode(nextMode: AuthMode) {
    setMode(nextMode)
    setError('')
    setNotice('')
  }

  async function handleLoginSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const username = loginValues.username.trim()
    const password = loginValues.password

    if (!username || !password) {
      setError('账号和密码为必填项')
      return
    }

    setStatus('submitting')
    setError('')

    try {
      const nextToken = await loginShop({ username, password })
      let nextShop: Shop | null = null
      try {
        nextShop = await getCurrentShop(nextToken)
      } catch {
        nextShop = null
      }
      onLogin(nextToken, nextShop)
    } catch (err) {
      setError(getErrorText(err, '登录失败，请检查账号或密码'))
    } finally {
      setStatus('idle')
    }
  }

  async function handleRegisterSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const username = registerValues.username.trim()
    const password = registerValues.password
    const shopName = registerValues.shopName.trim()

    if (!username || !password || !shopName) {
      setError('账号、密码和商铺名称为必填项')
      return
    }

    setStatus('submitting')
    setError('')

    try {
      const uploadedLogo = logoFile ? await uploadShopLogo(logoFile) : ''
      await registerShop({
        username,
        password,
        shopName,
        logo: uploadedLogo || registerValues.logo.trim(),
        description: registerValues.description.trim(),
        phone: registerValues.phone.trim(),
        email: registerValues.email.trim(),
      })

      setLoginValues({ username, password: '' })
      setRegisterValues({
        username: '',
        password: '',
        shopName: '',
        logo: '',
        description: '',
        phone: '',
        email: '',
      })
      setLogoFile(null)
      setMode('login')
      setNotice('商铺已创建，请使用账号登录')
    } catch (err) {
      setError(getErrorText(err, '注册失败，请稍后重试'))
    } finally {
      setStatus('idle')
    }
  }

  return (
    <main className="auth-page">
      <section className="auth-shell" aria-label="商家后台登录注册">
        <div className="auth-intro">
          <p className="brand">
            <span>LiveBid</span>
            商家后台
          </p>
          <h1>直播竞拍经营工作台</h1>
          <p>围绕商品、竞拍和直播流程构建的商家管理后台。</p>
          <div className="intro-grid" aria-label="当前接入能力">
            <span>商品上架</span>
            <span>竞拍活动</span>
            <span>出价记录</span>
          </div>
        </div>

        <section className="auth-panel">
          <div className="mode-tabs" aria-label="认证方式">
            <button
              type="button"
              className={mode === 'login' ? 'active' : ''}
              onClick={() => switchMode('login')}
            >
              登录
            </button>
            <button
              type="button"
              className={mode === 'register' ? 'active' : ''}
              onClick={() => switchMode('register')}
            >
              注册
            </button>
          </div>

          {notice && <div className="notice success">{notice}</div>}
          {error && <div className="notice danger">{error}</div>}

          {mode === 'login' ? (
            <form className="auth-form" onSubmit={handleLoginSubmit}>
              <FormHeader title="商铺登录" description="使用商家账号进入后台" />
              <TextField
                label="账号"
                required
                value={loginValues.username}
                placeholder="请输入商铺登录账号"
                autoComplete="username"
                onChange={(value) =>
                  setLoginValues((current) => ({ ...current, username: value }))
                }
              />
              <TextField
                label="密码"
                required
                type="password"
                value={loginValues.password}
                placeholder="请输入密码"
                autoComplete="current-password"
                onChange={(value) =>
                  setLoginValues((current) => ({ ...current, password: value }))
                }
              />
              <button className="primary-action" type="submit" disabled={status === 'submitting'}>
                {status === 'submitting' ? '正在登录...' : '登录'}
              </button>
            </form>
          ) : (
            <form className="auth-form" onSubmit={handleRegisterSubmit}>
              <FormHeader title="商铺注册" description="创建新的商铺账号" />
              <TextField
                label="账号"
                required
                value={registerValues.username}
                placeholder="请输入商铺登录账号"
                autoComplete="username"
                onChange={(value) =>
                  setRegisterValues((current) => ({ ...current, username: value }))
                }
              />
              <TextField
                label="密码"
                required
                type="password"
                value={registerValues.password}
                placeholder="请输入密码"
                autoComplete="new-password"
                onChange={(value) =>
                  setRegisterValues((current) => ({ ...current, password: value }))
                }
              />
              <TextField
                label="商铺名称"
                required
                value={registerValues.shopName}
                placeholder="请输入商铺名称"
                autoComplete="organization"
                onChange={(value) =>
                  setRegisterValues((current) => ({ ...current, shopName: value }))
                }
              />
              <label className="field">
                <span className="field-label">Logo</span>
                <div className="upload-row">
                  <label className="file-picker">
                    <input
                      type="file"
                      accept="image/png,image/jpeg,image/webp,image/svg+xml"
                      onChange={(event) => setLogoFile(event.target.files?.[0] || null)}
                    />
                    <span>{logoFile ? logoFile.name : '选择图片'}</span>
                  </label>
                  <input
                    className="text-input"
                    value={registerValues.logo}
                    placeholder="或粘贴 Logo URL"
                    onChange={(event) =>
                      setRegisterValues((current) => ({ ...current, logo: event.target.value }))
                    }
                  />
                </div>
              </label>
              <label className="field">
                <span className="field-label">商铺简介</span>
                <textarea
                  value={registerValues.description}
                  maxLength={200}
                  placeholder="简单介绍您的商铺，最多 200 字"
                  onChange={(event) =>
                    setRegisterValues((current) => ({
                      ...current,
                      description: event.target.value,
                    }))
                  }
                />
              </label>
              <div className="inline-fields">
                <TextField
                  label="手机号"
                  value={registerValues.phone}
                  placeholder="选填"
                  autoComplete="tel"
                  onChange={(value) =>
                    setRegisterValues((current) => ({ ...current, phone: value }))
                  }
                />
                <TextField
                  label="邮箱"
                  value={registerValues.email}
                  placeholder="选填"
                  autoComplete="email"
                  onChange={(value) =>
                    setRegisterValues((current) => ({ ...current, email: value }))
                  }
                />
              </div>
              <button className="primary-action" type="submit" disabled={status === 'submitting'}>
                {status === 'submitting' ? '正在创建...' : '创建商铺'}
              </button>
            </form>
          )}
        </section>
      </section>
    </main>
  )
}

function ShopPage({
  shop,
  isLoadingShop,
  onRefreshShop,
}: {
  shop: Shop | null
  isLoadingShop: boolean
  onRefreshShop: () => void
}) {
  return (
    <>
      <div className="page-header">
        <div>
          <h1>商铺设置</h1>
          <p>当前只展示商铺资料，编辑能力可后续接入 /api/shop/:id。</p>
        </div>
        <button type="button" className="ghost-button" onClick={onRefreshShop}>
          刷新资料
        </button>
      </div>
      <section className="panel">
        {isLoadingShop ? (
          <p className="muted">正在读取 /api/shop/me...</p>
        ) : shop ? (
          <dl className="shop-profile">
            <Field label="商铺 ID" value={String(shop.id)} />
            <Field label="登录账号" value={shop.username} />
            <Field label="商铺名称" value={shop.shopName} />
            <Field label="联系电话" value={shop.phone || '未填写'} />
            <Field label="邮箱" value={shop.email || '未填写'} />
            <Field label="简介" value={shop.description || '未填写'} />
          </dl>
        ) : (
          <p className="muted">暂未读取到商铺资料。</p>
        )}
      </section>
    </>
  )
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt>{label}</dt>
      <dd>{value}</dd>
    </div>
  )
}

function FormHeader({ title, description }: { title: string; description: string }) {
  return (
    <div className="form-header">
      <h2>{title}</h2>
      <p>{description}</p>
    </div>
  )
}

function TextField({
  label,
  value,
  onChange,
  placeholder,
  required = false,
  type = 'text',
  autoComplete,
}: {
  label: string
  value: string
  onChange: (value: string) => void
  placeholder: string
  required?: boolean
  type?: 'text' | 'password'
  autoComplete?: string
}) {
  return (
    <label className="field">
      <span className="field-label">
        {label}
        {required && <span className="required"> *</span>}
      </span>
      <input
        className="text-input"
        type={type}
        value={value}
        required={required}
        placeholder={placeholder}
        autoComplete={autoComplete}
        onChange={(event) => onChange(event.target.value)}
      />
    </label>
  )
}

export default App
