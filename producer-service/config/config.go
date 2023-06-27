package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	_ "github.com/joho/godotenv/autoload" // load envs
)

const (
	baseApiUrlVar      = "BASE_API_URL"
	kafkaBrokersVar    = "KAFKA_BROKERS"
	kafkaTopicVar      = "KAFKA_TOPIC"
	apiKeyVar          = "API_KEY"
	fetchIntervalVar   = "FETCH_INTERVAL"
	youTubeChannelsIds = "YOUTUBE_CHANNELS_IDS"
	workersCapacityVar = "WORKERS_CAPACITY"
)

func GetBaseApiUrl() (s string) {
	s = os.Getenv(baseApiUrlVar)
	if s == "" {
		log.Printf("%s var must not be empty \n", baseApiUrlVar)
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

func GetKafkaTopic() (s string) {
	s = os.Getenv(kafkaTopicVar)
	if s == "" {
		log.Printf("%s var must not be empty \n", kafkaTopicVar)
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

func GetFetchInterval() (n int) {
	msg := "fetch interval not properly configured. using default value"
	defaultValue := 60
	n = getIntVar(fetchIntervalVar, msg, defaultValue)
	return
}

func GetWorkersCapacity() (n int) {
	msg := "workers capacity not properly configured. using default value"
	defaultValue := 1
	n = getIntVar(workersCapacityVar, msg, defaultValue)
	return
}

func GetYouTubeChannelsIds() (ids []string) {
	return getListVar(youTubeChannelsIds)
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
