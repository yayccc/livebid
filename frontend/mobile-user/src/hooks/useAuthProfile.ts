import { useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { getUserProfile } from '../services/userApi'
import { useAuthStore } from '../stores/authStore'

export function useAuthProfile() {
  const token = useAuthStore((state) => state.token)
  const userID = useAuthStore((state) => state.userID)
  const setProfile = useAuthStore((state) => state.setProfile)
  const clearSession = useAuthStore((state) => state.clearSession)

  const query = useQuery({
    queryKey: ['user-profile', userID],
    enabled: Boolean(token && userID),
    queryFn: () => getUserProfile(userID as string, token),
    retry: 1,
  })

  useEffect(() => {
    if (query.data) {
      setProfile(query.data)
    }
  }, [query.data, setProfile])

  useEffect(() => {
    if (query.error) {
      clearSession()
    }
  }, [clearSession, query.error])

  return query
}
