package infra

import (
	"context"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.ozon.dev/safariproxd/homework/internal/infra/mock"
)

func TestKafkaProducer_Send(t *testing.T) {
	t.Parallel()

	type testFixture struct {
		ctx     context.Context
		topic   string
		message []byte
	}

	fixture := testFixture{
		ctx:     context.Background(),
		topic:   "test-topic",
		message: []byte(`{"event_id": "test-123", "event_type": "test_event"}`),
	}

	expectSendMessage := func(mock *mock.SyncProducerMock, topic string, message []byte, partition int32, offset int64, err error) {
		mock.SendMessageMock.Set(func(msg *sarama.ProducerMessage) (int32, int64, error) {
			assert.Equal(t, topic, msg.Topic)
			assert.Equal(t, string(message), string(msg.Value.(sarama.ByteEncoder)))
			return partition, offset, err
		})
	}

	tests := []struct {
		name    string
		prepare func(*testing.T, *mock.SyncProducerMock, testFixture)
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Success_SendMessage",
			prepare: func(t *testing.T, mock *mock.SyncProducerMock, f testFixture) {
				expectSendMessage(mock, f.topic, f.message, 0, 1, nil)
			},
			wantErr: assert.NoError,
		},
		{
			name: "Fail_KafkaError",
			prepare: func(t *testing.T, mock *mock.SyncProducerMock, f testFixture) {
				expectSendMessage(mock, f.topic, f.message, 0, 0, assert.AnError)
			},
			wantErr: func(t assert.TestingT, err error, _ ...interface{}) bool {
				return assert.Error(t, err) && assert.Contains(t, err.Error(), "send message to kafka")
			},
		},
		{
			name: "Fail_ContextCanceled",
			prepare: func(t *testing.T, mock *mock.SyncProducerMock, f testFixture) {
			},
			wantErr: func(t assert.TestingT, err error, _ ...interface{}) bool {
				return assert.Equal(t, context.Canceled, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := minimock.NewController(t)
			mockProducer := mock.NewSyncProducerMock(ctrl)

			producer := &KafkaProducer{
				producer: mockProducer,
				topic:    fixture.topic,
			}

			ctx := fixture.ctx
			if tt.name == "Fail_ContextCanceled" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(fixture.ctx)
				cancel()
			} else {
				tt.prepare(t, mockProducer, fixture)
			}

			err := producer.Send(ctx, fixture.message)
			tt.wantErr(t, err)
		})
	}
}

func TestKafkaProducer_Send_ContextTimeout(t *testing.T) {
	t.Parallel()

	ctrl := minimock.NewController(t)
	mockProducer := mock.NewSyncProducerMock(ctrl)

	producer := &KafkaProducer{
		producer: mockProducer,
		topic:    "test-topic",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	time.Sleep(1 * time.Millisecond)

	err := producer.Send(ctx, []byte("test"))

	require.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestKafkaProducer_Close(t *testing.T) {
	t.Parallel()
	expectClose := func(mock *mock.SyncProducerMock, err error) {
		mock.CloseMock.Return(err)
	}

	tests := []struct {
		name    string
		prepare func(*testing.T, *mock.SyncProducerMock)
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Success_Close",
			prepare: func(t *testing.T, mock *mock.SyncProducerMock) {
				expectClose(mock, nil)
			},
			wantErr: assert.NoError,
		},
		{
			name: "Fail_CloseError",
			prepare: func(t *testing.T, mock *mock.SyncProducerMock) {
				expectClose(mock, assert.AnError)
			},
			wantErr: func(t assert.TestingT, err error, _ ...interface{}) bool {
				return assert.Error(t, err) && assert.Contains(t, err.Error(), "close producer")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := minimock.NewController(t)
			mockProducer := mock.NewSyncProducerMock(ctrl)

			producer := &KafkaProducer{
				producer: mockProducer,
				topic:    "test-topic",
			}

			tt.prepare(t, mockProducer)

			err := producer.Close()
			tt.wantErr(t, err)
		})
	}
}
