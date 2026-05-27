import { useState, useCallback, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { apiService } from '../services/api';
import type { Shop, RegisterRequest } from '../types/auth';

export const useAuth = () => {
  const navigate = useNavigate();
  const [token, setToken] = useState<string | null>(null);
  const [shop, setShop] = useState<Shop | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  // 初始化：从 localStorage 恢复认证状态
  useEffect(() => {
    const savedToken = localStorage.getItem('access_token');
    const savedShop = localStorage.getItem('shop');

    if (savedToken) {
      setToken(savedToken);
    }

    if (savedShop) {
      try {
        setShop(JSON.parse(savedShop));
      } catch {
        localStorage.removeItem('shop');
      }
    }
  }, []);

  // 登录
  const login = useCallback(
    async (username: string, password: string) => {
      setIsLoading(true);
      try {
        const response = await apiService.login({ username, password });

        // 保存 token 和 shop 信息
        localStorage.setItem('access_token', response.access_token);
        localStorage.setItem('shop', JSON.stringify(response.shop));

        setToken(response.access_token);
        setShop(response.shop);

        // 导航到管理后台
        navigate('/admin', { replace: true });
      } finally {
        setIsLoading(false);
      }
    },
    [navigate]
  );

  // 注册
  const register = useCallback(
    async (data: RegisterRequest) => {
      setIsLoading(true);
      try {
        await apiService.register(data);
        // 注册成功，提示用户去登录
        // 这里可以通过返回的响应让组件决定是否自动登录
      } finally {
        setIsLoading(false);
      }
    },
    []
  );

  // 登出
  const logout = useCallback(() => {
    localStorage.removeItem('access_token');
    localStorage.removeItem('shop');
    setToken(null);
    setShop(null);
    navigate('/', { replace: true });
  }, [navigate]);

  return {
    token,
    shop,
    isLoading,
    login,
    register,
    logout,
    isAuthenticated: !!token,
  };
};
