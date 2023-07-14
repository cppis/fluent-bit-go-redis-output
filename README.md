# [fluent-bit redis output plugin](https://github.com/majst01/fluent-bit-go-redis-output) 수정

[![Build Status](https://travis-ci.org/majst01/fluent-bit-go-redis-output.svg?branch=master)](https://travis-ci.org/majst01/fluent-bit-go-redis-output)
[![codecov](https://codecov.io/gh/majst01/fluent-bit-go-redis-output/branch/master/graph/badge.svg)](https://codecov.io/gh/majst01/fluent-bit-go-redis-output)
[![Go Report Card](https://goreportcard.com/badge/majst01/fluent-bit-go-redis-output)](https://goreportcard.com/report/github.com/majst01/fluent-bit-go-redis-output)

fluent-bit TCP input 으로 받은 json 을 파싱하면서 문자열이 Base64 인코딩이 되는 이슈가 있어서 개선 중임.  

<br/><br/>

## Usage

```bash
docker run -it --rm -v /path/to/fluent-bit.conf:/fluent-bit/etc/fluent-bit.conf majst01/fluent-bit-go-redis-output
```

### Building

```bash
docker build --no-cache --tag localhost:5001/fluent-bit-go-redis-output:latest .
docker push localhost:5001/fluent-bit-go-redis-output:latest

go build -ldflags "-X 'main.revision=00000000' -X 'main.builddate=023-07-14 14:00:00+09:00'" -buildmode=c-shared -o out_redis.so .

docker run -it --rm -v /path/to/fluent-bit.conf:/fluent-bit/etc/fluent-bit.conf fluent-bit-go-redis-output
```

### Configuration Options

| Key           | Description                                    | Default        |
| --------------|------------------------------------------------|----------------|
| Hosts         | Host(s) of redis servers, whitespace separated ip/host:port | 127.0.0.1:6379 |
| Password      | Optional redis password for all redis instances | "" |
| DB            | redis database (integer)  | 0 |
| UseTLS        | connect to redis with tls | False |
| TlsSkipVerify | if tls is configured skip tls certificate validation for self signed certificates | True |
| Key           | the key where to store the entries in redis | "logstash" |


Example:

add this section to fluent-bit.conf

```properties
[Output]
    Name redis
    Match *
    UseTLS true
    TLSSkipVerify true
    # if port is ommited, 6379 is used
    Hosts 172.17.0.1 172.17.0.1:6380 172.17.0.1:6381 172.17.0.1:6382 172.17.0.1:6383
    Password homer
    DB 0
    Key elastic-logstash
```

<br/><br/>


## Fixes  

* builder 의 Go 버전을 1.19 에서 1.18 로 수정  

  `/lib/x86_64_linux_gnu/libc.so.6: version 'GLIBC_2.32' not found` 에러를 수정하기 위해 Go 버전을 GLIBC_2.31 을 사용하도록 Go 버전 다운그레이드.   
* 

<br/><br/>

## References  

### Redis format

- [logrus-redis-hook](https://github.com/rogierlommers/logrus-redis-hook/blob/master/logrus_redis.go)

### Logstash Redis Output

- [logstash-redis-docu](https://github.com/logstash-plugins/logstash-output-redis/blob/master/docs/index.asciidoc)
