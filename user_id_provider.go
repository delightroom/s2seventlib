package s2seventlib

type UserIDProvider interface {
	UserID(token string) (string, error)
}
