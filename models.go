package s2seventlib

type EventType string

const (
	EventPurchase         EventType = "purchase"
	EventRenew            EventType = "renew"
	EventRecover          EventType = "recover"
	EventRestart          EventType = "restart"
	EventPause            EventType = "pause"
	EventTurnOnAutoRenew  EventType = "turn_on_auto_renew"
	EventTurnOffAutoRenew EventType = "turn_off_auto_renew"
	EventCancel           EventType = "cancel"
)

type PaymentState int64

const (
	PaymentStatePending         PaymentState = 0 // Cancel과 같이 PaymentState값이 없는 경우도 0으로 지정한다.
	PaymentStateReceived        PaymentState = 1
	PaymentStateTrial           PaymentState = 2
	PaymentStatePendingDeferred PaymentState = 3
)

type Event struct {
	EventType       EventType       `json:"event_type"`
	UserID          string          `json:"user_id"`
	Platform        string          `json:"platform"`
	EventTimeMillis int             `json:"event_time_millis"`
	Env             string          `json:"env"`
	Properties      EventProperties `json:"properties"`
}

type EventProperties struct {
	PaymentState       PaymentState `json:"payment_state"`
	AppID              string       `json:"app_id"`
	ProductID          string       `json:"product_id"`
	Currency           string       `json:"currency"`
	Price              float64      `json:"price"`
	Quantity           int          `json:"quantity"`
	CancellationReason string       `json:"cancellation_reason"`
}
