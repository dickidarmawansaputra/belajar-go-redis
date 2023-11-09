package test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

var client = redis.NewClient(&redis.Options{
	Addr: "localhost:6379",
	DB:   0,
})

func TestConnection(t *testing.T) {
	assert.NotNil(t, client)
	fmt.Println(client)
	// err := client.Close()
	// assert.Nil(t, err)
}

var ctx = context.Background()

func TestPing(t *testing.T) {
	result, err := client.Ping(ctx).Result()
	assert.Nil(t, err)
	assert.Equal(t, "PONG", result)
}

func TestString(t *testing.T) {
	client.SetEx(ctx, "name", "Dicki Darmawan Saputra", 3*time.Second)
	result, err := client.Get(ctx, "name").Result()
	assert.Nil(t, err)
	assert.Equal(t, "Dicki Darmawan Saputra", result)

	time.Sleep(5 * time.Second)

	result, err = client.Get(ctx, "name").Result()
	assert.NotNil(t, err) // harusnya error karna datanya sudah tidak ada

	client.Del(ctx, "name")
}

func TestList(t *testing.T) {
	client.RPush(ctx, "names", "Dicki")
	client.RPush(ctx, "names", "Darmawan")
	client.RPush(ctx, "names", "Saputra")

	assert.Equal(t, "Dicki", client.LPop(ctx, "names").Val())
	assert.Equal(t, "Darmawan", client.LPop(ctx, "names").Val())
	assert.Equal(t, "Saputra", client.LPop(ctx, "names").Val())

	client.Del(ctx, "names")
}

func TestSet(t *testing.T) {
	client.SAdd(ctx, "students", "Dicki")
	client.SAdd(ctx, "students", "Dicki")
	client.SAdd(ctx, "students", "Darmawan")
	client.SAdd(ctx, "students", "Darmawan")
	client.SAdd(ctx, "students", "Saputra")
	client.SAdd(ctx, "students", "Saputra")

	assert.Equal(t, int64(3), client.SCard(ctx, "students").Val())
	assert.Equal(t, []string{"Dicki", "Darmawan", "Saputra"}, client.SMembers(ctx, "students").Val())

	client.Del(ctx, "students")
}

func TestSortedSet(t *testing.T) {
	client.ZAdd(ctx, "scores", redis.Z{Score: 100, Member: "Dicki"})
	client.ZAdd(ctx, "scores", redis.Z{Score: 70, Member: "Darmawan"})
	client.ZAdd(ctx, "scores", redis.Z{Score: 80, Member: "Saputra"})

	assert.Equal(t, []string{"Darmawan", "Saputra", "Dicki"}, client.ZRange(ctx, "scores", 0, 2).Val())
	assert.Equal(t, "Dicki", client.ZPopMax(ctx, "scores").Val()[0].Member)
	assert.Equal(t, "Saputra", client.ZPopMax(ctx, "scores").Val()[0].Member)
	assert.Equal(t, "Darmawan", client.ZPopMax(ctx, "scores").Val()[0].Member)

	client.Del(ctx, "scores")
}

func TestHash(t *testing.T) {
	client.HSet(ctx, "user:1", "id", "1")
	client.HSet(ctx, "user:1", "name", "Dicki")
	client.HSet(ctx, "user:1", "email", "dicki@mail.com")

	user := client.HGetAll(ctx, "user:1").Val()
	assert.Equal(t, "1", user["id"])
	assert.Equal(t, "Dicki", user["name"])
	assert.Equal(t, "dicki@mail.com", user["email"])

	client.Del(ctx, "user:1")
}

