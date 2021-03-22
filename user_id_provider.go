package s2seventlib

type UserIDProvider interface {
	userID(token string) (string, error)
}
