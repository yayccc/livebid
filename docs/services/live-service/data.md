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

## 2026-07-17 目标数据模型增量

为支持持久房间、离线互动和独立直播场次，`live_room` 目标增加：

```sql
ALTER TABLE live_room
    ADD COLUMN visibility TINYINT NOT NULL DEFAULT 2
        COMMENT '可见性：1-draft 2-published 3-disabled',
    ADD COLUMN chat_enabled TINYINT NOT NULL DEFAULT 1
        COMMENT '是否允许公开互动：0-否 1-是',
    ADD COLUMN current_live_session_id BIGINT NULL
        COMMENT '当前业务直播场次ID，未开播时为空',
    ADD COLUMN active_publish_client_id VARCHAR(128) NULL
        COMMENT '当前活动SRS推流连接ID，媒体离线时为空',
    ADD COLUMN media_stream_generation BIGINT NOT NULL DEFAULT 0
        COMMENT '每次成功on_publish递增的媒体连接代次',
    ADD INDEX idx_visibility_status (visibility, status),
    ADD INDEX idx_current_live_session_id (current_live_session_id);
```

直播场次使用独立表保存，不把历史场次覆盖在 `live_room.actual_start_time/actual_end_time` 中：

```sql
CREATE TABLE live_session (
    id BIGINT PRIMARY KEY COMMENT '直播场次ID，采用雪花算法',
    room_id BIGINT NOT NULL COMMENT '持久直播间ID',
    shop_id BIGINT NOT NULL COMMENT '开播时商铺ID快照',
    status TINYINT NOT NULL COMMENT '场次状态：1-living 2-ended',
    started_at DATETIME NOT NULL COMMENT '业务开播时间',
    ended_at DATETIME NULL COMMENT '业务结束时间',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_room_started_at (room_id, started_at),
    INDEX idx_shop_started_at (shop_id, started_at),
    INDEX idx_status (status)
) COMMENT='直播业务场次';
```

实施约束：

1. 存量未删除房间回填为 `visibility=published`、`chat_enabled=1`。
2. `StartLive` 在同一事务中创建场次并写入 `live_room.current_live_session_id`。
3. `EndLive` 在同一事务中结束场次并清空当前场次 ID。
4. `actual_start_time/actual_end_time` 在兼容期继续表示当前或最近一次开关播时间；历史事实以 `live_session` 为准。
5. SRS 回调不能修改 `current_live_session_id`。
6. `on_publish` 在条件更新中递增 `media_stream_generation`、替换 `active_publish_client_id` 并置媒体在线。
7. `on_unpublish` 只有在回调 `client_id = active_publish_client_id` 时才能清空连接并置离线；不匹配表示旧连接迟到，只记录审计。