func TestGeoPoint(t *testing.T) {
	client.GeoAdd(ctx, "sellers", &redis.GeoLocation{
		Name:      "Toko A",
		Longitude: 106.818489,
		Latitude:  -6.178966,
	})
	client.GeoAdd(ctx, "sellers", &redis.GeoLocation{
		Name:      "Toko B",
		Longitude: 106.821568,
		Latitude:  -6.180662,
	})

	distance := client.GeoDist(ctx, "sellers", "Toko A", "Toko B", "km").Val()
	assert.Equal(t, 0.3892, distance)

	sellers := client.GeoSearch(ctx, "sellers", &redis.GeoSearchQuery{
		Longitude:  106.819143,
		Latitude:   -6.180182,
		Radius:     5,
		RadiusUnit: "km",
	}).Val()

	assert.Equal(t, []string{"Toko A", "Toko B"}, sellers)

	client.Del(ctx, "sellers")
}

func TestHyperLogLog(t *testing.T) {
	client.PFAdd(ctx, "visitors", "Dicki", "Darmawan", "Saputra")
	client.PFAdd(ctx, "visitors", "Dicki", "Budi", "Joko")
	client.PFAdd(ctx, "visitors", "Rully", "Budi", "Joko")

	total := client.PFCount(ctx, "visitors").Val()
	assert.Equal(t, int64(6), total)

	client.Del(ctx, "visitors")
}

func TestPipeline(t *testing.T) {
	_, err := client.Pipelined(ctx, func(pipeliner redis.Pipeliner) error {
		pipeliner.SetEx(ctx, "name", "Dicki", 5*time.Second)
		pipeliner.SetEx(ctx, "country", "Indonesia", 5*time.Second)
		return nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "Dicki", client.Get(ctx, "name").Val())
	assert.Equal(t, "Indonesia", client.Get(ctx, "country").Val())

	client.Del(ctx, "name")
	client.Del(ctx, "country")
}

func TestTransaction(t *testing.T) {
	_, err := client.TxPipelined(ctx, func(pipeliner redis.Pipeliner) error {
		pipeliner.SetEx(ctx, "name", "Dicki", 5*time.Second)
		pipeliner.SetEx(ctx, "address", "Pontianak", 5*time.Second)
		return nil
	})

	assert.Equal(t, "Dicki", client.Get(ctx, "name").Val())
	assert.Equal(t, "Pontianak", client.Get(ctx, "address").Val())

	client.Del(ctx, "name")
	client.Del(ctx, "address")
}

func TestPublishStream(t *testing.T) {
	for i := 0; i < 10; i++ {
		err := client.XAdd(ctx, &redis.XAddArgs{
			Stream: "members",
			Values: map[string]interface{}{
				"name":    "Dicki",
				"country": "Indonesia",
			},
		}).Err()
		assert.Nil(t, err)
	}
}

func TestCreateConsumerGroup(t *testing.T) {
	client.XGroupCreate(ctx, "members", "group-1", "0")
	client.XGroupCreateConsumer(ctx, "members", "group-1", "consumer-1")
	client.XGroupCreateConsumer(ctx, "members", "group-1", "consumer-2")
}

func TestConsumeStream(t *testing.T) {
	result := client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    "group-1",
		Consumer: "consumer-1",
		// > yg terakhir / belum dibaca
		Streams: []string{"members", ">"},
		// jumlah yg mau dibaca
		Count: 2,
		Block: 5 * time.Second,
	}).Val()

	for _, stream := range result {
		for _, message := range stream.Messages {
			fmt.Println(message.ID)
			fmt.Println(message.Values)
		}
	}
}

// tipe sederhana dari Stream
// ketika tidak ada subsciber ketika kirim data otomatis hilang
// saat buat aplikasi jalankan gunakan goroutine agar tidak memblok program
func TestSubscribePubSub(t *testing.T) {
	subsciber := client.Subscribe(ctx, "channel-1")
	defer subsciber.Close()
	for i := 0; i < 10; i++ {
		message, err := subsciber.ReceiveMessage(ctx)
		assert.Nil(t, err)
		fmt.Println(message.Payload)
	}
}

func TestPublisherPubSub(t *testing.T) {
	for i := 0; i < 10; i++ {
		err := client.Publish(ctx, "channel-1", "Hello "+strconv.Itoa(i)).Err()
		assert.Nil(t, err)
	}
}
