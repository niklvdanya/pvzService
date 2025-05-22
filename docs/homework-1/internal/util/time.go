package util

import "time"

// возможно избыточный пакет, но я добавил поля(часы и минуты) во flag expires
// чтобы проще было дебажить некоторые команды, где есть проверка с time.Now()
// но я столкнулся с проблемой часовых поясов и поэтому добавил этот пакет
var moscowTime, _ = time.LoadLocation("Europe/Moscow")

func GetMoscowLocation() *time.Location {
	return moscowTime
}

func NowInMoscow() time.Time {
	return time.Now().In(moscowTime)
}
