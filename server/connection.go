package server

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/xid"
	log "github.com/sirupsen/logrus"
	"gitlab.citicom.kz/CloudServer/server/icontext"
	"gitlab.citicom.kz/CloudServer/server/models"
)

const (
	outputChannelBufferSize = 10
	pongWait                = 6 * time.Second
	pingPeriod              = (pongWait * 9) / 10
)

type SocketConnection struct {
	ID              string
	User            *models.User
	conn            *websocket.Conn
	server          *Server
	outgoingMessage chan *models.OutputMessage
	closeCh         chan bool
	logger          *log.Entry
}

func NewSocketConnection(server *Server, conn *websocket.Conn, user *models.User) *SocketConnection {
	if conn == nil {
		panic("connection cannot be nil")
	}
	if server == nil {
		panic(" Server cannot be nil")
	}

	guid := xid.New()
	id := guid.String()

	ch := make(chan *models.OutputMessage, outputChannelBufferSize)
	closeCh := make(chan bool)

	connectionLogger := log.WithFields(log.Fields{
		"ConnectionId": id,
		"UserId":       user.UserID,
	})

	return &SocketConnection{
		id,
		user,
		conn,
		server,
		ch,
		closeCh,
		connectionLogger,
	}
}

func (cc *SocketConnection) Close() {
	cc.logger.Info("Signal to close ")
	close(cc.closeCh)
}

func (cc *SocketConnection) Run() {
	cc.logger.Info("Running")
	go cc.listenWrite()
	go cc.listenRead()
}

func (cc *SocketConnection) listenWrite() {
	pingTicker := time.NewTicker(pingPeriod)
	defer func() {
		_ = cc.conn.Close()
		pingTicker.Stop()
	}()

	for {
		select {
		case <-pingTicker.C:
			if err := cc.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				cc.logger.Errorf("Can't write ping to connection: %s", err.Error())
			}
		case mes := <-cc.outgoingMessage:
			err := cc.conn.WriteJSON(&mes)
			if err != nil {
				cc.logger.Errorf("Can't write to connection: %s", err.Error())
			}
		case <-cc.closeCh:
			cc.logger.Info("Closed writing")
			return
		}
	}
}

func (cc *SocketConnection) listenRead() {
	defer func() {
		_ = cc.conn.Close()
	}()

	_ = cc.conn.SetReadDeadline(time.Now().Add(pongWait))
	cc.conn.SetPongHandler(func(string) error {
		_ = cc.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		select {

		case <-cc.closeCh:
			cc.logger.Info("Closed reading")
			return
		default:
			var messageObject models.InputMessage
			err := cc.conn.ReadJSON(&messageObject)

			ctx := context.Background()
			guid := xid.New()
			requestID := guid.String()
			requestLogger := log.WithFields(log.Fields{"request_id": requestID})
			ctx = context.WithValue(ctx, icontext.LoggerContextKey, requestLogger)
			if err != nil {
				if c, k := err.(*websocket.CloseError); k {
					if c.Code >= 1000 && c.Code <= 1015 {
						cc.logger.Infof("Close websocket by code: %d, message: %s", c.Code, c.Text)
					}
				} else {
					cc.logger.Errorf("Error while reading JSON from websocket %s", err.Error())
				}
				cc.server.socketDisconnect(ctx, cc.User.UserID, cc.ID)
				return
			} else {
				//cc.server.NewIncomingMessage(ctx, &messageObject, cc.User)
			}
		}
	}
}

// SendMessage - add message to output queue
func (cc *SocketConnection) SendMessage(mes *models.OutputMessage) {
	cc.outgoingMessage <- mes
}
