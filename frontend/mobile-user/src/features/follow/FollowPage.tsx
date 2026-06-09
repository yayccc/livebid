import { Button } from 'antd-mobile'
import { Heart } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { EmptyState } from '../../components/common/EmptyState'

export function FollowPage() {
  const navigate = useNavigate()

  return (
    <div className="plain-page plain-page--dark">
      <EmptyState
        title="关注功能开发中"
        description="后续会展示你关注的直播间和主播动态"
        action={
          <Button color="primary" size="small" onClick={() => navigate('/')}>
            <Heart size={16} aria-hidden="true" />
            去首页看看
          </Button>
        }
      />
    </div>
  )
}
