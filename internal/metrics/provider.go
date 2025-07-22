package metrics

type MetricsProvider interface {
	OrderAccepted()
	OrdersIssued(count uint64)
	OrdersReturned(returnType string, count uint64)
	UpdateOrderStatusMetrics(statusCounts map[string]int)

	RecordGRPCDuration(method, status string, duration float64)

	UpdateWorkerPoolMetrics(active, total, queueSize, queueCapacity int)

	KafkaMessageProcessed(status string)

	UpdateCacheMetrics(stats map[string]int)
	RecordCacheHit(cacheType, result string)
}

type PrometheusProvider struct{}

func NewPrometheusProvider() *PrometheusProvider {
	return &PrometheusProvider{}
}

func (p *PrometheusProvider) OrderAccepted() {
	OrdersAcceptedTotal.Inc()
}

func (p *PrometheusProvider) OrdersIssued(count uint64) {
	OrdersIssuedTotal.Add(float64(count))
}

func (p *PrometheusProvider) OrdersReturned(returnType string, count uint64) {
	OrdersReturnedTotal.WithLabelValues(returnType).Add(float64(count))
}

func (p *PrometheusProvider) UpdateOrderStatusMetrics(statusCounts map[string]int) {
	for status, count := range statusCounts {
		OrdersByStatus.WithLabelValues(status).Set(float64(count))
	}
}

func (p *PrometheusProvider) RecordGRPCDuration(method, status string, duration float64) {
	GRPCDuration.WithLabelValues(method, status).Observe(duration)
}

func (p *PrometheusProvider) UpdateWorkerPoolMetrics(active, total, queueSize, queueCapacity int) {
	WorkerPoolActive.Set(float64(active))
	WorkerPoolQueueSize.Set(float64(queueSize))
}

func (p *PrometheusProvider) KafkaMessageProcessed(status string) {
	KafkaMessagesProcessed.WithLabelValues(status).Inc()
}

func (p *PrometheusProvider) UpdateCacheMetrics(stats map[string]int) {
	for cacheType, size := range stats {
		CacheSize.WithLabelValues(cacheType).Set(float64(size))
	}
}

func (p *PrometheusProvider) RecordCacheHit(cacheType, result string) {
	CacheHits.WithLabelValues(cacheType, result).Inc()
}

type NoOpProvider struct{}

func NewNoOpProvider() *NoOpProvider {
	return &NoOpProvider{}
}

func (p *NoOpProvider) OrderAccepted()                                                      {}
func (p *NoOpProvider) OrdersIssued(count uint64)                                           {}
func (p *NoOpProvider) OrdersReturned(returnType string, count uint64)                      {}
func (p *NoOpProvider) UpdateOrderStatusMetrics(statusCounts map[string]int)                {}
func (p *NoOpProvider) RecordGRPCDuration(method, status string, duration float64)          {}
func (p *NoOpProvider) UpdateWorkerPoolMetrics(active, total, queueSize, queueCapacity int) {}
func (p *NoOpProvider) KafkaMessageProcessed(status string)                                 {}
func (p *NoOpProvider) UpdateCacheMetrics(stats map[string]int)                             {}
func (p *NoOpProvider) RecordCacheHit(cacheType, result string)                             {}
