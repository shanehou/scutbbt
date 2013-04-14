package main

import (
	. "../types"
	"flag"
	"github.com/garyburd/redigo/redis"
)

func Subscribe() redis.PubSubConn {
	redisConn, err := RedisConnect(redisServer, redisPasswd)
	if err != nil {
		panic(err)
	}

	psc := redis.PubSubConn{redisConn}
	err = psc.Subscribe(BUS_GPS_DATA, BUS_TIP_INFO)
	if err != nil {
		panic(err)
	}
	return psc
}

func main() {
	flag.Parse()

	psc := Subscribe()
	handler := NewBusDataHandlerWithRedis(redisServer, redisPasswd)

	for {
		switch n := psc.Receive().(type) {
		case redis.Message:
			handler.HandleMessage(n)
		case redis.Subscription:
			continue
		case error:
			panic(n)
		}
	}
}
