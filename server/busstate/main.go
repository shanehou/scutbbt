package main

import (
	. "../types"
	"encoding/json"
	"flag"
	"github.com/garyburd/redigo/redis"
	"time"
)

type BusDataHandler struct {
	redisConn redis.Conn
}

func (h *BusDataHandler) touchBus(busName string) {
	_, err := h.redisConn.Do("HSET", "BUS:"+busName, "LastGPRS", time.Now().Unix())
	if err != nil {
		panic(err)
	}
}

func (h *BusDataHandler) fetchBus(busName string) (coordIndex int, isNew bool) {
	reply, err := redis.Values(h.redisConn.Do("HMGET", "BUS:"+busName, "Name", "CoordIndex"))
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
func (h *BusDataHandler) handleGPSData(data *BusGPSData) {
	busName := data.Name

	// 从redis中获取校巴索引，方向和是否为新校巴
	oldCoordIndex, isNew := h.fetchBus(busName)

	// 获取最近的站点，设置站名，索引和距离
	index, percent, distance, coordIndex := GetClosest(data.Latitude, data.Longitude)
	if distance > 0.05 {
		// TODO: Record Fly
		_, err := h.redisConn.Do("HMSET", "BUS:"+busName,
			"Fly", true,
			"LastGPS", time.Now().Unix(),
			"GPS_OK", true,
		)
		if err != nil {
			panic(err)
		}
		h.touchBus(busName)
	}

	// 校巴在总站，删除校巴
	if index == 0 {
		_, err := h.redisConn.Do("DEL", "BUS:"+busName)
		if err != nil {
			panic(err)
		}
	}

	var stationIndex int
	var station string
	var direction bool

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

	if index == 1 && percent < 0 || index == 0 {
		percent = 0
	}

	if direction == true {
		percent *= -1
	}

	_, err := h.redisConn.Do("HMSET", "BUS:"+busName,
		"Name", busName,
		"CoordIndex", coordIndex,
		"StationIndex", stationIndex,
		"Station", station,
		"Percent", percent,
		"Direction", direction,
		"LastGPS", time.Now().Unix(),
		"GPS_OK", true,
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
		_, err := h.redisConn.Do("HSET", "BUS:"+busName, "GPS_OK", false)
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
