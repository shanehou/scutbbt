package main

import (
	. "../types"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	TIP_PREFIX = "TIP:"
)

var (
	addr        string
	redisServer string
	redisPasswd string
	historyFile string
)

func init() {
	flag.StringVar(&addr, "addr", ":6000", "Udp addr to listen")
	flag.StringVar(&redisServer, "redis_server", "127.0.0.1:6379", "redis server")
	flag.StringVar(&redisPasswd, "redis_passwd", "BBTWREDIS", "redis password")
	flag.StringVar(&historyFile, "history_file", "./history.txt", "history file path")
}

func UdpServer(addr string, redisConn redis.Conn) {
	_addr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		log.Println(err)
		os.Exit(-1)
	}
	conn, err := net.ListenUDP("udp", _addr)
	if err != nil {
		log.Println(err)
		os.Exit(-1)
	}
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			log.Println(err)
			os.Exit(-1)
		}
		data := string(buf[:n])

		f, err := os.OpenFile("history.txt", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			log.Println(err)
		} else {
			fmt.Fprintln(f, time.Now(), data)
			f.Close()
		}

		handleData(data, redisConn)
	}
}

func handleData(data string, redisConn redis.Conn) {
	if rawData := parseData(data); rawData != nil {
		if busGPSData, ok := rawData.(*BusGPSData); ok {
			b, err := json.Marshal(busGPSData)
			if err != nil {
				log.Println(err)
				return
			}
			_, err = redisConn.Do("PUBLISH", "BusGPSData", string(b))
			if err != nil {
				log.Println(err)
				return
			}
		}
		if busTipInfo, ok := rawData.(*BusTipInfo); ok {
			b, err := json.Marshal(busTipInfo)
			if err != nil {
				log.Println(err)
				return
			}
			_, err = redisConn.Do("PUBLISH", "BusTipInfo", string(b))
			if err != nil {
				log.Println(err)
				return
			}
		}
	}
}

func parseData(data string) interface{} {
	if data == "" {
		return nil
	}

	fields := strings.Split(data, ",")
	if len(fields) == 1 {
		log.Printf("invalid data: %s", data)
		return nil
	}

	busName := strings.TrimSpace(fields[0])

	if fields[1] == "$GPRMC" {
		if len(fields) < 11 {
			log.Printf("invalid gps data: %s", data)
			return nil
		}

		busGPSData := new(BusGPSData)
		busGPSData.Name = busName

		latitudeStr := fields[4]
		latitude, err := strconv.ParseFloat(latitudeStr, 32)
		if err != nil {
			log.Printf("invalid latitude: %s", fields[4])
			return nil
		}
		busGPSData.Latitude = float32(latitude)

		longitudeStr := fields[6]
		longitude, err := strconv.ParseFloat(longitudeStr, 32)
		if err != nil {
			log.Printf("invalid longitude: %s", fields[6])
			return nil
		}
		busGPSData.Longitude = float32(longitude)

		return busGPSData
	}

	if strings.HasPrefix(fields[1], TIP_PREFIX) {
		if len(fields) != 2 {
			log.Printf("invalid tip: %s", data)
			return nil
		}

		tip := fields[1][len(TIP_PREFIX):]

		return &BusTipInfo{busName, tip}
	}

	log.Printf("invalid data: %s", data)
	return nil
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

func main() {
	flag.Parse()
	redisConn, err := RedisConnect(redisServer, redisPasswd)
	if err != nil {
		os.Exit(-1)
	}
	defer redisConn.Close()
	UdpServer(addr, redisConn)
}
