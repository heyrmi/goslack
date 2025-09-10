package api

// HubAdapter adapts the Hub to implement the MessageHub interface
type HubAdapter struct {
	hub *Hub
}

func NewHubAdapter(hub *Hub) *HubAdapter {
	return &HubAdapter{hub: hub}
}

// BroadcastToChannel implements the MessageHub interface
func (ha *HubAdapter) BroadcastToChannel(workspaceID, channelID int64, message interface{}) {
	ha.hub.BroadcastToChannelGeneric(workspaceID, channelID, message)
}

// BroadcastToUser implements the MessageHub interface
func (ha *HubAdapter) BroadcastToUser(userID int64, message interface{}) {
	ha.hub.BroadcastToUserGeneric(userID, message)
}
