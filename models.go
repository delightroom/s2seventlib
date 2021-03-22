package s2seventlib

type CommonEventType string

const (
	CommonEventPurchase         CommonEventType = "purchase"
	CommonEventRenew            CommonEventType = "renew"
	CommonEventRecover          CommonEventType = "recover"
	CommonEventReEnable         CommonEventType = "re_enable"
	CommonEventPause            CommonEventType = "pause"
	CommonEventTurnOnAutoRenew  CommonEventType = "turn_on_auto_renew"
	CommonEventTurnOffAutoRenew CommonEventType = "turn_off_auto_renew"
	CommonEventCancel           CommonEventType = "cancel"
)

type PaymentState int64

const (
	PaymentStatePending         PaymentState = 0
	PaymentStateReceived        PaymentState = 1
	PaymentStateTrial           PaymentState = 2
	PaymentStatePendingDeferred PaymentState = 3
)

type CommonEvent struct {
	EventType       CommonEventType       `json:"event_type"`
	UserID          string                `json:"user_id"`
	Platform        string                `json:"platform"`
	EventTimeMillis int                   `json:"event_time_millis"`
	Env             string                `json:"env"`
	Properties      CommonEventProperties `json:"properties"`
}

type CommonEventProperties struct {
	PaymentState       PaymentState `json:"payment_state"`
	AppID              string       `json:"app_id"`
	ProductID          string       `json:"product_id"`
	Currency           string       `json:"currency"`
	Price              float64      `json:"price"`
	Quantity           int          `json:"quantity"`
	CancellationReason string       `json:"cancellation_reason"`
}
