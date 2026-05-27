// 商铺信息
export interface Shop {
  id: number;
  username: string;
  shop_name: string;
  logo?: string;
  description?: string;
  phone?: string;
  email?: string;
  status: number;
  audit_status: number;
  audit_reason?: string;
  created_at?: string;
  updated_at?: string;
}

// 登录请求
export interface LoginRequest {
  username: string;
  password: string;
}

// 注册请求
export interface RegisterRequest {
  username: string;
  password: string;
  shop_name: string;
  logo?: string;
  description?: string;
  phone?: string;
  email?: string;
}

// 登录响应
export interface LoginResponse {
  access_token: string;
  token_type: string;
  expires_in: number;
  expires_at: string;
  shop: Shop;
}

// 注册响应
export interface RegisterResponse {
  shop: Shop;
}

// API 响应格式
export interface ApiResponse<T> {
  code: number;
  message: string;
  data?: T;
}

// 认证上下文
export interface AuthContext {
  token: string | null;
  shop: Shop | null;
  isLoading: boolean;
  login: (username: string, password: string) => Promise<void>;
  register: (data: RegisterRequest) => Promise<void>;
  logout: () => void;
}
