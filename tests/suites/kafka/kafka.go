package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go/modules/kafka"

	"gitlab.ozon.dev/safariproxd/homework/internal/domain"
	kf "gitlab.ozon.dev/safariproxd/homework/internal/infra/kafka"
)

type KafkaIntegrationSuite struct {
	suite.Suite
	ctx            context.Context
	kafkaContainer *kafka.KafkaContainer
	brokers        []string
}

func (s *KafkaIntegrationSuite) SetupSuite() {
	s.ctx = context.Background()

	kafkaContainer, err := kafka.Run(s.ctx,
		"confluentinc/cp-kafka:7.5.0",
		kafka.WithClusterID("test-cluster"),
	)
	require.NoError(s.T(), err)
	s.kafkaContainer = kafkaContainer

	brokers, err := kafkaContainer.Brokers(s.ctx)
	require.NoError(s.T(), err)
	s.brokers = brokers

	config := sarama.NewConfig()
	config.Version = sarama.V3_0_0_0
	for i := 0; i < 10; i++ {
		client, err := sarama.NewClient(s.brokers, config)
		if err == nil {
			client.Close()
			break
		}
		s.T().Logf("Waiting for Kafka to be ready: %v", err)
		time.Sleep(2 * time.Second)
	}
}

func (s *KafkaIntegrationSuite) TearDownSuite() {
	if s.kafkaContainer != nil {
		_ = s.kafkaContainer.Terminate(s.ctx)
	}
}

func (s *KafkaIntegrationSuite) createTopic(topic string) {
	sanitizedTopic := strings.ReplaceAll(topic, "/", "_")
	s.T().Logf("Creating topic: %s", sanitizedTopic)

	config := sarama.NewConfig()
	config.Version = sarama.V3_0_0_0

	admin, err := sarama.NewClusterAdmin(s.brokers, config)
	require.NoError(s.T(), err)
	defer admin.Close()

	topicDetail := &sarama.TopicDetail{
		NumPartitions:     1,
		ReplicationFactor: 1,
	}

	err = admin.CreateTopic(sanitizedTopic, topicDetail, false)
	require.NoError(s.T(), err)

	time.Sleep(5 * time.Second)

	topics, err := admin.ListTopics()
	require.NoError(s.T(), err)
	_, exists := topics[sanitizedTopic]
	assert.True(s.T(), exists, "Topic %s was not created", sanitizedTopic)
}

func (s *KafkaIntegrationSuite) Test_ProducerConsumer_SingleMessage() {
	ctx := s.ctx
	topic := "test-events-single-" + strings.ReplaceAll(s.T().Name(), "/", "_")

	s.createTopic(topic)

	consumerCfg := kf.KafkaConsumerConfig{
		Brokers:         s.brokers,
		Topic:           topic,
		ConsumerGroup:   "test-group-single-" + strings.ReplaceAll(s.T().Name(), "/", "_"),
		AutoOffsetReset: "earliest",
	}
	consumer, err := kf.NewKafkaConsumer(consumerCfg)
	require.NoError(s.T(), err)
	defer consumer.Close()

	receivedEvents := make(chan domain.Event, 1)
	handler := &testMessageHandler{
		receivedEvents: receivedEvents,
	}

	consumeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	go func() {
		s.T().Logf("Starting consumer for topic: %s, group: %s", topic, consumerCfg.ConsumerGroup)
		err := consumer.Consume(consumeCtx, handler)
		if err != nil {
			s.T().Logf("Consumer error: %v", err)
		}
	}()

	time.Sleep(5 * time.Second)

	producer, err := kf.NewKafkaProducer(s.brokers, topic)
	require.NoError(s.T(), err)
	defer producer.Close()

	testEvent := domain.Event{
		EventID:   "test-123",
		EventType: domain.EventTypeOrderAccepted,
		Timestamp: time.Now(),
		Actor: domain.Actor{
			Type: domain.ActorTypeCourier,
			ID:   42,
		},
		Order: domain.OrderInfo{
			ID:     999,
			UserID: 777,
			Status: "accepted",
		},
		Source: "integration-test",
	}

	message, err := json.Marshal(testEvent)
	require.NoError(s.T(), err)

	s.T().Logf("Sending message: %s", message)
	err = producer.Send(ctx, message)
	require.NoError(s.T(), err)

	for i := 0; i < 3; i++ {
		select {
		case receivedEvent := <-receivedEvents:
			s.T().Logf("Received event: %+v", receivedEvent)
			assert.Equal(s.T(), testEvent.EventID, receivedEvent.EventID)
			assert.Equal(s.T(), testEvent.EventType, receivedEvent.EventType)
			assert.Equal(s.T(), testEvent.Actor.ID, receivedEvent.Actor.ID)
			assert.Equal(s.T(), testEvent.Order.ID, receivedEvent.Order.ID)
			return
		case <-time.After(5 * time.Second):
			s.T().Logf("Attempt %d: Timeout waiting for message", i+1)
			if i == 2 {
				s.T().Fatal("Timeout waiting for message after retries")
			}
		}
	}
}

