# 🌐 HTTP + gRPC API Contract

## HTTP API


### GET /order/{order_id}/history
**Описание:** Получить историю статусов по заказу

**Response:**
```json
{
  "history": [
    { "order_id": "123", "status": "issued", "timestamp": "2025-04-10T10:00:00Z" }
  ]
}
```

---

## gRPC API

```proto
service OrdersService {
  rpc GetOrderHistory (OrderHistoryRequest) returns (OrderHistoryResponse);
}

message OrderHistoryRequest {
  string order_id = 1;
}

message OrderHistoryResponse {
  repeated OrderHistory history = 1;
}
```


# Формат ошибок (HTTP+gRPC) 

```json
{"error": { "code": "ORDER_NOT_FOUND", "message": "Order not found" }
```

*message может быть произвольным, главное использовать допустимый code*