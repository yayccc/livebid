import type React from 'react'
import { useState } from 'react'
import { Button, Toast } from 'antd-mobile'
import { Lock, UserRound } from 'lucide-react'
import { getJwtSubject } from '../../lib/authToken'
import { getErrorText } from '../../lib/format'
import { getUserProfile, loginUser, registerUser } from '../../services/userApi'
import { useAuthStore } from '../../stores/authStore'

type AuthMode = 'login' | 'register'
type FormStatus = 'idle' | 'submitting'

type LoginValues = {
  username: string
  password: string
}

type RegisterValues = {
  username: string
  password: string
}

export function AuthPanel() {
  const setSession = useAuthStore((state) => state.setSession)
  const setProfile = useAuthStore((state) => state.setProfile)
  const [mode, setMode] = useState<AuthMode>('login')
  const [status, setStatus] = useState<FormStatus>('idle')
  const [error, setError] = useState('')
  const [loginValues, setLoginValues] = useState<LoginValues>({ username: '', password: '' })
  const [registerValues, setRegisterValues] = useState<RegisterValues>({
    username: '',
    password: '',
  })

  function handleModeChange(nextMode: AuthMode) {
    setMode(nextMode)
    setError('')
  }

  async function handleLogin(event: React.FormEvent<HTMLFormElement>) {
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
      const result = await loginUser({ username, password })
      setSession(result.token)
      const userID = getJwtSubject(result.token)
      if (userID) {
        try {
          const profile = await getUserProfile(userID, result.token)
          setProfile(profile)
        } catch {
          setProfile(null)
        }
      }
      Toast.show('登录成功')
    } catch (err) {
      setError(getErrorText(err, '登录失败，请检查账号或密码'))
    } finally {
      setStatus('idle')
    }
  }

  async function handleRegister(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const username = registerValues.username.trim()
    const password = registerValues.password
    if (!username || !password) {
      setError('账号和密码为必填项')
      return
    }

    setStatus('submitting')
    setError('')
    try {
      await registerUser({
        username,
        password,
      })
      setLoginValues({ username, password: '' })
      setRegisterValues({
        username: '',
        password: '',
      })
      setMode('login')
      Toast.show('注册成功，请登录')
    } catch (err) {
      setError(getErrorText(err, '注册失败，请稍后重试'))
    } finally {
      setStatus('idle')
    }
  }

  return (
    <section className="auth-panel" aria-label="用户登录注册">
      <div className="auth-panel__brand">
        <div className="auth-panel__logo" aria-hidden="true">LB</div>
        <h1>登录后，体验完整功能</h1>
      </div>

      {mode === 'login' ? (
        <form className="auth-form" onSubmit={handleLogin}>
          <label>
            <span>账号/邮箱/手机号</span>
            <UserRound size={21} aria-hidden="true" />
            <input
              className="auth-form__input"
              value={loginValues.username}
              placeholder="请输入您的账号/邮箱/手机号"
              autoComplete="username"
              onChange={(event) => setLoginValues((current) => ({ ...current, username: event.target.value }))}
            />
          </label>
          <label>
            <span>密码</span>
            <Lock size={20} aria-hidden="true" />
            <input
              className="auth-form__input"
              value={loginValues.password}
              placeholder="请输入密码"
              type="password"
              autoComplete="current-password"
              onChange={(event) => setLoginValues((current) => ({ ...current, password: event.target.value }))}
            />
          </label>
          {error && <p className="auth-form__error">{error}</p>}
          <Button block color="primary" type="submit" loading={status === 'submitting'}>
            登录
          </Button>
          <p className="auth-form__switch">
            没有账号？
            <button type="button" onClick={() => handleModeChange('register')}>去注册</button>
          </p>
        </form>
      ) : (
        <form className="auth-form" onSubmit={handleRegister}>
          <label>
            <span>账号/邮箱/手机号</span>
            <UserRound size={21} aria-hidden="true" />
            <input
              className="auth-form__input"
              value={registerValues.username}
              placeholder="请输入您的账号/邮箱/手机号"
              autoComplete="username"
              onChange={(event) => setRegisterValues((current) => ({ ...current, username: event.target.value }))}
            />
          </label>
          <label>
            <span>密码</span>
            <Lock size={20} aria-hidden="true" />
            <input
              className="auth-form__input"
              value={registerValues.password}
              placeholder="请输入密码"
              type="password"
              autoComplete="new-password"
              onChange={(event) => setRegisterValues((current) => ({ ...current, password: event.target.value }))}
            />
          </label>
          {error && <p className="auth-form__error">{error}</p>}
          <Button block color="primary" type="submit" loading={status === 'submitting'}>
            创建账号
          </Button>
          <p className="auth-form__switch">
            已有账号？
            <button type="button" onClick={() => handleModeChange('login')}>去登录</button>
          </p>
        </form>
      )}
    </section>
  )
}
