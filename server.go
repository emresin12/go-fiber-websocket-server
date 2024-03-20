package main

import (
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"log"
	"sync"
	"time"
)

var duration = 10 * time.Second
var n_conn = 20

type ThroughputData struct {
	sync.Mutex
	totalThroughput float64
	totalBytesSent  int
	clientCount     int
}

func send_data(c *websocket.Conn, throughputData *ThroughputData, shutdown chan int) {
	defer c.Close()

	message := make([]byte, 512)
	totalBytes := 0
	timer := time.NewTimer(duration)

loop:
	for {
		select {
		case <-timer.C:
			break loop
		default:
			err := c.WriteMessage(websocket.TextMessage, message)

			if err != nil {
				log.Println("write:", err)
				break
			}
			totalBytes += len(message)
		}
	}

	throughput := float64(totalBytes) / 1024 / duration.Seconds()

	// Protect shared data with mutex
	terminate := false

	throughputData.Lock()
	throughputData.totalThroughput += throughput
	throughputData.totalBytesSent += totalBytes
	throughputData.clientCount++
	if throughputData.clientCount == n_conn {
		terminate = true
	}
	throughputData.Unlock()

	if terminate == true { // to prevent deadlock
		close(shutdown)
	}

	log.Printf("client throughput: %f KB/s", throughput)
}

func echo(c websocket.Conn) {
	defer c.Close()
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			log.Println("error while reading ", err)
			break
		}
		err = c.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Println("error while writing ", err)
			break
		}
	}
}

func main() {
	throughputData := ThroughputData{}
	shutdown := make(chan int)

	app := fiber.New()

	app.Use("/", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/", websocket.New(func(c *websocket.Conn) {
		send_data(c, &throughputData, shutdown)
		//echo(*c)
	}))

	go func() {
		err := app.Listen(":3000")
		if err != nil {
			log.Fatal(err)
		}
	}()

	<-shutdown

	averageDataSent := throughputData.totalBytesSent / throughputData.clientCount
	averageThroughput := throughputData.totalThroughput / float64(throughputData.clientCount)
	log.Printf("total throughput is %f KB/s", float64(throughputData.totalBytesSent)/1024/duration.Seconds())
	log.Printf("Total data sent is %f KB", float64(throughputData.totalBytesSent)/1024)
	log.Printf("Average throughput is %f KB/s", averageThroughput)
	log.Printf("Average byte sent is %f KB", float64(averageDataSent)/1024)

}
