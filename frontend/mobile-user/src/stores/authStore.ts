import { create } from 'zustand'
import { getJwtSubject, isJwtExpired } from '../lib/authToken'
import type { EntityID, UserProfile } from '../types/domain'

const tokenStorageKey = 'livebid_user_access_token'

type AuthState = {
  token: string | null
  userID: EntityID | null
  profile: UserProfile | null
  setSession: (token: string, profile?: UserProfile | null) => void
  setProfile: (profile: UserProfile | null) => void
  clearSession: () => void
}

function getInitialToken() {
  const token = localStorage.getItem(tokenStorageKey)
  if (!token || isJwtExpired(token)) {
    localStorage.removeItem(tokenStorageKey)
    return null
  }

  return token
}

const initialToken = getInitialToken()

export const useAuthStore = create<AuthState>((set) => ({
  token: initialToken,
  userID: getJwtSubject(initialToken),
  profile: null,
  setSession: (token, profile = null) => {
    localStorage.setItem(tokenStorageKey, token)
    set({
      token,
      userID: getJwtSubject(token),
      profile,
    })
  },
  setProfile: (profile) => set({ profile }),
  clearSession: () => {
    localStorage.removeItem(tokenStorageKey)
    set({
      token: null,
      userID: null,
      profile: null,
    })
  },
}))
