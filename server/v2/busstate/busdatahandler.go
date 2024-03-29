package main

import (
	. "../types"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"os"
	"time"
)

const (
	FLY_DISTANCE = 0.05
)

var (
	flyLogFile string
)

func init() {
	flag.StringVar(&flyLogFile, "fly_log_path", "fly.log", "fly log path")
}

type BusDataHandler struct {
	redisConn redis.Conn
}

// 设置最后收到GPRS数据时间
func (h *BusDataHandler) touchBus(busName string) {
	_, err := h.redisConn.Do("HSET", "BUS:"+busName+":INFO", "LastGPRS", time.Now().Unix())
	if err != nil {
		panic(err)
	}
	_, err = h.redisConn.Do("EXPIRE", "BUS:"+busName+":INFO", 60)
	if err != nil {
		panic(err)
	}
}

// 从redis中获取校巴
func (h *BusDataHandler) fetchBus(busName string) (coordIndex int, isNew bool) {
	reply, err := redis.Values(h.redisConn.Do("HMGET", "BUS:"+busName+":INFO", "Name", "CoordIndex"))
	if err != nil {
		panic(err)
	}

	// 若没有此校巴的记录，则返回新校巴
	if reply[0] == nil {
		isNew = true
		return
	}

	_, err = redis.Scan(reply[1:], &coordIndex)
	if err != nil {
		panic(err)
	}
	return
}

// 记录跑飞到文件
func (h *BusDataHandler) recordFly(data *BusGPSData, index int, coordIndex int, distance float64) {
	f, err := os.OpenFile(flyLogFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	fmt.Fprintln(f, data, index, coordIndex, distance)
}

// 判断校巴为跑飞
func (h *BusDataHandler) isFly(distance float64) bool {
	return distance > FLY_DISTANCE
}

// 删除校巴
func (h *BusDataHandler) delBus(busName string) {
	_, err := h.redisConn.Do("DEL", "BUS:"+busName+":INFO")
	if err != nil {
		panic(err)
	}
}

// 设置校巴跑飞
func (h *BusDataHandler) setFly(busName string) {
	_, err := h.redisConn.Do("HMSET", "BUS:"+busName+":INFO",
		"Fly", true,
		"LastGPS", time.Now().Unix(),
		"GPS_OK", true,
	)
	if err != nil {
		panic(err)
	}
}

func (h *BusDataHandler) handleGPSData(data *BusGPSData) {
	busName := data.Name

	oldCoordIndex, isNew := h.fetchBus(busName)

	// 获取最近的站点，设置站名，索引和距离
	index, percent, distance, coordIndex := GetClosest(data.Latitude, data.Longitude)

	// 校巴在总站，校巴已停，删除校巴
	if index == 0 {
		h.delBus(busName)
		return
	}

	// 判断跑飞
	if h.isFly(distance) {
		h.recordFly(data, index, coordIndex, distance)
		h.setFly(busName)
		h.touchBus(busName)
	}

	var stationIndex int
	var station string
	var direction bool

	// 站点为0时，强制设置为第一个站
	if index == 0 {
		stationIndex = 1
		station = IndexToStation[1]
	} else {
		stationIndex = index
		station = IndexToStation[index]
	}

	// 若busData不是该校巴收到的第一个数据，才进行方向和离开站点的判断
	if !isNew {
		if oldCoordIndex > coordIndex { // 若站点索引减小，则为开往站点索引小的方向
			direction = true
		} else if oldCoordIndex < coordIndex {
			direction = false
		} else {
			// 站点索引未变化时保持原方向
		}
	}

	// 在车场，百分比强制归0
	if index == 1 && percent < 0 || index == 0 {
		percent = 0
	}

	// 反向，百分比相反
	if direction == true {
		percent *= -1
	}

	_, err := h.redisConn.Do("HMSET", "BUS:"+busName+":INFO",
		"Name", busName,
		"Latitude", data.Latitude,
		"Longitude", data.Longitude,
		"CoordIndex", coordIndex,
		"StationIndex", stationIndex,
		"Station", station,
		"Percent", percent,
		"Direction", direction,
		"LastGPS", time.Now().Unix(),
		"GPS_OK", true,
		"FLY", false,
	)
	if err != nil {
		panic(err)
	}
	h.touchBus(busName)
}

func (h *BusDataHandler) handleTipInfo(data *BusTipInfo) {
	busName := data.Name
	switch data.Tip {
	case "GPS ERROR":
		fallthrough
	case "GPS_WRONG":
		_, err := h.redisConn.Do("HSET", "BUS:"+busName+":INFO", "GPS_OK", false)
		if err != nil {
			panic(err)
		}
	case "GPRS succeed":
		h.touchBus(busName)
		_, err := h.redisConn.Do("LPUSH", "BUS:"+busName+":SUCCEED", time.Now().Unix())
		if err != nil {
			panic(err)
		}
	case "GPRS reconnect":
		h.touchBus(busName)
		_, err := h.redisConn.Do("LPUSH", "BUS:"+busName+":RECONNECT", time.Now().Unix())
		if err != nil {
			panic(err)
		}
	}
}

func (h *BusDataHandler) HandleMessage(msg redis.Message) {
	switch msg.Channel {
	case BUS_GPS_DATA:
		var busGPSData BusGPSData
		err := json.Unmarshal(msg.Data, &busGPSData)
		if err != nil {
			panic(err)
		}
		h.handleGPSData(&busGPSData)
		return
	case BUS_TIP_INFO:
		var busTipInfo BusTipInfo
		err := json.Unmarshal(msg.Data, &busTipInfo)
		if err != nil {
			panic(err)
		}
		h.handleTipInfo(&busTipInfo)
		return
	}
	panic("not reached")
}

func NewBusDataHandlerWithRedis(redisServer, redisPasswd string) *BusDataHandler {
	redisConn, err := RedisConnect(redisServer, redisPasswd)
	if err != nil {
		panic(err)
	}

	return &BusDataHandler{redisConn: redisConn}
}