func (s *KafkaIntegrationSuite) Test_ProducerConsumer_MultipleMessages() {
	ctx := s.ctx
	topic := "test-events-multiple-" + strings.ReplaceAll(s.T().Name(), "/", "_")

	s.createTopic(topic)

	consumerCfg := kf.KafkaConsumerConfig{
		Brokers:         s.brokers,
		Topic:           topic,
		ConsumerGroup:   "test-group-multiple-" + strings.ReplaceAll(s.T().Name(), "/", "_"),
		AutoOffsetReset: "earliest",
	}
	consumer, err := kf.NewKafkaConsumer(consumerCfg)
	require.NoError(s.T(), err)

	receivedEvents := make(chan domain.Event, 10)
	handler := &testMessageHandler{
		receivedEvents: receivedEvents,
	}

	consumeCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	go func() {
		s.T().Logf("Starting consumer for topic: %s, group: %s", topic, consumerCfg.ConsumerGroup)
		err := consumer.Consume(consumeCtx, handler)
		if err != nil {
			s.T().Logf("Consumer error: %v", err)
		}
	}()

	time.Sleep(5 * time.Second)

	producer, err := kf.NewKafkaProducer(s.brokers, topic)
	require.NoError(s.T(), err)
	defer producer.Close()

	messageCount := 10
	sentEvents := make([]domain.Event, messageCount)

	for i := 0; i < messageCount; i++ {
		event := domain.Event{
			EventID:   fmt.Sprintf("test-%d", i),
			EventType: domain.EventTypeOrderIssued,
			Timestamp: time.Now(),
			Actor: domain.Actor{
				Type: domain.ActorTypeClient,
				ID:   uint64(100 + i),
			},
			Order: domain.OrderInfo{
				ID:     uint64(1000 + i),
				UserID: uint64(2000 + i),
				Status: "issued",
			},
			Source: "integration-test-multiple",
		}
		sentEvents[i] = event

		message, err := json.Marshal(event)
		require.NoError(s.T(), err)

		s.T().Logf("Sending message %d: %s", i, message)
		err = producer.Send(ctx, message)
		require.NoError(s.T(), err)
		time.Sleep(200 * time.Millisecond)
	}

	receivedMap := make(map[string]domain.Event)
	timeout := time.After(30 * time.Second)
	for len(receivedMap) < messageCount {
		select {
		case event := <-receivedEvents:
			s.T().Logf("Received event: %+v", event)
			receivedMap[event.EventID] = event
		case <-timeout:
			s.T().Fatalf("Timeout: received only %d/%d messages", len(receivedMap), messageCount)
		}
	}

	consumer.Close()

	assert.Len(s.T(), receivedMap, messageCount, "Expected to receive %d messages", messageCount)
	for _, sentEvent := range sentEvents {
		received, ok := receivedMap[sentEvent.EventID]
		assert.True(s.T(), ok, "Event %s not received", sentEvent.EventID)
		assert.Equal(s.T(), sentEvent.EventType, received.EventType, "EventType mismatch for %s", sentEvent.EventID)
		assert.Equal(s.T(), sentEvent.Order.ID, received.Order.ID, "Order.ID mismatch for %s", sentEvent.EventID)
	}
}

