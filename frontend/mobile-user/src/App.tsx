import { ConfigProvider } from 'antd-mobile'
import zhCN from 'antd-mobile/es/locales/zh-CN'
import { Navigate, RouterProvider, createBrowserRouter } from 'react-router-dom'
import { AppShell } from './components/common/AppShell'
import { FollowPage } from './features/follow/FollowPage'
import { LiveHomePage } from './features/live/LiveHomePage'
import { MessagesPage } from './features/messages/MessagesPage'
import { ProfilePage } from './features/profile/ProfilePage'
import './App.css'

const router = createBrowserRouter([
  {
    path: '/',
    element: <AppShell />,
    children: [
      { index: true, element: <LiveHomePage /> },
      { path: 'follow', element: <FollowPage /> },
      { path: 'messages', element: <MessagesPage /> },
      { path: 'profile', element: <ProfilePage /> },
      { path: '*', element: <Navigate to="/" replace /> },
    ],
  },
])

function App() {
  return (
    <ConfigProvider locale={zhCN}>
      <RouterProvider router={router} />
    </ConfigProvider>
  )
}

export default App
