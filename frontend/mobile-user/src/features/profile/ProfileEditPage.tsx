import type React from 'react'
import { useEffect, useRef, useState } from 'react'
import { Camera, ChevronLeft, ChevronRight, Loader2, Save, UserRound } from 'lucide-react'
import { Navigate, useNavigate } from 'react-router-dom'
import { useAuthProfile } from '../../hooks/useAuthProfile'
import { getErrorText } from '../../lib/format'
import { updateUserProfile, uploadUserAvatar } from '../../services/userApi'
import { useAuthStore } from '../../stores/authStore'

type ProfileDraft = {
  nickname: string
  avatar: string
  gender: string
  birthday: string
  phone: string
  email: string
}

const emptyDraft: ProfileDraft = {
  nickname: '',
  avatar: '',
  gender: '0',
  birthday: '',
  phone: '',
  email: '',
}

export function ProfileEditPage() {
  const navigate = useNavigate()
  const token = useAuthStore((state) => state.token)
  const userID = useAuthStore((state) => state.userID)
  const profile = useAuthStore((state) => state.profile)
  const setProfile = useAuthStore((state) => state.setProfile)
  const profileQuery = useAuthProfile()
  const fileInputRef = useRef<HTMLInputElement | null>(null)
  const coverInputRef = useRef<HTMLInputElement | null>(null)
  const [draft, setDraft] = useState<ProfileDraft>(emptyDraft)
  const [isSaving, setIsSaving] = useState(false)
  const [isUploading, setIsUploading] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    setDraft({
      nickname: profile?.nickname || '',
      avatar: profile?.avatar || '',
      gender: String(profile?.gender ?? 0),
      birthday: normalizeBirthday(profile?.birthday),
      phone: profile?.phone || '',
      email: profile?.email || '',
    })
  }, [profile])

  if (!token) {
    return <Navigate to="/profile" replace />
  }

  function updateDraft(key: keyof ProfileDraft, value: string) {
    setDraft((current) => ({ ...current, [key]: value }))
    setError('')
  }

  async function handleAvatarChange(event: React.ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    event.target.value = ''
    if (!file || isUploading) {
      return
    }

    setIsUploading(true)
    setError('')
    try {
      const uploaded = await uploadUserAvatar(file, token)
      updateDraft('avatar', uploaded.url)
    } catch (err) {
      setError(getErrorText(err, '头像上传失败'))
    } finally {
      setIsUploading(false)
    }
  }

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!userID || isSaving) {
      return
    }

    const nickname = draft.nickname.trim()
    if (!nickname) {
      setError('用户名称不能为空')
      return
    }

    setIsSaving(true)
    setError('')
    try {
      const nextProfile = await updateUserProfile(userID, {
        nickname,
        avatar: draft.avatar.trim(),
        gender: Number(draft.gender) || 0,
        birthday: draft.birthday,
        phone: draft.phone.trim(),
        email: draft.email.trim(),
      }, token)
      setProfile(nextProfile)
      navigate('/profile')
    } catch (err) {
      setError(getErrorText(err, '个人资料保存失败'))
    } finally {
      setIsSaving(false)
    }
  }

  return (
    <div className="profile-page profile-edit-page">
      <header className="profile-edit-hero">
        <button className="profile-edit-hero__back" type="button" onClick={() => navigate('/profile')} aria-label="返回我的">
          <ChevronLeft size={32} aria-hidden="true" />
        </button>
        <button className="profile-edit-hero__cover" type="button" onClick={() => coverInputRef.current?.click()}>
          <Camera size={21} aria-hidden="true" />
          更换封面
        </button>
        <input ref={coverInputRef} className="profile-header__file" type="file" accept="image/*" />
      </header>

      <form className="profile-detail-form" onSubmit={handleSubmit}>
        <section className="profile-edit-sheet">
          <div className="profile-edit-sheet__avatar-wrap">
            <button
              type="button"
              className="profile-edit-sheet__avatar"
              onClick={() => fileInputRef.current?.click()}
              disabled={isUploading}
              aria-label="更换头像"
            >
              {draft.avatar ? <img src={draft.avatar} alt="" /> : <UserRound size={54} aria-hidden="true" />}
              <span>{isUploading ? <Loader2 size={26} aria-hidden="true" /> : <Camera size={30} aria-hidden="true" />}</span>
              <b>更换头像</b>
            </button>
            <input
              ref={fileInputRef}
              className="profile-header__file"
              type="file"
              accept="image/*"
              onChange={handleAvatarChange}
            />
            <div className="profile-edit-sheet__progress" aria-label="资料完成度">
              <span><i style={{ width: `${profileCompletion(draft)}%` }} /></span>
              <b>资料完成度 {profileCompletion(draft)}%</b>
            </div>
          </div>

          <label className="profile-detail-row">
            <span>名字</span>
            <input
              value={draft.nickname}
              maxLength={32}
              placeholder="请输入用户名称"
              onChange={(event) => updateDraft('nickname', event.target.value)}
            />
            <ChevronRight size={24} aria-hidden="true" />
          </label>

          <label className="profile-detail-row">
            <span>性别</span>
            <select value={draft.gender} onChange={(event) => updateDraft('gender', event.target.value)}>
              <option value="0">选择性别，表达自我</option>
              <option value="1">男</option>
              <option value="2">女</option>
            </select>
            <ChevronRight size={24} aria-hidden="true" />
          </label>

          <label className="profile-detail-row">
            <span>生日</span>
            <input
              type="date"
              value={draft.birthday}
              onChange={(event) => updateDraft('birthday', event.target.value)}
            />
            <ChevronRight size={24} aria-hidden="true" />
          </label>

          <label className="profile-detail-row">
            <span>手机号</span>
            <input
              value={draft.phone}
              inputMode="tel"
              placeholder="请输入手机号"
              onChange={(event) => updateDraft('phone', event.target.value)}
            />
            <ChevronRight size={24} aria-hidden="true" />
          </label>

          <label className="profile-detail-row">
            <span>邮箱</span>
            <input
              value={draft.email}
              inputMode="email"
              placeholder="请输入邮箱"
              onChange={(event) => updateDraft('email', event.target.value)}
            />
            <ChevronRight size={24} aria-hidden="true" />
          </label>

          <div className="profile-detail-row profile-detail-row--readonly">
            <span>用户ID</span>
            <b>{userID || '-'}</b>
            <ChevronRight size={24} aria-hidden="true" />
          </div>

          {profileQuery.isLoading ? <p className="profile-detail-form__hint">正在同步用户信息</p> : null}
          {error ? <p className="profile-detail-form__error">{error}</p> : null}

          <button className="profile-detail-form__submit" type="submit" disabled={isSaving}>
            {isSaving ? <Loader2 size={16} aria-hidden="true" /> : <Save size={16} aria-hidden="true" />}
            保存资料
          </button>
        </section>
      </form>
    </div>
  )
}

function profileCompletion(draft: ProfileDraft) {
  const fields = [draft.nickname, draft.avatar, draft.gender !== '0' ? draft.gender : '', draft.birthday, draft.phone, draft.email]
  const completed = fields.filter((value) => String(value).trim()).length

  return Math.round((completed / fields.length) * 100)
}

function normalizeBirthday(value?: string) {
  if (!value) {
    return ''
  }

  return value.slice(0, 10)
}
