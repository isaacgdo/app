package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	_ "github.com/joho/godotenv/autoload" // load envs
)

const (
	kafkaBrokersVar     = "KAFKA_BROKERS"
	kafkaInputTopicsVar = "KAFKA_INPUT_TOPICS"
	kafkaOutputTopicVar = "KAFKA_OUTPUT_TOPIC"
	kafkaFailuresTopic  = "KAFKA_FAILURES_TOPIC"
	baseApiUrlVar       = "BASE_API_URL"
	apiKeyVar           = "API_KEY"
	msgProcessingDelay  = "MSG_PROCESSING_DELAY"
)

func GetBaseApiUrl() (s string) {
	s = os.Getenv(baseApiUrlVar)
	if s == "" {
		log.Printf("%s var must not be empty \n", baseApiUrlVar)
	}
	return
}

func GetApiKey() (s string) {
	s = os.Getenv(apiKeyVar)
	if s == "" {
		log.Printf("%s var must not be empty \n", apiKeyVar)
	}
	return
}

func GetKafkaBrokers() (s string) {
	s = os.Getenv(kafkaBrokersVar)
	if s == "" {
		log.Printf("%s var must not be empty \n", kafkaBrokersVar)
	}
	return
}

func GetKafkaInputTopics() (s []string) {
	return getListVar(kafkaInputTopicsVar)
}

func GetKafkaOutputTopic() (s string) {
	s = os.Getenv(kafkaOutputTopicVar)
	if s == "" {
		log.Printf("%s var must not be empty \n", kafkaOutputTopicVar)
	}
	return
}

func GetKafkaFailuresTopic() (s string) {
	s = os.Getenv(kafkaFailuresTopic)
	if s == "" {
		log.Printf("%s var must not be empty \n", kafkaFailuresTopic)
	}
	return
}

func getListVar(name string) []string {
	eValue := os.Getenv(name)
	if eValue == "" {
		return []string{}
	}
	return strings.Split(eValue, ",")
}

func getIntVar(name, msg string, defaultValue int) int {
	eValue := os.Getenv(name)
	iValue, err := strconv.Atoi(eValue)
	if err != nil {
		log.Println(msg)
		return defaultValue
	}
	return iValue
}

func GetMsgProcessingDelay() (n int) {
	msg := "processing delay not configured, processing messages without delay"
	defaultValue := 0
	n = getIntVar(msgProcessingDelay, msg, defaultValue)
	return
}
