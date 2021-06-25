## Dynamic Application Configuration Management

支持监听阿里云acm、nacos配置变化并更新本地配置文件, 支持同步完成后执行自定义命令

- 自定义执行命令支持配置延时执行、超时时间控制
- 支持同时监听多个命名空间、多个group、多个data id的配置

配置文件说明

```text
{
     "cluster_type": "nacos", // 集群模式支持acm和nacos, acm配置请参考config/acm.json.example
     "cluster_nodes": [ // nacos 集群节点列表
         {
             "ip": "10.1.5.83",
             "port": 8848
         }
     ],
     "log_level": "debug", // 日志级别
     "cache_dir": "/data/dacm/cache", //缓存目录
     "log_dir": "/data/dacm/log", // 日志目录
     "rotate_time": "1h", // 日志滚动时间
     "max_age": 3,  // 保留最近3小时的日志
     "namespaces": [ // 命名空间配置
         {
             "id": "02d4cb2d-2833-48a2-a8ea-bdcee4259359", // 命名空间id
             "name": "sunline-uat", // 命名空间名称
             "username": "", // 授权用户名
             "password": "", // 授权用户密码
             "configs": [
                 {
                     "data_id": "uat-sunline-snactiv-core", // 配置data id
                     "group": "DEFAULT_GROUP", // 配置所在的group
                     "sync_file": "/data/apps/sunline/snactiv/server.properties", // 需要动态更新的本地配置文件所在路径
                     "execute": "cd /data/apps/sunline/snactiv/; source /etc/profile ${HOME}/.bash_profile; sh start.sh", // 动态更新本地配置文件之后执行自定义命令，如果为空默认只更新本地配置文件
                     "execute_delay": 10, // 自定义命令执行延时时间，单位毫秒
                     "execute_timeout": 10000, // 自定义命令执行超时时间，单位毫秒
                     "not_load_cache_at_start": true, // 设置为true
                     "timeout": 5000 // 监听配置超时时间
                 }
             ]
         }
     ],
     "max_cpu_rate": 1, // 允许绑定cpu核心数占比(cpu核心数*max_cpu_rate)
     "max_mem_rate": 1 // 允许使用的内存占比(内存总大小*max_mem_rate)
}
```

使用systemd管理服务

```text
将压缩包解压到 /usr/local/dacm/ 目录, 创建配置文件 /usr/local/dacm/config/app.json，然后执行以下命令

cd /usr/local/dacm/
cp systemd/dacm.service /usr/lib/systemd/system/dacm.service
systemctl daemon-reload
systemctl start dacm
systemctl enable dacm
systemctl status dacm
```