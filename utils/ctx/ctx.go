package ctx

import (
	"context"
	"fmt"
	"os"
	"os/signal"
)

type ConsoleAppContext struct {
	ctx       context.Context
	cancel    context.CancelFunc
	osSigChan chan os.Signal
}

var consoleAppCtx *ConsoleAppContext

func init() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	osSigChan := make(chan os.Signal, 1)

	consoleAppCtx = &ConsoleAppContext{ctx: ctx,
		cancel:    cancel,
		osSigChan: osSigChan}

	signal.Notify(osSigChan, os.Interrupt)
	// trap Ctrl+C and call cancel on the context
}

func GetContext() context.Context {
	return consoleAppCtx.ctx
}

func WaitOnCtrlCSignalOrCompletion() {
	select {
	case <-consoleAppCtx.osSigChan:
		fmt.Println("program terminated because of user cancellation")
		consoleAppCtx.cancel()
	case <-consoleAppCtx.ctx.Done():
	}
}

func CleanUp() {
	consoleAppCtx.cleanUp()
}

func (consoleAppCtx *ConsoleAppContext) cleanUp() {
	signal.Stop(consoleAppCtx.osSigChan)
	consoleAppCtx.cancel()
}
