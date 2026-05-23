
## 商铺主表

```
CREATE TABLE shop (
    id BIGINT PRIMARY KEY COMMENT '主键ID，采用雪花算法',

    -- 登录信息
    username VARCHAR(64) NOT NULL COMMENT '商铺登录账号',
    password_hash VARCHAR(255) NOT NULL COMMENT '密码密文（BCrypt算法）',

    -- 基础信息
    shop_name VARCHAR(128) NOT NULL COMMENT '商铺名称',
    logo VARCHAR(255) COMMENT '商铺头像',
    description TEXT COMMENT '商铺简介',

    -- 联系信息
    phone VARCHAR(20) COMMENT '绑定手机号',
    email VARCHAR(128) COMMENT '邮箱',

    -- 状态控制（使用数值标志位）
    status TINYINT NOT NULL DEFAULT 1 COMMENT '商铺状态：1-启用 2-禁用 3-封禁 4-审核中',
    -- 审核信息
    audit_status TINYINT DEFAULT 0 COMMENT '审核状态：0-未审核 1-通过 2-拒绝',
    audit_reason VARCHAR(255) COMMENT '审核失败原因',

    -- 通用审计字段
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    is_deleted TINYINT DEFAULT 0 COMMENT '是否删除 0-否 1-是'   
    
    -- 扩展字段
    extra JSON COMMENT '扩展字段'
    -- 索引设计
    UNIQUE KEY uk_username(username),
    UNIQUE KEY uk_shop_name(shop_name),
    INDEX idx_status(status)
);
```

## 商铺登录日志表
```
CREATE TABLE shop_login_log (
    id BIGINT PRIMARY KEY COMMENT '主键ID，采用雪花算法',

    shop_id BIGINT NOT NULL COMMENT '商铺ID，关联shop表主键',

    login_ip VARCHAR(64) COMMENT '登录IP地址',
    user_agent VARCHAR(255) COMMENT '客户端User-Agent信息',

    login_time DATETIME NOT NULL COMMENT '登录时间',

    login_result TINYINT NOT NULL COMMENT '登录结果：1-成功 2-失败',

    -- 扩展字段（防未来变更）
    extra JSON COMMENT '扩展字段（如设备信息、地理位置等）',

    -- 通用审计字段
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    is_deleted TINYINT DEFAULT 0 COMMENT '是否删除 0-否 1-是'   
    -- 索引设计
    INDEX idx_shop_id(shop_id),
    INDEX idx_login_time(login_time),
    INDEX idx_shop_time(shop_id, login_time)
);
```


接口设计
功能
接口
方法
商铺注册
/api/shop/register
POST
商铺登录
/api/shop/login
POST
查询商铺信息
/api/shop/{id}
GET
修改商铺信息
/api/shop/{id}
PUT
