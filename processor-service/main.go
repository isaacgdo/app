package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"tcc/app/processor-service/config"
	"tcc/app/processor-service/models"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/patrickmn/go-cache"
)

var (
	kafkaBrokers       = config.GetKafkaBrokers()
	kafkaInputTopics   = config.GetKafkaInputTopics()
	kafkaOutputTopic   = config.GetKafkaOutputTopic()
	kafkaFailuresTopic = config.GetKafkaFailuresTopic()
	baseApiUrl         = config.GetBaseApiUrl()
	apiKey             = config.GetApiKey()
	delay              = config.GetMsgProcessingDelay()
	workersCapacity    = config.GetWorkersCapacity()
	cacheTTL           = config.GetCacheTTL()
)

func main() {
	// Create a channel to listen for interrupt signals e.g. (Ctrl+C) or interrupt <- os.Interrupt
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Create a new Kafka consumer configuration
	consumerCfg := &kafka.ConfigMap{
		"bootstrap.servers":             kafkaBrokers,
		"group.id":                      "processor-service-group",
		"client.id":                     "processor-service",
		"enable.auto.commit":            "false",
		"partition.assignment.strategy": "roundrobin",
	}

	// Create a new Kafka consumer instance
	consumer, err := kafka.NewConsumer(consumerCfg)
	if err != nil {
		log.Fatalf("Failed to create Kafka consumer: %v", err)
	}

	// Create a new Kafka producer instance
	producer, err := kafka.NewProducer(
		&kafka.ConfigMap{
			"bootstrap.servers":  kafkaBrokers,
			"acks":               "all",
			"retries":            5,
			"retry.backoff.ms":   1000,
			"request.timeout.ms": 10000,
			"client.id":          "processor-service",
		},
	)
	if err != nil {
		log.Println("Failed to create Kafka producer:", err)
	}
	defer producer.Close()

	cache := config.NewCache()

	// Subscribe to the Kafka topic
	err = consumer.SubscribeTopics(kafkaInputTopics, nil)
	if err != nil {
		log.Fatalf("Failed to subscribe to Kafka topics: %v", err)
	}
	defer consumer.Close()

	// Create channels to receive consumed messages and commit then
	messageChan := make(chan *kafka.Message)
	commitCH := make(chan bool, 1)

	// create channel to send processed messages to be produce at output topic
	produceChan := make(chan *models.CommentOutputData)

	// Start a goroutine to consume messages and a go routine to send messages to kafka output topic
	go consume(consumer, messageChan, commitCH)

	for worker := 1; worker <= workersCapacity; worker++ {
		// start a goroutine to process the consumed messages
		go processMessage(messageChan, commitCH, produceChan, producer, cache)

		// start a go routine to send messages to kafka output topic
		go sendOutputMessage(producer, produceChan)

		// Delivery report handler for produced messages
		go deliveryReport(producer)
	}

	// Wait for the interrupt signal
	<-signals
	log.Println("Application interrupted. Exiting...")

	// Wait for any outstanding messages to be delivered before exiting
	producer.Flush(3000)
}

func processMessage(messageChan chan *kafka.Message, commitCH chan<- bool,
	produceChan chan *models.CommentOutputData, producer *kafka.Producer, cache *cache.Cache) {
	for m := range messageChan {
		// Process the consumed message
		var comment models.CommentItemData
		err := json.Unmarshal(m.Value, &comment)
		if err != nil {
			log.Printf("Failed to parse message data: %v \n", err)
			commitCH <- true

			err = sendToKafka(producer, m.Value, kafkaFailuresTopic)
			if err != nil {
				log.Printf("Failed to send message to kafka failures topic: %v \n", err)
			}
			continue
		}
		log.Printf("Received message: %s  \n", comment.Snippet.TopLevelComment.Snippet.TextDisplay)

		msgs, errM := createOutputMessages(&comment, cache)
		if errM != nil {
			log.Printf("Failed to create output message: %v \n", err)
			commitCH <- true

			err = sendToKafka(producer, m.Value, kafkaFailuresTopic)
			if err != nil {
				log.Printf("Failed to send message to kafka failures topic: %v \n", err)
			}
			continue
		}

		for _, msg := range msgs {
			produceChan <- msg
		}

		commitCH <- true

		if delay > 0 {
			time.Sleep(time.Duration(delay) * time.Second)
		}
	}
}

// consume consumes messages from kafka and wait for a command to commit the message
func consume(consumer *kafka.Consumer, messageChan chan *kafka.Message, commitCH <-chan bool) {
	defer close(messageChan)
	for {
		m, err := consumer.ReadMessage(-1)
		if err != nil {
			log.Printf("Consumer error: %v (%v)\n", err, m)
			continue
		}

		messageChan <- m

		commit := <-commitCH
		if !commit {
			log.Println("command to avoid commit received")
			continue
		}

		_, err = consumer.CommitMessage(m)
		if err != nil {
			log.Println("could not commit message on kafka", err)
			continue
		}
		log.Println("kafka commit succeeded")
	}
}

