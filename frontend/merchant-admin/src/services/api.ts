import axios from 'axios';
import type { AxiosInstance } from 'axios';
import type {
  LoginRequest,
  RegisterRequest,
  LoginResponse,
  RegisterResponse,
  ApiResponse,
} from '../types/auth';

const API_BASE_URL = 'http://localhost:8080/api';

class ApiService {
  private client: AxiosInstance;

  constructor() {
    this.client = axios.create({
      baseURL: API_BASE_URL,
      timeout: 10000,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // 请求拦截器：添加 token
    this.client.interceptors.request.use((config) => {
      const token = localStorage.getItem('access_token');
      if (token) {
        config.headers.Authorization = `Bearer ${token}`;
      }
      return config;
    });

    // 响应拦截器：处理错误
    this.client.interceptors.response.use(
      (response) => response,
      (error) => {
        if (error.response?.status === 401) {
          // Token 过期或无效，清除本地存储
          localStorage.removeItem('access_token');
          localStorage.removeItem('shop');
          window.location.href = '/';
        }
        return Promise.reject(error);
      }
    );
  }

  /**
   * 商铺登录
   */
  async login(data: LoginRequest): Promise<LoginResponse> {
    const response = await this.client.post<ApiResponse<LoginResponse>>(
      '/shop/login',
      data
    );

    if (response.data.code !== 0) {
      throw new Error(response.data.message || '登录失败');
    }

    if (!response.data.data) {
      throw new Error('登录失败：服务器响应异常');
    }

    return response.data.data;
  }

  /**
   * 商铺注册
   */
  async register(data: RegisterRequest): Promise<RegisterResponse> {
    const response = await this.client.post<ApiResponse<RegisterResponse>>(
      '/shop/register',
      data
    );

    if (response.data.code !== 0) {
      throw new Error(response.data.message || '注册失败');
    }

    if (!response.data.data) {
      throw new Error('注册失败：服务器响应异常');
    }

    return response.data.data;
  }
}

export const apiService = new ApiService();
