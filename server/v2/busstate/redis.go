package main

import (
	"flag"
	"github.com/garyburd/redigo/redis"
	"log"
)

var (
	redisServer string
	redisPasswd string
)

func init() {
	flag.StringVar(&redisServer, "redis_server", "127.0.0.1:6379", "redis server")
	flag.StringVar(&redisPasswd, "redis_passwd", "BBTWREDIS", "redis password")
}

// 连接 redis
func RedisConnect(redisServer string, redisPasswd string) (conn redis.Conn, err error) {
	conn, err = redis.Dial("tcp", redisServer)
	if err != nil {
		log.Println(err)
		return
	}
	// 认证
	_, err = conn.Do("AUTH", redisPasswd)
	if err != nil {
		log.Println(err)
		return
	}
	return
}