func createOutputMessages(msg *models.CommentItemData, cache *cache.Cache) ([]*models.CommentOutputData, error) {
	var videoRawData []byte

	v, found := cache.Get(msg.Snippet.VideoID)
	if found {
		log.Println("Video data found at cache, using it:", msg.Snippet.VideoID)
		videoRawData = v.([]byte)
	} else {
		log.Println("Video not found at cache, starting fetching from API")
		videoEndpoint := fmt.Sprintf("%s/videos/", baseApiUrl)
		params := url.Values{}
		params.Set("part", "id,snippet,statistics,topicDetails")
		params.Set("id", msg.Snippet.VideoID)
		params.Set("key", apiKey)

		u, err := url.Parse(videoEndpoint)
		if err != nil {
			log.Println("Error parsing API URL:", err)
			return nil, err
		}

		u.RawQuery = params.Encode()
		fullURL := u.String()

		resp, err := http.Get(fullURL)
		if err != nil {
			log.Println("Error while fetching data form API:", err)
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Println("API request returned non-OK status:", resp.StatusCode)
			return nil, errors.New("API request returned non-OK status")
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println("Error while reading response body:", err)
			return nil, err
		}
		log.Println("Video data retrieved from API")

		var videoList models.VideoListData
		err = json.Unmarshal(body, &videoList)
		if err != nil {
			log.Println("Error while unmarshalling json:", err)
			return nil, err
		}

		if len(videoList.Items) == 0 {
			log.Println("Video not found")
			return []*models.CommentOutputData{}, nil
		}

		videoRawData, err = json.Marshal(videoList.Items[0])
		if err != nil {
			log.Println("Error while marshalling video data:", err)
			return nil, err
		}

		cache.Set(msg.Snippet.VideoID, videoRawData, time.Duration(cacheTTL)*time.Second)
		log.Println("New video cached:", msg.Snippet.VideoID)
	}

	var video models.VideoItemData
	err := json.Unmarshal(videoRawData, &video)
	if err != nil {
		log.Println("Error while unmarshalling json:", err)
		return nil, err
	}

	result, err := buildOutputResponse(msg, &video)
	if err != nil {
		log.Println("Error while building output response:", err)
		return nil, err
	}
	return result, nil
}

func buildOutputResponse(comment *models.CommentItemData, video *models.VideoItemData) ([]*models.CommentOutputData, error) {
	var commentsList = make([]*models.CommentOutputData, 0, len(comment.Replies.Comments)+1)

	localTimezone, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		log.Println("Error while loading timezone", err)
		return nil, err
	}

	var videoViewCount int
	if video.Statistics.LikeCount != "" {
		videoViewCount, err = strconv.Atoi(video.Statistics.ViewCount)
		if err != nil {
			log.Println("Error while converting videoViewCount string", err)
			return nil, err
		}
	}

	var videoLikeCount int
	if video.Statistics.LikeCount != "" {
		videoLikeCount, err = strconv.Atoi(video.Statistics.LikeCount)
		if err != nil {
			log.Println("Error while converting videoLikeCount string", err)
			return nil, err
		}
	}

	var videoCommentCount int
	if video.Statistics.CommentCount != "" {
		videoCommentCount, err = strconv.Atoi(video.Statistics.CommentCount)
		if err != nil {
			log.Println("Error while converting videoCommentCount string", err)
			return nil, err
		}
	}

	mainComment := &models.CommentOutputData{
		ID:                comment.ID,
		ChannelID:         comment.Snippet.ChannelID,
		ChannelTitle:      video.Snippet.ChannelTitle,
		VideoID:           comment.Snippet.VideoID,
		TextDisplay:       comment.Snippet.TopLevelComment.Snippet.TextDisplay,
		TextOriginal:      comment.Snippet.TopLevelComment.Snippet.TextOriginal,
		AuthorDisplayName: comment.Snippet.TopLevelComment.Snippet.AuthorDisplayName,
		LikeCount:         comment.Snippet.TopLevelComment.Snippet.LikeCount,
		PublishedAt:       comment.Snippet.TopLevelComment.Snippet.PublishedAt.In(localTimezone).Format("2006-01-02T15:04:05Z"),
		TotalReplyCount:   comment.Snippet.TotalReplyCount,
		ParentID:          "",
		VideoTitle:        video.Snippet.Title,
		VideoViewCount:    videoViewCount,
		VideoLikeCount:    videoLikeCount,
		VideoCommentCount: videoCommentCount,
		VideoTags:         video.Snippet.Tags,
	}

	commentsList = append(commentsList, mainComment)

	for _, reply := range comment.Replies.Comments {
		r := &models.CommentOutputData{
			ID:                reply.ID,
			ChannelID:         reply.Snippet.ChannelID,
			ChannelTitle:      video.Snippet.ChannelTitle,
			VideoID:           reply.Snippet.VideoID,
			TextDisplay:       reply.Snippet.TextDisplay,
			TextOriginal:      reply.Snippet.TextOriginal,
			AuthorDisplayName: reply.Snippet.AuthorDisplayName,
			LikeCount:         reply.Snippet.LikeCount,
			PublishedAt:       reply.Snippet.PublishedAt.In(localTimezone).Format("2006-01-02T15:04:05Z"),
			TotalReplyCount:   0,
			ParentID:          reply.Snippet.ParentID,
			VideoTitle:        video.Snippet.Title,
			VideoViewCount:    videoViewCount,
			VideoLikeCount:    videoLikeCount,
			VideoCommentCount: videoCommentCount,
			VideoTags:         video.Snippet.Tags,
		}
		commentsList = append(commentsList, r)
	}
	return commentsList, nil
}

func sendOutputMessage(producer *kafka.Producer, produceChan <-chan *models.CommentOutputData) {
	for data := range produceChan {
		message, err := json.Marshal(data)
		if err != nil {
			log.Println("Failed to create message:", err)
		}

		err = sendToKafka(producer, message, kafkaOutputTopic)
		if err != nil {
			log.Println("Failed to send message to kafka output topic:", err)
		}
	}
}

// Delivery report handler for produced messages
func deliveryReport(producer *kafka.Producer) {
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
}

func sendToKafka(producer *kafka.Producer, message []byte, topic string) error {
	err := producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          message,
	}, nil)

	if err != nil {
		log.Println("Failed to produce message to Kafka:", err)
		return err
	}

	return nil
}
