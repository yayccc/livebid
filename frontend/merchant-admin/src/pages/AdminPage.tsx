import { useAuth } from '../hooks/useAuth';
import './AdminPage.css';

export const AdminPage = () => {
  const { shop, logout } = useAuth();

  return (
    <div className="admin-container">
      <div className="admin-header">
        <h1>商铺管理后台</h1>
        <div className="user-info">
          <span className="shop-name">{shop?.shop_name}</span>
          <button className="logout-btn" onClick={logout}>
            退出登录
          </button>
        </div>
      </div>

      <div className="admin-content">
        <div className="placeholder">
          <h2>欢迎来到商铺管理后台</h2>
          <p>功能正在开发中...</p>
          <p>当前登录商铺：<strong>{shop?.shop_name}</strong></p>
          {shop?.description && (
            <p>商铺简介：{shop.description}</p>
          )}
        </div>
      </div>
    </div>
  );
};
