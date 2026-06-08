# Live Service 数据模型

## 设计约定

- 数据库以 MySQL 为目标。
- 主键 ID 使用雪花算法生成。
- 推流码只保存哈希，不保存明文。

## 直播间主表

```sql
CREATE TABLE live_room (
    id BIGINT PRIMARY KEY COMMENT '主键ID，采用雪花算法',

    shop_id BIGINT NOT NULL COMMENT '商铺ID，来自当前登录主体，不信任请求参数',

    title VARCHAR(128) NOT NULL COMMENT '直播间标题',
    cover VARCHAR(255) COMMENT '直播间封面图地址',
    description TEXT COMMENT '直播间简介',

    status TINYINT NOT NULL DEFAULT 1 COMMENT '直播间状态：1-not_live未直播 2-living直播中',

    stream_name VARCHAR(128) NOT NULL COMMENT 'SRS媒体流名称，建议全局唯一',
    stream_key_hash VARCHAR(255) NOT NULL COMMENT '推流码哈希，不保存明文推流码',
    media_stream_status TINYINT NOT NULL DEFAULT 0 COMMENT '媒体流状态：0-offline未推流 1-online推流中',

    actual_start_time DATETIME COMMENT '实际开播时间',
    actual_end_time DATETIME COMMENT '实际关播时间',

    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    is_deleted TINYINT NOT NULL DEFAULT 0 COMMENT '是否删除：0-否 1-是',

    extra JSON COMMENT '扩展字段',

    UNIQUE KEY uk_stream_name(stream_name),
    INDEX idx_shop_id(shop_id),
    INDEX idx_status(status),
    INDEX idx_shop_status(shop_id, status),
    INDEX idx_media_stream_status(media_stream_status)
) COMMENT='直播间主表';
```
