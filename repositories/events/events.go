package events

const (
	PushEvent string = "push"
)

var (
	exists = struct{}{}

	validEvents = map[string]struct{}{
		PushEvent: exists,
	}
)

func IsValidEvent(event string) bool {
	_, ok := validEvents[event]
	return ok
}
