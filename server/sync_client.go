package server

import (
	"context"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/xid"
	log "github.com/sirupsen/logrus"
	"gitlab.citicom.kz/CloudServer/server/icontext"
	"gitlab.citicom.kz/CloudServer/server/models"
)

type SyncClient struct {
	OilFieldID      int64
	address         string
	conn            *websocket.Conn
	server          *Server
	outgoingMessage chan *models.OutputMessage
	closeCh         chan bool
	logger          *log.Entry
}

func NewSyncClient(oilFieldID int64, address string, conn *websocket.Conn, server *Server, logger *log.Entry) *SyncClient {
	ch := make(chan *models.OutputMessage, outputChannelBufferSize)
	closeCh := make(chan bool)

	return &SyncClient{
		OilFieldID:      oilFieldID,
		address:         address,
		conn:            conn,
		server:          server,
		outgoingMessage: ch,
		closeCh:         closeCh,
		logger:          logger,
	}
}

func (cc *SyncClient) Close() {
	cc.logger.Info("Signal to close ")
	close(cc.closeCh)
}

func (cc *SyncClient) Run() {
	cc.logger.Info("Running")
	go cc.listenWrite()
	go cc.listenRead()
}

func (cc *SyncClient) listenWrite() {
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
			cc.logger.Infof("Ping message has been send")
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

func (cc *SyncClient) listenRead() {
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
					fmt.Println("Error while reading JSON from websocket %s", err.Error())
					cc.logger.Errorf("Error while reading JSON from websocket %s", err.Error())
				}
				cc.server.masterSocketDisconnect(ctx, cc.OilFieldID)
				return
			} else {
				cc.server.NewIncomingMessage(ctx, &messageObject, cc.OilFieldID)
			}
		}
	}
}

// SendMessage - add message to output queue
func (cc *SyncClient) SendMessage(mes *models.OutputMessage) {
	cc.outgoingMessage <- mes
}
