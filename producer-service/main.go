package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"tcc/app/producer-service/config"
	"tcc/app/producer-service/models"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

var (
	baseApiUrl    = config.GetBaseApiUrl()
	kafkaBrokers  = config.GetKafkaBrokers()
	kafkaTopic    = config.GetKafkaTopic()
	apiKey        = config.GetApiKey()
	fetchInterval = config.GetFetchInterval()
	ytChannelsIds = config.GetYouTubeChannelsIds()
)

func main() {
	// Create a channel to listen for interrupt signals e.g. (Ctrl+C) or interrupt <- os.Interrupt
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	producer, err := kafka.NewProducer(
		&kafka.ConfigMap{
			"bootstrap.servers":  kafkaBrokers,
			"acks":               "all",
			"retries":            5,
			"retry.backoff.ms":   100,
			"request.timeout.ms": 5000,
			"message.max.bytes":  2097164, // the same of kafka topics
			"client.id":          "producer-service",
			// min.insync.replicas
			// linger.ms   // batch by time
			// batch.size   // batch by size
		},
	)
	if err != nil {
		log.Println("Failed to create Kafka producer:", err)
	}
	defer producer.Close()

	dataChan := make(chan *models.CommentItemData)

	// go routines to retrieve data and send to kafka
	for _, channelId := range ytChannelsIds {
		go fetchCommentsData(dataChan, channelId)
	}

	go sendToKafka(producer, dataChan)

	// Wait for the interrupt signal
	<-signals

	close(dataChan)
	log.Println("Application interrupted. Exiting...")
}

func fetchCommentsData(dataChan chan<- *models.CommentItemData, channelId string) {
	// initializing latestCommentDate with ${fetchInterval} time ago
	latestCommentDate := time.Now().Add(time.Duration(-fetchInterval) * time.Second)
	var latestFetchDate time.Time

	for { // just a loop to fetch data every fetchInterval seconds
		nextPage := "initial page"
		latestFetchDate = latestCommentDate

		for nextPage != "" { // while have pagination
			commentsEndpoint := fmt.Sprintf("%s/commentThreads/", baseApiUrl)
			params := url.Values{}
			params.Set("part", "snippet,replies")
			params.Set("allThreadsRelatedToChannelId", channelId)
			params.Set("maxResults", "100")
			params.Set("order", "time")
			params.Set("textFormat", "plainText")
			params.Set("key", apiKey)
			if nextPage != "initial page" {
				log.Println("fetching next page of comments for channel id ", channelId)
				params.Set("pageToken", nextPage)
			}

			u, err := url.Parse(commentsEndpoint)
			if err != nil {
				log.Println("Error parsing URL:", err)
				continue
			}

			u.RawQuery = params.Encode()
			fullURL := u.String()

			resp, err := http.Get(fullURL)
			if err != nil {
				log.Println("Error while fetching data:", err)
				continue
			}

			log.Println("fetched data from api to channel id ", channelId)

			if resp.StatusCode != http.StatusOK {
				log.Println("API request returned non-OK status:", resp.StatusCode)
				resp.Body.Close()
				continue
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Println("Error while reading response body:", err)
				continue
			}

			resp.Body.Close()

			var comment models.CommentData
			err = json.Unmarshal(body, &comment)
			if err != nil {
				log.Println("Error while decoding json:", err)
			}

			nextPage = iterateCommentItems(dataChan, &comment, &latestCommentDate, &latestFetchDate)
		}

		// Sleep for 1 second before fetching the next data
		time.Sleep(time.Duration(fetchInterval) * time.Second)
	}
}

func iterateCommentItems(dataChan chan<- *models.CommentItemData, data *models.CommentData, latestCommentDate, latestFetchDate *time.Time) (nextPage string) {
	// Send the data to the channel
	for _, item := range data.Items {
		// verifying if is the last comment published
		isAfter := compareDates(item.Snippet.TopLevelComment.Snippet.PublishedAt, *latestCommentDate)
		if isAfter {
			*latestCommentDate = item.Snippet.TopLevelComment.Snippet.PublishedAt
		}

		// verifying if comment is new to last fetch request
		isAfter = compareDates(*latestFetchDate, item.Snippet.TopLevelComment.Snippet.PublishedAt)
		if isAfter {
			log.Println("Stopping reading comments from requests, all new comments were read, waiting for next fetch")
			return ""
		}

		dataChan <- item
	}

	return data.NextPageToken
}

func createMessage(data *models.CommentItemData) ([]byte, error) {
	message, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return message, nil
}

func sendToKafka(producer *kafka.Producer, dataChan <-chan *models.CommentItemData) {
	// Delivery report handler for produced messages
	go func() {
		for e := range producer.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					log.Println("Failed to deliver message to Kafka:", ev.TopicPartition.Error)
				} else {
					log.Println("Message sent to Kafka:", ev.TopicPartition)
				}
			}
		}
	}()

	for data := range dataChan {
		message, err := createMessage(data)
		if err != nil {
			log.Println("Failed to create message:", err)
			continue
		}

		log.Printf(
			"comment to send to kafka -> date: %s -> channel: %s -> comment: %s \n",
			data.Snippet.TopLevelComment.Snippet.PublishedAt,
			data.Snippet.ChannelID,
			data.Snippet.TopLevelComment.Snippet.TextDisplay,
		)

		err = producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &kafkaTopic, Partition: kafka.PartitionAny},
			Value:          message,
		}, nil)

		if err != nil {
			log.Println("Failed to produce message to Kafka:", err)
			continue
		}
	}

	// Wait for any outstanding messages to be delivered before exiting
	producer.Flush(15000)
}

func compareDates(date1, date2 time.Time) bool {
	if date1.After(date2) {
		return true
	} else {
		return false
	}
}
