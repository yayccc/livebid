import { useState } from 'react';
import { Tabs, Form, Input, Button, Checkbox, message, Upload } from 'antd';
import { UserOutlined, LockOutlined, ShopOutlined, PhoneOutlined, MailOutlined, PictureOutlined, UploadOutlined } from '@ant-design/icons';
import type { TabsProps } from 'antd';
import { useAuth } from '../hooks/useAuth';
import type { RegisterRequest } from '../types/auth';
import './AuthPage.css';

type TabType = 'login' | 'register';

export const AuthPage = () => {
  const { login, register, isLoading } = useAuth();
  const [activeTab, setActiveTab] = useState<TabType>('login');
  
  const [loginForm] = Form.useForm();
  const [registerForm] = Form.useForm();
  const [agreeTerms, setAgreeTerms] = useState(false);

  // helper: convert file to base64 and set to form field
  const getBase64 = (file: File, cb: (url: string) => void) => {
    const reader = new FileReader();
    reader.addEventListener('load', () => cb(String(reader.result ?? '')));
    reader.readAsDataURL(file);
  };

  const beforeUpload = (file: File) => {
    // convert to base64 preview and store into form field 'logo'
    getBase64(file, (url) => {
      registerForm.setFieldsValue({ logo: url });
      message.success('已选择 Logo（仅本地预览）');
    });
    // prevent auto upload
    return false;
  };

  // 登录提交
  const handleLoginFinish = async (values: { username: string; password: string; rememberMe?: boolean }) => {
    try {
      await login(values.username, values.password);
      message.success('登录成功！');
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : '登录失败';
      message.error(errorMsg);
    }
  };

  // 注册提交：单步完整提交
  const handleRegisterFinish = async (_values?: RegisterRequest) => {
    if (!agreeTerms) {
      message.error('请同意用户协议和商铺入驻协议');
      return;
    }

    try {
      // 从表单读取所有字段，确保多步表单中所有输入都会被包含
      const values = registerForm.getFieldsValue();
      const registerData: RegisterRequest = {
        username: values.username,
        password: values.password,
        shop_name: values.shop_name,
        logo: values.logo || undefined,
        phone: values.phone || undefined,
        email: values.email || undefined,
        description: values.description || undefined,
      };

      await register(registerData);
      message.success('注册成功！正在跳转到登录页面...');

      setTimeout(() => {
        setActiveTab('login');
        
        setAgreeTerms(false);
        registerForm.resetFields();
      }, 2000);
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : '注册失败';
      message.error(errorMsg);
    }
  };

  

  const tabItems: TabsProps['items'] = [
    {
      key: 'login',
      label: '商铺登录',
      children: (
        <Form
          form={loginForm}
          layout="vertical"
          onFinish={handleLoginFinish}
          className="auth-form-antd"
        >
          <Form.Item
            name="username"
            label="用户名"
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input
              prefix={<UserOutlined />}
                    placeholder="请输入商铺登录账号"
              size="large"
              disabled={isLoading}
            />
          </Form.Item>

          <Form.Item
            name="password"
              label="密码"
            rules={[{ required: true, message: '请输入密码' }]}
          >
            <Input.Password
              prefix={<LockOutlined />}
                placeholder="请输入登录密码"
              size="large"
              disabled={isLoading}
            />
          </Form.Item>

          <Form.Item name="rememberMe" valuePropName="checked">
            <Checkbox disabled={isLoading}>记住我的登录</Checkbox>
          </Form.Item>

          <Form.Item>
            <Button
              type="primary"
              htmlType="submit"
              block
              size="large"
              loading={isLoading}
            >
              {isLoading ? '登录中...' : '登录'}
            </Button>
          </Form.Item>

          <div style={{ textAlign: 'right' }}>
            <a href="#">忘记密码？</a>
          </div>
        </Form>
      ),
    },
    {
      key: 'register',
      label: '商铺入驻',
      children: (
        <div>
          <Form
            form={registerForm}
            layout="vertical"
            onFinish={handleRegisterFinish}
            className="auth-form-antd"
          >
            <div className="register-step-content">
              <h3 className="step-title">商铺入驻</h3>

              <Form.Item
                name="username"
                label="用户名"
                rules={[
                  { required: true, message: '请输入用户名' },
                  { min: 3, message: '用户名至少 3 个字符' },
                ]}
              >
                <Input
                  prefix={<UserOutlined />}
                  placeholder="请输入商铺登录账号（将作为登录名）"
                  size="large"
                  disabled={isLoading}
                />
              </Form.Item>

              <Form.Item
                name="password"
                label="密码"
                rules={[
                  { required: true, message: '请输入密码' },
                  { min: 6, message: '密码至少 6 个字符' },
                ]}
              >
                <Input.Password
                  prefix={<LockOutlined />}
                  placeholder="请输入登录密码（建议不少于6位）"
                  size="large"
                  disabled={isLoading}
                />
              </Form.Item>

              <Form.Item
                name="shop_name"
                label="商铺名称"
                rules={[
                  { required: true, message: '请输入商铺名称' },
                  { min: 2, message: '商铺名称至少 2 个字符' },
                ]}
              >
                <Input
                  prefix={<ShopOutlined />}
                  placeholder="请输入商铺名称"
                  size="large"
                  disabled={isLoading}
                />
              </Form.Item>

              <Form.Item
                name="logo"
                label="商铺 Logo"
              >
                <Upload
                  beforeUpload={beforeUpload}
                  showUploadList={false}
                  accept="image/*"
                >
                  <Button icon={<UploadOutlined />}>选择或上传 Logo</Button>
                </Upload>
                <div style={{ marginTop: 8 }}>
                  <Input
                    prefix={<PictureOutlined />}
                    placeholder="Logo URL 或已上传图片预览（仅本地）"
                    size="large"
                    disabled
                    value={registerForm.getFieldValue('logo') || ''}
                  />
                </div>
              </Form.Item>

              <Form.Item
                name="phone"
                label="手机号码"
                rules={[
                  { pattern: /^1[3-9]\d{9}$/, message: '请输入有效的手机号码' },
                ]}
              >
                <Input
                  prefix={<PhoneOutlined />}
                  placeholder="请输入手机号码"
                  size="large"
                  disabled={isLoading}
                />
              </Form.Item>

              <Form.Item
                name="email"
                label="邮箱"
                rules={[{ type: 'email', message: '请输入有效的邮箱地址' }]}
              >
                <Input
                  prefix={<MailOutlined />}
                  placeholder="请输入邮箱"
                  size="large"
                  disabled={isLoading}
                />
              </Form.Item>

              <Form.Item
                name="description"
                label="商铺简介"
                rules={[{ max: 500, message: '商铺简介不能超过 500 个字符' }]}
              >
                <Input.TextArea
                  placeholder="简述商铺的核心竞争力、主要商品、服务特色等，帮助买家快速了解商铺"
                  rows={5}
                  disabled={isLoading}
                />
              </Form.Item>

              <Form.Item>
                <Checkbox
                  checked={agreeTerms}
                  onChange={(e) => setAgreeTerms(e.target.checked)}
                  disabled={isLoading}
                >
                  同意
                  <a href="#">《用户协议》</a>
                  及
                  <a href="#">《商铺入驻协议》</a>
                </Checkbox>
              </Form.Item>

              <Form.Item>
                <Button
                  type="primary"
                  htmlType="submit"
                  size="large"
                  loading={isLoading}
                  disabled={!agreeTerms}
                >
                  {isLoading ? '注册中...' : '完成注册并开店'}
                </Button>
              </Form.Item>
            </div>
          </Form>
        </div>
      ),
    },
  ];

  return (
    <div className="auth-container">
      {/* 左侧品牌区域 */}
      <div className="auth-brand">
        <div className="brand-logo">
          <div className="logo-icon">🎯</div>
          <div className="logo-text">
            <div className="logo-title">AUCTION ENGINE</div>
            <div className="logo-subtitle">直播竞拍平台</div>
          </div>
        </div>

        <div className="brand-content">
          <h2 className="brand-title">高并发实时竞拍</h2>

          <div className="feature-list">
            <div className="feature-item">
              <div className="feature-icon">⚡</div>
              <div className="feature-text">
                <div className="feature-name">极限高并发架构设计</div>
                <div className="feature-desc">
                  基于 gRPC 微服务，支持秒级交易、毫秒级推送...
                </div>
              </div>
            </div>

            <div className="feature-item">
              <div className="feature-icon">🤖</div>
              <div className="feature-text">
                <div className="feature-name">AI 驱动的价格预测</div>
                <div className="feature-desc">
                  智能算法实时分析市场，精准把握价格走势...
                </div>
              </div>
            </div>

            <div className="feature-item">
              <div className="feature-icon">📊</div>
              <div className="feature-text">
                <div className="feature-name">产销端数据融合</div>
                <div className="feature-desc">
                  连接厂商与消费者，实现透明高效的交易...
                </div>
              </div>
            </div>
          </div>
        </div>

        <div className="brand-stats">
          <div className="stat-item">
            <div className="stat-value">99.99%</div>
            <div className="stat-label">系统稳定</div>
          </div>
          <div className="stat-separator"></div>
          <div className="stat-item">
            <div className="stat-value">&lt; 5ms</div>
            <div className="stat-label">响应速度</div>
          </div>
          <div className="stat-separator"></div>
          <div className="stat-item">
            <div className="stat-value">180+</div>
            <div className="stat-label">商户入驻</div>
          </div>
        </div>
      </div>

      {/* 右侧表单区域 */}
      <div className="auth-form-wrapper">
        <Tabs
          activeKey={activeTab}
          onChange={(key) => setActiveTab(key as TabType)}
          items={tabItems}
          className="auth-tabs-antd"
        />
      </div>
    </div>
  );
};
