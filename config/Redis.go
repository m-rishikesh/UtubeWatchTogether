package config

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	Ctx context.Context
	Rdb *redis.Client
)

type RoomState struct {
	RoomCode   string
	PlayerTime float64
	VideoURL   string
	IsPlaying  bool
}

func InitRedis() {
	redis_url := os.Getenv("REDIS_URL")
	Ctx = context.Background()
	Rdb = redis.NewClient(&redis.Options{
		Addr: redis_url,
	})
	res, err := Rdb.Ping(Ctx).Result()
	if err != nil {
		log.Println("Failed to Ping the Redis")
	}
	log.Println("Redis Connected: ", res)
}

// 3 function - GetRoom(),SaveRoom(),DeleteRoom()

func SaveRoom(room RoomState) error {
	key := "Room:" + room.RoomCode
	return Rdb.HSet(
		Ctx, key, map[string]any{
			"code":       room.RoomCode,
			"url":        room.VideoURL,
			"playertime": room.PlayerTime,
			"isPlaying":  room.IsPlaying,
		},
	).Err()
}

func GetRoom(code string) (*RoomState, error) {
	code = "Room:" + code
	data, err := Rdb.HGetAll(Ctx, code).Result()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	playerTime, err := strconv.ParseFloat(
		data["playertime"],
		64,
	)
	if err != nil {
		return nil, err
	}

	isPlaying, err := strconv.ParseBool(
		data["isPlaying"],
	)
	if err != nil {
		return nil, err
	}

	return &RoomState{
		RoomCode:   data["code"],
		VideoURL:   data["url"],
		IsPlaying:  isPlaying,
		PlayerTime: playerTime,
	}, nil
}

func DeleteRoom(code string) error {
	return Rdb.Del(
		Ctx,
		"Room:"+code,
	).Err()
}

func RoomExists(code string) (bool, error) {

	count, err := Rdb.Exists(
		Ctx,
		"Room:"+code,
	).Result()

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func SetRoomTTL(code string, ttl time.Duration) error {
	return Rdb.Expire(
		Ctx,
		"Room:"+code,
		ttl,
	).Err()
}

func RefreshRoomTTL(code string) error {
	return Rdb.Expire(
		Ctx,
		"Room:"+code,
		24*time.Hour,
	).Err()
}
