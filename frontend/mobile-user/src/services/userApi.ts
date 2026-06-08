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
