package gorpc

import (
	"bytes"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

// Profile .
type Profile struct {
	Timestamp time.Time // profile timestamp
	Pipelines int32     // register pipelines
	Actives   int32     // actived pipelines
	Received  uint64    // received messages
	Send      uint64    // send messages
	Errors    uint64    // panic times
}

type _ProfileHandler struct {
	Profile
}

func (profile *_ProfileHandler) Register(context Context) error {

	atomic.AddInt32(&profile.Pipelines, 1)

	return nil
}

func (profile *_ProfileHandler) Unregister(context Context) {
	atomic.AddInt32(&profile.Pipelines, -1)
}

func (profile *_ProfileHandler) Active(context Context) error {

	atomic.AddInt32(&profile.Actives, 1)

	return nil
}

func (profile *_ProfileHandler) Inactive(context Context) {

	atomic.AddInt32(&profile.Actives, -1)
}

func (profile *_ProfileHandler) MessageReceived(context Context, message *Message) (*Message, error) {

	atomic.AddUint64(&profile.Received, 1)

	return message, nil
}

func (profile *_ProfileHandler) MessageSending(context Context, message *Message) (*Message, error) {

	atomic.AddUint64(&profile.Send, 1)

	return message, nil
}

func (profile *_ProfileHandler) Panic(context Context, err error) {
	atomic.AddUint64(&profile.Errors, 1)
}

var profileHandler = &_ProfileHandler{}

// ProfileHandler create profile handler
func ProfileHandler() Handler {
	return profileHandler
}

// GetProfile .
func GetProfile() *Profile {

	profile := profileHandler.Profile

	profile.Timestamp = time.Now()

	return &profile
}

var lastprofile = GetProfile()

// PrintProfile .
func PrintProfile() string {
	profile := GetProfile()

	var buff bytes.Buffer

	titles := []string{"pipeline", "actives", "recv", "send", "errors", "speed-r(op/s)", "speed-s(op/s)"}

	width := 18

	for _, title := range titles {
		buff.WriteString(title)
		buff.WriteString(strings.Repeat(" ", width-len(title)))
	}

	buff.WriteString("\n")

	val := fmt.Sprintf("%d", profile.Pipelines)
	buff.WriteString(val)
	buff.WriteString(strings.Repeat(" ", width-len(val)))

	val = fmt.Sprintf("%d", profile.Actives)
	buff.WriteString(val)
	buff.WriteString(strings.Repeat(" ", width-len(val)))

	val = fmt.Sprintf("%d", profile.Received)
	buff.WriteString(val)
	buff.WriteString(strings.Repeat(" ", width-len(val)))

	val = fmt.Sprintf("%d", profile.Send)
	buff.WriteString(val)
	buff.WriteString(strings.Repeat(" ", width-len(val)))

	val = fmt.Sprintf("%d", profile.Errors)
	buff.WriteString(val)
	buff.WriteString(strings.Repeat(" ", width-len(val)))

	duration := profile.Timestamp.Sub(lastprofile.Timestamp)

	val = fmt.Sprintf("%f", float64(profile.Received-lastprofile.Received)*float64(time.Second)/float64(duration))
	buff.WriteString(val)
	buff.WriteString(strings.Repeat(" ", width-len(val)))

	val = fmt.Sprintf("%f", float64(profile.Send-lastprofile.Send)*float64(time.Second)/float64(duration))
	buff.WriteString(val)
	buff.WriteString(strings.Repeat(" ", width-len(val)))

	lastprofile = profile

	return buff.String()
}
