package state

var currentState string

func GetState() string {
	return currentState
}

func SetState(s string) {
	currentState = s
}
