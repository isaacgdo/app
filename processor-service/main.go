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
	"strconv"
	"syscall"
	"tcc/app/processor-service/config"
	"tcc/app/processor-service/models"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

var (
	kafkaBrokers       = config.GetKafkaBrokers()
	kafkaInputTopics   = config.GetKafkaInputTopics()
	kafkaOutputTopic   = config.GetKafkaOutputTopic()
	kafkaFailuresTopic = config.GetKafkaFailuresTopic()
	baseApiUrl         = config.GetBaseApiUrl()
	apiKey             = config.GetApiKey()
	delay              = config.GetMsgProcessingDelay()
)

func main() {
	// Create a channel to listen for interrupt signals e.g. (Ctrl+C) or interrupt <- os.Interrupt
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Create a new Kafka consumer configuration
	consumerCfg := &kafka.ConfigMap{
		"bootstrap.servers":             kafkaBrokers,
		"group.id":                      "processor-service-group",
		"auto.offset.reset":             "earliest",
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
			"retry.backoff.ms":   100,
			"request.timeout.ms": 5000,
			"message.max.bytes":  2097164, // the same of kafka topics
			"client.id":          "processor-service",
		},
	)
	if err != nil {
		log.Println("Failed to create Kafka producer:", err)
	}
	defer producer.Close()

	// Subscribe to the Kafka topic
	err = consumer.SubscribeTopics(kafkaInputTopics, nil)
	if err != nil {
		log.Fatalf("Failed to subscribe to Kafka topics: %v", err)
	}
	defer consumer.Close()

	// Create channels to receive consumed messages and commit then
	messageChan := make(chan *kafka.Message)
	commitCH := make(chan bool, 1)

	// create to send messages processed to produce at output topic
	produceChan := make(chan *models.CommentOutputData)

	// Start a goroutine to consume messages and a go routine to send messages to kafka output topic
	go consume(consumer, messageChan, commitCH)
	go sendOutputMessage(producer, produceChan, kafkaOutputTopic)

	// start a goroutine to process the consumed messages
	go func() {
		for m := range messageChan {
			// Process the consumed message
			var comment models.CommentItemData
			err = json.Unmarshal(m.Value, &comment)
			if err != nil {
				log.Printf("Failed to parse message data: %v \n", err)
				commitCH <- false
				err = sendToKafka(producer, m.Value, kafkaFailuresTopic)
				if err != nil {
					log.Printf("Failed to send message to kafka failures topic: %v \n", err)
				}
				continue
			}
			log.Printf("Received message: %s  \n", comment.Snippet.TopLevelComment.Snippet.TextDisplay)

			msgs, errM := createOutputMessages(&comment)
			if errM != nil {
				err = sendToKafka(producer, m.Value, kafkaFailuresTopic)
				if err != nil {
					log.Printf("Failed to send message to kafka failures topic: %v \n", err)
				}
			}

			for _, msg := range msgs {
				produceChan <- msg
			}

			commitCH <- true

			if delay > 0 {
				time.Sleep(time.Duration(delay) * time.Second)
			}
		}
	}()

	// Wait for the interrupt signal
	<-signals

	// close channels?
	log.Println("Application interrupted. Exiting...")
}

