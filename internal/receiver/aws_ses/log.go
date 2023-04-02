package receiver

import (
	"encoding/json"
	"fmt"
	"time"
)

type AwsLog struct {
	Time      time.Time
	Component string
	Msg       *string `json:"Msg,omitempty"`
	Error     *string `json:"Error,omitempty"`
}

func LogError(component string, format string, args ...any) error {
	err := fmt.Errorf(format, args...)
	errStr := err.Error()
	entry := AwsLog{
		Time:      time.Now().UTC(),
		Component: component,
		Error:     &errStr,
	}
	b, _ := json.Marshal(entry)
	fmt.Println(string(b))
	return err
}

func Log(component string, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	entry := AwsLog{
		Time:      time.Now().UTC(),
		Component: component,
		Msg:       &msg,
	}
	b, _ := json.Marshal(entry)
	fmt.Println(string(b))
}
