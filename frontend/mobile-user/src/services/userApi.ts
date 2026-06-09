import { requestJson } from './request'
import type { EntityID, UserProfile } from '../types/domain'

export type LoginUserPayload = {
  username: string
  password: string
}

export type RegisterUserPayload = {
  username: string
  password: string
  nickname?: string
  avatar?: string
  phone?: string
  email?: string
}

export type LoginUserResult = {
  token: string
  expires_in?: number
}

export type RegisterUserResult = {
  userId: EntityID
}

export type UpdateUserProfilePayload = {
  nickname?: string
  avatar?: string
  gender?: number
  birthday?: string
  phone?: string
  email?: string
}

export type UploadAvatarResult = {
  url: string
  object_key?: string
}

export function loginUser(payload: LoginUserPayload) {
  return requestJson<LoginUserResult>('/api/users/login', {
    method: 'POST',
    body: payload,
  })
}

export function registerUser(payload: RegisterUserPayload) {
  return requestJson<RegisterUserResult>('/api/users/register', {
    method: 'POST',
    body: payload,
  })
}

export function getUserProfile(userID: EntityID, token?: string | null) {
  return requestJson<UserProfile>(`/api/users/${userID}`, { token })
}

export function updateUserProfile(userID: EntityID, payload: UpdateUserProfilePayload, token?: string | null) {
  return requestJson<UserProfile>(`/api/users/${userID}`, {
    method: 'PUT',
    body: payload,
    token,
  })
}

export function uploadUserAvatar(file: File, token?: string | null) {
  const formData = new FormData()
  formData.append('file', file)

  return requestJson<UploadAvatarResult>('/api/users/avatar/upload', {
    method: 'POST',
    body: formData,
    token,
  })
}