// consume consumes messages from kafka and wait for a command to commit the message
func consume(consumer *kafka.Consumer, messageChan chan *kafka.Message, commitCH <-chan bool) {
	defer close(messageChan)
	for {
		m, err := consumer.ReadMessage(-1)
		if err != nil {
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

func createOutputMessages(msg *models.CommentItemData) ([]*models.CommentOutputData, error) {
	videoEndpoint := fmt.Sprintf("%s/videos/", baseApiUrl)
	params := url.Values{}
	params.Set("part", "id,snippet,statistics,topicDetails")
	params.Set("id", msg.Snippet.VideoID)
	params.Set("key", apiKey)

	u, err := url.Parse(videoEndpoint)
	if err != nil {
		log.Println("Error parsing URL:", err)
		return nil, err
	}

	u.RawQuery = params.Encode()
	fullURL := u.String()

	resp, err := http.Get(fullURL)
	if err != nil {
		log.Println("Error while fetching data:", err)
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		log.Println("API request returned non-OK status:", resp.StatusCode)
		resp.Body.Close()
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error while reading response body:", err)
		return nil, err
	}
	log.Println("Video data retrieved")

	var video models.VideoListData
	err = json.Unmarshal(body, &video)
	if err != nil {
		log.Println("Error while unmarshalling json:", err)
		return nil, err
	}

	if len(video.Items) == 0 {
		log.Println("Video not found")
		return []*models.CommentOutputData{}, nil
	}
	return buildOutputResponse(msg, video.Items[0]), nil
}

func buildOutputResponse(comment *models.CommentItemData, video *models.VideoItemData) []*models.CommentOutputData {
	var commentsList = make([]*models.CommentOutputData, 0, len(comment.Replies.Comments)+1)

	localTimezone, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		log.Fatalln("Error while loading timezone", err)
	}

	videoViewCount, err := strconv.Atoi(video.Statistics.ViewCount)
	if err != nil {
		log.Fatalln("Error while converting string", err)
	}

	videoLikeCount, err := strconv.Atoi(video.Statistics.LikeCount)
	if err != nil {
		log.Fatalln("Error while converting string", err)
	}

	videoCommentCount, err := strconv.Atoi(video.Statistics.CommentCount)
	if err != nil {
		log.Fatalln("Error while converting string", err)
	}

	mainComment := &models.CommentOutputData{
		ID:                comment.ID,
		ChannelID:         comment.Snippet.ChannelID,
		ChannelTitle:      video.Snippet.ChannelTitle,
		VideoID:           comment.Snippet.VideoID,
		TextDisplay:       comment.Snippet.TopLevelComment.Snippet.TextDisplay,
		TextOriginal:      comment.Snippet.TopLevelComment.Snippet.TextOriginal,
		AuthorDisplayName: comment.Snippet.TopLevelComment.Snippet.AuthorDisplayName,
		CanRate:           comment.Snippet.TopLevelComment.Snippet.CanRate,
		ViewerRating:      comment.Snippet.TopLevelComment.Snippet.ViewerRating,
		LikeCount:         comment.Snippet.TopLevelComment.Snippet.LikeCount,
		PublishedAt:       comment.Snippet.TopLevelComment.Snippet.PublishedAt.In(localTimezone),
		CanReply:          comment.Snippet.CanReply,
		TotalReplyCount:   comment.Snippet.TotalReplyCount,
		ParentID:          "",
		VideoPublishedAt:  video.Snippet.PublishedAt.In(localTimezone),
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
			CanRate:           reply.Snippet.CanRate,
			ViewerRating:      reply.Snippet.ViewerRating,
			LikeCount:         reply.Snippet.LikeCount,
			PublishedAt:       reply.Snippet.PublishedAt.In(localTimezone),
			CanReply:          false,
			TotalReplyCount:   0,
			ParentID:          reply.Snippet.ParentID,
			VideoPublishedAt:  video.Snippet.PublishedAt.In(localTimezone),
			VideoTitle:        video.Snippet.Title,
			VideoViewCount:    videoViewCount,
			VideoLikeCount:    videoLikeCount,
			VideoCommentCount: videoCommentCount,
			VideoTags:         video.Snippet.Tags,
		}
		commentsList = append(commentsList, r)
	}
	return commentsList
}

func sendOutputMessage(producer *kafka.Producer, produceChan <-chan *models.CommentOutputData, topic string) {
	for data := range produceChan {
		message, err := json.Marshal(data)
		if err != nil {
			log.Println("Failed to create message:", err)
		}

		err = sendToKafka(producer, message, topic)
		if err != nil {
			log.Println("Failed to send message to kafka output topic:", err)
		}
	}

	// Wait for any outstanding messages to be delivered before exiting
	producer.Flush(15000)
}

func sendToKafka(producer *kafka.Producer, message []byte, topic string) error {
	deliveryChan := make(chan kafka.Event)

	err := producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          message,
	}, deliveryChan)

	if err != nil {
		log.Println("Failed to produce message to Kafka:", err)
		return err
	}

	e := <-deliveryChan
	m := e.(*kafka.Message)

	if m.TopicPartition.Error != nil {
		log.Println("Failed to deliver message to Kafka:", m.TopicPartition.Error)
		return m.TopicPartition.Error
	}
	log.Println("Message sent to Kafka:", m.TopicPartition)

	return nil
}