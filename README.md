## Free-Proxy
    
golang 实现的类似anyproxy的代理服务
    
 

```     
首次运行，会生成默认的根证书和私钥（当然你可以替换自己的）
  -p http://example:port     指定代理（梯子）
  -b :8080 
  -r localhost:6379          请求和相应可以写入redis
  -R rule.yml                支持正则配置
  -l 1/2/3                   输出级别
  -f '*.js*'                 全局过滤

```

rule 规则说明

 ```
version: 1
rules:
  - host: default
    regex: '.*\.mp4'
    option: use-local-response
    content: data/480.mp4
  - host: www.baidu.com
    regex: '/sugrec$'
    option: to-redis
  - host: www.baidu.com
    regex: '.*world='
    option: to-redis

```

- host：指定要过滤的host（default 通配所有host）
- regex： 正则匹配 uri
- option:
    - use-local-response: 用本地内容回包，不包含头部信息
    - to-redis: 将请求，响应输出到redis
   
build:

```
go get github.com/goroom/free-proxy

cd $GOPATH/src/github.com/goroom/free-proxy

go build -o free-proxy cmd/main.go 
```
    
usage examples:
```cassandraql
#./free-proxy -l 2

---------------
> GET /-Po3dSag_xI4khGko9WTAnF6hhy/super/pic/item/1ad5ad6eddc451daba32b647b8fd5266d1163251.jpg

< 200 
---------------
> GET /94o3dSag_xI4khGko9WTAnF6hhy/super/pic/item/8c1001e93901213f58f7f4705ae736d12e2e9552.jpg

< 200 


#./free-proxy -l 2 -f ".*\.js"

---------------
> GET /5eN1bjq8AAUYm2zgoY3K/r/www/cache/static/protocol/https/plugins/https_useable_sample_8f5c5a9.js

< 200 


./free-proxy -p http://example:port
./free-proxy -r localhost:6379
./free-proxy -R rule.yml 
```
