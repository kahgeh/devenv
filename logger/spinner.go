package logger

import (
	"time"

	"github.com/kahgeh/devenv/utils/ctx"
	"github.com/theckman/yacspin"
)

type Spinner struct {
	logStream    chan string
	newComponent chan *yacspin.Spinner
}

const (
	SucceedCompletionStatus = "[succeed]"
	FailedCompletionStatus  = "[failed]"
)

type CompletionStatus string

func slowDownForReading() {
	time.Sleep(2 * time.Second)
}
func (spinner *Spinner) run() {
	var component *yacspin.Spinner
	for {
		select {
		case newComponent := <-spinner.newComponent:
			component = newComponent
			component.Start()
		case message := <-spinner.logStream:
			switch message {
			case "[succeed]":
				if component != nil {
					component.Stop()
				}
				component = nil
			case "[failed]":
				if component != nil {
					component.StopFail()
				}
				component = nil
			default:
				if component == nil {
					panic("ensure a started spinner is available")
				}
				component.Message(message)
				slowDownForReading()
			}
		case <-ctx.GetContext().Done():
			if component != nil {
				component.StopFail()
			}
			return
		}
	}
}

func (spinner *Spinner) update(message string) {
	spinner.logStream <- message
}

func (spinner *Spinner) failed(message string) {
	spinner.logStream <- message
	spinner.logStream <- FailedCompletionStatus
}

func (spinner *Spinner) succeed() {
	spinner.logStream <- SucceedCompletionStatus
}
