package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	OrdersIssuedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pvz_orders_issued_total",
		Help: "The total number of orders issued to clients",
	})

	OrdersAcceptedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pvz_orders_accepted_total",
		Help: "The total number of orders accepted from couriers",
	})

	OrdersReturnedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pvz_orders_returned_total",
		Help: "The total number of returned orders",
	}, []string{"type"})

	OrdersByStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pvz_orders_by_status",
		Help: "Number of orders by status",
	}, []string{"status"})

	GRPCDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "pvz_grpc_duration_seconds",
		Help:    "Duration of gRPC calls in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "status"})

	WorkerPoolActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pvz_worker_pool_active",
		Help: "Number of active workers in the pool",
	})

	WorkerPoolQueueSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pvz_worker_pool_queue_size",
		Help: "Current size of the worker pool queue",
	})

	KafkaMessagesProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pvz_kafka_messages_processed_total",
		Help: "Total number of processed Kafka messages",
	}, []string{"status"})

	CacheSize = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pvz_cache_size",
		Help: "Current cache size by cache type",
	}, []string{"cache_type"})

	CacheHits = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pvz_cache_hits_total",
		Help: "Total number of cache hits/misses",
	}, []string{"cache_type", "result"})
)
