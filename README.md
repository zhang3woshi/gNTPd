# NTP-Server

### 提供NTP校时服务

1.支持NTP协议版本4，仅支持主要功能，其他功能未实现

2.定时从上游获取当前时间，本地时间同远端时间相差超过1秒则立即更新本地时间

3.更新本地时间是只支持到秒级，毫秒级会引起不可预测错误

4.从上游获取时间失败则不更新本地时间

5.可通过修改配置文件关闭UDP服务仅作为客户端使用



### 配置文件

```json
{
  "ntp_server": "pool.ntp.org",
  "ntp_server_port": 123,
  "server_mode": true,
  "service_port": 123,
  "interval": 60
}
```

| 名称            | 描述                | 备注        |
| --------------- | ------------------- | ----------- |
| ntp_server      | 上游NTP服务地址     |             |
| ntp_server_port | 上游NTP服务端口     | 默认123     |
| server_mode     | 是否开启本地NTP服务 | 默认开启    |
| service_port    | 本地服务端口        |             |
| interval        | 向上游请求间隔      | 单位秒(sec) |

