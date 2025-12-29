package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/garrettladley/thoop/internal/client/sse"
	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/xsync"
)

type SSEConnectedMsg struct{}

type SSEDisconnectedMsg struct {
	Err error
}

type NotificationMsg struct {
	Notification storage.Notification
	Result       xsync.ProcessResult
}

// StartSSECmd launches a goroutine that connects to the SSE server and pushes
// notifications to the provided channel. The channel bridges the blocking SSE
// client with bubbletea's message system.
func StartSSECmd(ctx context.Context, client *sse.Client, notifCh chan<- storage.Notification) tea.Cmd {
	return func() tea.Msg {
		err := client.Connect(ctx, func(n storage.Notification) {
			select {
			case notifCh <- n:
			case <-ctx.Done():
			}
		})

		return SSEDisconnectedMsg{Err: err}
	}
}

// ListenNotificationsCmd reads from the notification channel, processes the
// notification via the NotificationProcessor, and returns a NotificationMsg.
// This command should be re-invoked after each message to continue listening.
func ListenNotificationsCmd(ctx context.Context, notifCh <-chan storage.Notification, processor *xsync.NotificationProcessor) tea.Cmd {
	return func() tea.Msg {
		select {
		case n, ok := <-notifCh:
			if !ok {
				return SSEDisconnectedMsg{Err: nil}
			}
			result := processor.Process(ctx, n)
			return NotificationMsg{
				Notification: n,
				Result:       result,
			}
		case <-ctx.Done():
			return SSEDisconnectedMsg{Err: ctx.Err()}
		}
	}
}