func (s *KafkaIntegrationSuite) Test_ConsumerGroup_Rebalance() {
	ctx := s.ctx
	topic := "test-events-rebalance-" + strings.ReplaceAll(s.T().Name(), "/", "_")

	s.createTopic(topic)
	consumerCfg := kf.KafkaConsumerConfig{
		Brokers:         s.brokers,
		Topic:           topic,
		ConsumerGroup:   "test-group-rebalance-" + strings.ReplaceAll(s.T().Name(), "/", "_"),
		AutoOffsetReset: "earliest",
	}

	consumer1, err := kf.NewKafkaConsumer(consumerCfg)
	require.NoError(s.T(), err)

	consumer2, err := kf.NewKafkaConsumer(consumerCfg)
	require.NoError(s.T(), err)

	var wg sync.WaitGroup
	receivedEvents := make(chan string, 20)

	handler := &countingHandler{
		receivedIDs: receivedEvents,
	}

	consumeCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	wg.Add(2)
	go func() {
		defer wg.Done()
		s.T().Logf("Starting consumer1 for topic: %s, group: %s", topic, consumerCfg.ConsumerGroup)
		err := consumer1.Consume(consumeCtx, handler)
		if err != nil {
			s.T().Logf("Consumer1 error: %v", err)
		}
	}()

	time.Sleep(5 * time.Second)

	go func() {
		defer wg.Done()
		s.T().Logf("Starting consumer2 for topic: %s, group: %s", topic, consumerCfg.ConsumerGroup)
		err := consumer2.Consume(consumeCtx, handler)
		if err != nil {
			s.T().Logf("Consumer2 error: %v", err)
		}
	}()

	time.Sleep(5 * time.Second)

	producer, err := kf.NewKafkaProducer(s.brokers, topic)
	require.NoError(s.T(), err)
	defer producer.Close()

	messageCount := 10
	for i := 0; i < messageCount; i++ {
		event := domain.Event{
			EventID:   fmt.Sprintf("rebalance-%d", i),
			EventType: domain.EventTypeOrderReturnedByClient,
			Timestamp: time.Now(),
			Actor:     domain.Actor{Type: domain.ActorTypeClient, ID: uint64(i)},
			Order:     domain.OrderInfo{ID: uint64(i), UserID: uint64(i), Status: "returned"},
			Source:    "rebalance-test",
		}

		message, err := json.Marshal(event)
		require.NoError(s.T(), err)

		s.T().Logf("Sending message %d: %s", i, message)
		err = producer.Send(ctx, message)
		require.NoError(s.T(), err)
		time.Sleep(200 * time.Millisecond)
	}

	receivedCount := 0
	timeout := time.After(20 * time.Second)

	for receivedCount < messageCount {
		select {
		case id := <-receivedEvents:
			s.T().Logf("Received event ID: %s", id)
			receivedCount++
		case <-timeout:
			s.T().Fatalf("Timeout: received only %d/%d messages", receivedCount, messageCount)
		}
	}

	consumer1.Close()
	consumer2.Close()

	assert.Equal(s.T(), messageCount, receivedCount)
}

type testMessageHandler struct {
	receivedEvents chan domain.Event
}

func (h *testMessageHandler) HandleMessage(ctx context.Context, message *sarama.ConsumerMessage) error {
	var event domain.Event
	if err := json.Unmarshal(message.Value, &event); err != nil {
		return err
	}

	select {
	case h.receivedEvents <- event:
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

type countingHandler struct {
	receivedIDs chan string
}

func (h *countingHandler) HandleMessage(ctx context.Context, message *sarama.ConsumerMessage) error {
	var event domain.Event
	if err := json.Unmarshal(message.Value, &event); err != nil {
		return err
	}

	select {
	case h.receivedIDs <- event.EventID:
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}
