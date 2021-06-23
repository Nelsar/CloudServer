package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/xid"
	log "github.com/sirupsen/logrus"
	"gitlab.citicom.kz/CloudServer/server/database"
	"gitlab.citicom.kz/CloudServer/server/icontext"
	"gitlab.citicom.kz/CloudServer/server/influx"
	"gitlab.citicom.kz/CloudServer/server/middleware"
	"gitlab.citicom.kz/CloudServer/server/models"
	"gitlab.citicom.kz/CloudServer/server/response"
	"gitlab.citicom.kz/CloudServer/server/upload"
	"gitlab.citicom.kz/CloudServer/server/utils"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type incomingMessageWithContext struct {
	Message    *models.InputMessage
	OilFieldId int64
	Context    context.Context
}

type Server struct {
	closeCh                     chan bool
	middleware                  []func(next http.Handler) http.Handler
	connectString               string
	SocketConnectionsPool       map[int64][]*SocketConnection
	MasterSocketConnectionsPool map[int64]*SyncClient
	newIncomingMessage          chan *incomingMessageWithContext
	logger                      *log.Entry
	db                          *database.DB
	influxDB                    *influx.Influx
}

func NewServer(host, port string, db *database.DB, influxDB *influx.Influx) *Server {
	closeCh := make(chan bool)
	socketConnectionsPool := make(map[int64][]*SocketConnection)
	masterSocketConnectionPool := make(map[int64]*SyncClient)

	var _middleware []func(next http.Handler) http.Handler
	_middleware = append(_middleware, middleware.PermissionsMiddleware())
	_middleware = append(_middleware, middleware.AuthMiddleware(db, []string{
		"/auth",
	}))
	//_middleware = append(_middleware, middleware.ActionLogMiddleware(db))
	_middleware = append(_middleware, middleware.LogMiddleware())
	_middleware = append(_middleware, middleware.UniqueVisitMiddleware([]string{
		"/auth",
	}))
	connectString := fmt.Sprintf("%s:%s", host, port)
	newIncomingMessage := make(chan *incomingMessageWithContext)
	serverLogger := log.WithFields(log.Fields{
		"ServerThread": "Main",
	})

	return &Server{
		closeCh:                     closeCh,
		middleware:                  _middleware,
		connectString:               connectString,
		SocketConnectionsPool:       socketConnectionsPool,
		MasterSocketConnectionsPool: masterSocketConnectionPool,
		logger:                      serverLogger,
		newIncomingMessage:          newIncomingMessage,
		db:                          db,
		influxDB:                    influxDB,
	}
}

func (server *Server) wrapMiddleware(handler http.Handler) http.Handler {
	next := handler
	for _, m := range server.middleware {
		next = m(next)
	}
	return next
}

// Run server
func (server *Server) Run() {
	server.logger.Infof("Server is starting...")

	var wg sync.WaitGroup

	wg.Add(2)
	go server.runREST(&wg)
	go server.runMasterWebsocket(&wg)
	go server.runMasterDaemon(&wg)
	go server.runWebsocket(&wg)

	wg.Wait()
}

func (server *Server) runMasterDaemon(wg *sync.WaitGroup) {
	defer func() {
		server.logger.Infof("Master daemon shutdowning...")
		wg.Done()
		if err := recover(); err != nil {
			server.logger.Errorf("Panic in master daemon: %v", err)
		}
	}()

	ctx := context.Background()
	guid := xid.New()
	requestID := guid.String()
	requestLogger := log.WithFields(log.Fields{"request_id": requestID})
	ctx = context.WithValue(ctx, icontext.LoggerContextKey, requestLogger)

	for {
		select {
		case <-time.After(time.Second * 3):
		}

		oilFields, err := server.db.GetOilFields(ctx, 0, true)
		if err != nil {
			continue
		}

		for _, oilField := range oilFields {
			if !oilField.IsDeleted {
				server.listenOilField(ctx, oilField)
			} else {
				server.disconnectOilField(ctx, oilField.OilFieldId)
			}

		}
	}
}

func (server *Server) runMasterWebsocket(wg *sync.WaitGroup) {
	defer func() {
		server.logger.Infof("Master websocket shutdowning...")
		wg.Done()
		if err := recover(); err != nil {
			server.logger.Errorf("Panic in runMasterWebsocket: %v", err)
		}
	}()

	server.logger.Infof("Master websocket listening...")
	for {
		select {
		case <-server.closeCh:
			return
		}
	}
}

func (server *Server) runWebsocket(wg *sync.WaitGroup) {
	defer func() {
		server.logger.Infof("Websocket shutdowning...")
		wg.Done()
		if err := recover(); err != nil {
			server.logger.Errorf("Panic in runWebsocket: %v", err)
		}
	}()

	server.logger.Infof("Websocket listening...")
	for {
		select {
		case mes := <-server.newIncomingMessage:
			go server.ProcessNewIncomingMessage(mes.Context, mes.Message, mes.OilFieldId)
		case <-server.closeCh:
			return
		}
	}
}

func (server *Server) runREST(wg *sync.WaitGroup) {
	defer func() {
		server.logger.Infof("Rest shutdowning...")
		wg.Done()

		if err := recover(); err != nil {
			server.logger.Errorf("Panic in runREST: %v", err)
		}
	}()

	http.Handle("/connect", server.wrapMiddleware(http.HandlerFunc(server.connect)))

	http.Handle("/auth", server.wrapMiddleware(http.HandlerFunc(server.auth)))

	http.Handle("/companyData", server.wrapMiddleware(http.HandlerFunc(server.companyData)))
	http.Handle("/companyData/save", server.wrapMiddleware(http.HandlerFunc(server.companyDataSave)))

	http.Handle("/users/list", server.wrapMiddleware(http.HandlerFunc(server.users)))
	http.Handle("/users/create", server.wrapMiddleware(http.HandlerFunc(server.usersCreate)))
	http.Handle("/users/update", server.wrapMiddleware(http.HandlerFunc(server.usersUpdate)))
	http.Handle("/users/usersChangePassword", server.wrapMiddleware(http.HandlerFunc(server.usersChangePassword)))
	http.Handle("/users/delete", server.wrapMiddleware(http.HandlerFunc(server.usersDelete)))

	http.Handle("/companies/list", server.wrapMiddleware(http.HandlerFunc(server.companies)))
	http.Handle("/companies/save", server.wrapMiddleware(http.HandlerFunc(server.companiesSave)))
	http.Handle("/companies/delete", server.wrapMiddleware(http.HandlerFunc(server.companyDelete)))

	http.Handle("/oil_fields/list", server.wrapMiddleware(http.HandlerFunc(server.oilFields)))
	http.Handle("/oil_fields/save", server.wrapMiddleware(http.HandlerFunc(server.oilFieldsSave)))
	http.Handle("/oil_fields/delete", server.wrapMiddleware(http.HandlerFunc(server.oilFieldsDelete)))

	http.Handle("/controllers/list", server.wrapMiddleware(http.HandlerFunc(server.controllersList)))
	http.Handle("/controllers/data", server.wrapMiddleware(http.HandlerFunc(server.controllerData)))

	http.Handle("/mnemoschemes/list", server.wrapMiddleware(http.HandlerFunc(server.mnemoschemesList)))
	http.Handle("/mnemoschemes/save", server.wrapMiddleware(http.HandlerFunc(server.mnemoschemesSave)))
	http.Handle("/mnemoschemes/data", server.wrapMiddleware(http.HandlerFunc(server.mnemoschemesData)))
	http.Handle("/mnemoschemes/mnemoschemesInfoSave", server.wrapMiddleware(http.HandlerFunc(server.mnemoschemesInfoSave)))

	http.Handle("/alarms/list", server.wrapMiddleware(http.HandlerFunc(server.alarmsList)))
	http.Handle("/alarms/markAsViewed", server.wrapMiddleware(http.HandlerFunc(server.markAlarmViewed)))

	http.Handle("/sensors/list", server.wrapMiddleware(http.HandlerFunc(server.sensorsList)))
	http.Handle("/actions/list", server.wrapMiddleware(http.HandlerFunc(server.actionsList)))

	http.Handle("/files", server.wrapMiddleware(http.HandlerFunc(server.files)))

	server.logger.Fatal(http.ListenAndServe(server.connectString, nil))
}

func (server *Server) connect(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, _ := icontext.GetUser(ctx)

	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		server.logger.Errorf("Connect error %s", err.Error())
		return
	}

	server.logger.Infof("Connect (userId: %d)", user.UserID)

	connection := NewSocketConnection(server, conn, user)
	server.SocketConnectionsPool[user.UserID] = append(server.SocketConnectionsPool[user.UserID], connection)
	connection.Run()

	users, err := server.db.GetUsers(ctx, user.CompanyID, user.IsSuperUser())
	if err != nil {
		connection.logger.Errorf("Can't receive user contact list %s", err.Error())
	} else {
		userResult := models.UserResult{
			UserID:    user.UserID,
			CompanyID: user.CompanyID,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Role:      user.GetResultRole(),
			IsDeleted: user.IsDeleted,
			IsOnline:  server.isUserOnline(user.UserID),
			CreatedTs: user.CreatedTs,
			UpdatedTs: user.UpdatedTs,
		}

		for _, currentUser := range users {
			if currentUser.UserID == user.UserID {
				continue
			}
			server.SendMessageTo(ctx, models.MessageTypeUserOnline, userResult, currentUser.UserID)
		}
	}
}

func (server *Server) auth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l, _ := icontext.GetLogger(ctx)
	uid, _ := icontext.GetUniqueIdentifier(ctx)

	input := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{}
	err := utils.ParseJson(r, &input)
	if err != nil {
		response.ErrorResponse(l, w, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	user, err := server.db.CheckAuth(ctx, input.Email, input.Password, uid)
	if err != nil {
		if err == database.MinuteBlocked {
			response.ErrorResponse(l, w, http.StatusInternalServerError, err.Error(), nil)
			return
		}

		response.ErrorResponse(l, w, http.StatusUnprocessableEntity, err.Error(), nil)
		return
	}

	sessionID := server.db.CreateSession(ctx, user.UserID)
	if sessionID == nil {
		response.ErrorResponse(l, w, http.StatusInternalServerError, "Internal server error", nil)
		return
	}
	user.Token = *sessionID

	response.Response(l, w, user)
}

func (server *Server) companyData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, _ := icontext.GetUser(ctx)
	l, _ := icontext.GetLogger(ctx)

	input := struct {
		CompanyID int64  `json:"company_id"`
		DataType  string `json:"data_type"`
	}{}
	err := utils.ParseJson(r, &input)
	if err != nil {
		response.ErrorResponse(l, w, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	if !user.IsSuperUser() {
		if user.CompanyID != input.CompanyID {
			response.ErrorResponse(l, w, http.StatusInternalServerError, "You are not in this company", nil)
			return
		}
	}

	desktop, err := server.db.GetCompanyData(ctx, input.CompanyID, input.DataType)
	if err != nil {
		response.ErrorResponse(l, w, http.StatusInternalServerError, "CompanyData not exists", nil)
		return
	}

	response.Response(l, w, desktop)
}

func (server *Server) companyDataSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, _ := icontext.GetUser(ctx)
	l, _ := icontext.GetLogger(ctx)
	input := models.CompanyData{}
	err := utils.ParseJson(r, &input)
	if err != nil {
		response.ErrorResponse(l, w, http.StatusInternalServerError, "Parse error", nil)
		return
	}

	if !user.IsSuperUser() {
		response.ErrorResponse(l, w, http.StatusForbidden, "Access denied", nil)
		return
	}

	desktop, err := server.db.SaveCompanyData(ctx, input.CompanyID, input.Value, input.DataType)
	if err != nil {
		response.ErrorResponse(l, w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	response.Response(l, w, desktop)
}

func (server *Server) users(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, _ := icontext.GetUser(ctx)
	l, _ := icontext.GetLogger(ctx)

	users, err := server.db.GetUsers(ctx, user.CompanyID, user.IsSuperUser())
	if err != nil {
		response.ErrorResponse(l, w, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	userResults := make([]models.UserResult, 0, 10)
	for _, user := range users {
		userResult := models.UserResult{
			UserID:      user.UserID,
			CompanyID:   user.CompanyID,
			CompanyName: user.CompanyName,
			Email:       user.Email,
			FirstName:   user.FirstName,
			LastName:    user.LastName,
			Role:        user.GetResultRole(),
			IsDeleted:   user.IsDeleted,
			IsOnline:    server.isUserOnline(user.UserID),
			CreatedTs:   user.CreatedTs,
			UpdatedTs:   user.UpdatedTs,
		}
		userResults = append(userResults, userResult)
	}

	response.Response(l, w, userResults)
}

func (server *Server) usersCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, _ := icontext.GetUser(ctx)
	l, _ := icontext.GetLogger(ctx)

	input := models.CreateUser{}
	l.Infof("usersCreate: %v", input.Email)
	err := utils.ParseJson(r, &input)
	if err != nil {
		l.Errorf("%v", err)
		response.ErrorResponse(l, w, http.StatusInternalServerError, "Parse error", nil)
		return
	}

	if !user.IsSuperUser() && input.CompanyID != user.CompanyID {
		response.ErrorResponse(l, w, http.StatusForbidden, "Access denied", nil)
		return
	}

	if err := input.Validate(); err != nil {
		l.Errorf("%v", err)
		response.ErrorResponse(l, w, http.StatusUnprocessableEntity, err.Error(), nil)
		return
	}

	if server.db.EmailExists(ctx, input.Email, 0) {
		response.ErrorResponse(l, w, http.StatusUnprocessableEntity, "User exists", nil)
		return
	}

	newUser, err := server.db.CreateUser(ctx, input)
	if err != nil {
		response.ErrorResponse(l, w, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	userResult := models.UserResult{
		UserID:    newUser.UserID,
		CompanyID: newUser.CompanyID,
		Email:     newUser.Email,
		FirstName: newUser.FirstName,
		LastName:  newUser.LastName,
		Role:      newUser.GetResultRole(),
		IsDeleted: newUser.IsDeleted,
		IsOnline:  server.isUserOnline(newUser.UserID),
		CreatedTs: newUser.CreatedTs,
		UpdatedTs: newUser.UpdatedTs,
	}

	if err != nil {
		response.ErrorResponse(l, w, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	response.Response(l, w, userResult)
}

func (server *Server) usersUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, _ := icontext.GetUser(ctx)
	l, _ := icontext.GetLogger(ctx)

	input := models.UpdateUser{}
	err := utils.ParseJson(r, &input)
	l.Errorf("%d", input.CompanyID)
	l.Errorf("%s", input.Email)
	l.Errorf("%s", input.FirstName)
	l.Errorf("%s", input.LastName)
	l.Errorf("%s", input.Role)
	if err != nil {
		response.ErrorResponse(l, w, http.StatusInternalServerError, "Parse error", nil)
		return
	}

	if !user.IsSuperUser() && input.CompanyID != user.CompanyID {
		response.ErrorResponse(l, w, http.StatusForbidden, "Access denied", nil)
		return
	}

	if err := input.Validate(); err != nil {
		l.Errorf("%v", err)
		response.ErrorResponse(l, w, http.StatusUnprocessableEntity, err.Error(), nil)
		return
	}

	if input.UserID == 0 {
		input.UserID = user.UserID
	}

	if server.db.EmailExists(ctx, input.Email, input.UserID) {
		response.ErrorResponse(l, w, http.StatusUnprocessableEntity, "Email exists", nil)
		return
	}

	updatedUser, err := server.db.UpdateUser(ctx, input)
	if err != nil {
		response.ErrorResponse(l, w, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	userResult := models.UserResult{
		UserID:    updatedUser.UserID,
		CompanyID: updatedUser.CompanyID,
		Email:     updatedUser.Email,
		FirstName: updatedUser.FirstName,
		LastName:  updatedUser.LastName,
		Role:      updatedUser.GetResultRole(),
		IsDeleted: updatedUser.IsDeleted,
		IsOnline:  server.isUserOnline(updatedUser.UserID),
		CreatedTs: updatedUser.CreatedTs,
		UpdatedTs: updatedUser.UpdatedTs,
	}

	response.Response(l, w, userResult)
}

func (server *Server) usersChangePassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, _ := icontext.GetUser(ctx)
	l, _ := icontext.GetLogger(ctx)

	input := models.UserPassword{}
	err := utils.ParseJson(r, &input)
	if err != nil {
		response.ErrorResponse(l, w, http.StatusInternalServerError, "Parse error", nil)
		return
	}

	selfUser := false

	if input.UserID != user.UserID {
		input.UserID = user.UserID
		selfUser = true
	}

	_, err = server.db.GetUser(ctx, input.UserID)
	if err != nil {
		response.ErrorResponse(l, w, http.StatusUnprocessableEntity, "User not found", nil)
		return
	}

	if !selfUser && (!user.IsAdminUser() && !user.IsSuperUser()) {
		response.ErrorResponse(l, w, http.StatusForbidden, "Access denied", nil)
		return
	}

	if err := input.Validate(); err != nil {
		response.ErrorResponse(l, w, http.StatusUnprocessableEntity, err.Error(), nil)
		return
	}

	updatedUser, err := server.db.ChangeUserPassword(ctx, input.UserID, input.Password)
	if err != nil {
		response.ErrorResponse(l, w, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	userResult := models.UserResult{
		UserID:    updatedUser.UserID,
		CompanyID: updatedUser.CompanyID,
		Email:     updatedUser.Email,
		FirstName: updatedUser.FirstName,
		LastName:  updatedUser.LastName,
		Role:      updatedUser.GetResultRole(),
		IsDeleted: updatedUser.IsDeleted,
		IsOnline:  server.isUserOnline(updatedUser.UserID),
		CreatedTs: updatedUser.CreatedTs,
		UpdatedTs: updatedUser.UpdatedTs,
	}

	response.Response(l, w, userResult)
}

func (server *Server) usersDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, _ := icontext.GetUser(ctx)
	l, _ := icontext.GetLogger(ctx)
	input := struct {
		UserID int64 `json:"userId"`
	}{}
	err := utils.ParseJson(r, &input)
	if err != nil {
		l.Errorf("%v", err)
		response.ErrorResponse(l, w, http.StatusInternalServerError, "Parse error", nil)
		return
	}

	if user.UserID == input.UserID {
		response.ErrorResponse(l, w, http.StatusUnprocessableEntity, "Don't allow self delete", nil)
		return
	}

	_, err = server.db.GetUser(ctx, input.UserID)
	if err != nil {
		l.Errorf("%v", err)
		response.ErrorResponse(l, w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	updatedUser, err := server.db.DeleteUser(ctx, input.UserID)
	if err != nil {
		l.Errorf("%v", err)
		response.ErrorResponse(l, w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	userResult := models.UserResult{
		UserID:    updatedUser.UserID,
		CompanyID: updatedUser.CompanyID,
		Email:     updatedUser.Email,
		FirstName: updatedUser.FirstName,
		LastName:  updatedUser.LastName,
		Role:      updatedUser.GetResultRole(),
		IsDeleted: updatedUser.IsDeleted,
		IsOnline:  server.isUserOnline(updatedUser.UserID),
		CreatedTs: updatedUser.CreatedTs,
		UpdatedTs: updatedUser.UpdatedTs,
	}

	response.Response(l, w, userResult)
}

func (server *Server) companies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l, _ := icontext.GetLogger(ctx)
	user, _ := icontext.GetUser(ctx)

	companies, err := server.db.GetCompanies(ctx, user.CompanyID, user.IsSuperUser())
	if err != nil {
		l.Errorf("%v", err)
		response.ErrorResponse(l, w, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	response.Response(l, w, companies)
}

func (server *Server) companiesSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l, _ := icontext.GetLogger(ctx)
	input := models.CompanyResult{}

	err := utils.ParseJson(r, &input)
	if err != nil {
		l.Errorf("%v", err)
		response.ErrorResponse(l, w, http.StatusInternalServerError, "Parse error", nil)
		return
	}

	if err := input.Validate(); err != nil {
		l.Errorf("%v", err)
		response.ErrorResponse(l, w, http.StatusUnprocessableEntity, err.Error(), nil)
		return
	}

	company, err := server.db.SaveCompany(ctx, input)
	if err != nil {
		l.Errorf("%v", err)
		response.ErrorResponse(l, w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	response.Response(l, w, company)
}

func (server *Server) companyDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l, _ := icontext.GetLogger(ctx)
	input := struct {
		CompanyID int64 `json:"companyId"`
	}{}
	err := utils.ParseJson(r, &input)
	if err != nil {
		response.ErrorResponse(l, w, http.StatusInternalServerError, "Parse error", nil)
		return
	}

	company, err := server.db.DeleteCompany(ctx, input.CompanyID)
	if err != nil {
		l.Errorf("%v", err)
		response.ErrorResponse(l, w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	response.Response(l, w, company)
}

func (server *Server) oilFields(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, _ := icontext.GetUser(ctx)
	l, _ := icontext.GetLogger(ctx)

	oilFields, err := server.db.GetOilFields(ctx, user.CompanyID, user.IsSuperUser())
	if err != nil {
		response.ErrorResponse(l, w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	oilFieldResults := make([]*models.OilFieldResult, 0, 10)
	for _, oilField := range oilFields {
		oilFieldResult := &models.OilFieldResult{
			OilFieldId:  oilField.OilFieldId,
			HttpAddress: oilField.HttpAddress,
			CompanyID:   oilField.CompanyID,
			CompanyName: oilField.CompanyName,
			Name:        oilField.Name,
			Lat:         oilField.Lat,
			Lon:         oilField.Lon,
			IsDeleted:   oilField.IsDeleted,
			CreatedTs:   oilField.CreatedTs,
			UpdatedTs:   oilField.UpdatedTs,
			IsOnline:    server.isOilFieldOnline(oilField.OilFieldId),
		}
		oilFieldResults = append(oilFieldResults, oilFieldResult)
	}

	response.Response(l, w, oilFieldResults)
}

func (server *Server) oilFieldsSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, _ := icontext.GetUser(ctx)
	l, _ := icontext.GetLogger(ctx)
	input := models.OilFieldResult{}
	err := utils.ParseJson(r, &input)
	if err != nil {
		response.ErrorResponse(l, w, http.StatusInternalServerError, "Parse error", nil)
		return
	}

	if !user.IsSuperUser() {
		input.CompanyID = user.CompanyID
	}

	oilField, err := server.db.SaveOilField(ctx, input)
	if err != nil {
		l.Errorf("%v", err)
		response.ErrorResponse(l, w, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	oilFieldResult := &models.OilFieldResult{
		OilFieldId:  oilField.OilFieldId,
		HttpAddress: oilField.HttpAddress,
		CompanyID:   oilField.CompanyID,
		Name:        oilField.Name,
		Lat:         oilField.Lat,
		Lon:         oilField.Lon,
		IsDeleted:   oilField.IsDeleted,
		CreatedTs:   oilField.CreatedTs,
		UpdatedTs:   oilField.UpdatedTs,
		IsOnline:    server.isOilFieldOnline(oilField.OilFieldId),
	}

	response.Response(l, w, oilFieldResult)
}

func (server *Server) oilFieldsDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, _ := icontext.GetUser(ctx)
	l, _ := icontext.GetLogger(ctx)
	input := struct {
		OilFieldId int64 `json:"oilFieldId"`
	}{}
	err := utils.ParseJson(r, &input)
	if err != nil {
		response.ErrorResponse(l, w, http.StatusInternalServerError, "Parse error", nil)
		return
	}

	existsOilField, err := server.db.GetOilField(ctx, input.OilFieldId)
	if err != nil {
		response.ErrorResponse(l, w, http.StatusNotFound, "Oil field not found", nil)
		return
	}

	if !user.IsSuperUser() && existsOilField.CompanyID != user.CompanyID {
		response.ErrorResponse(l, w, http.StatusForbidden, "Access denied", nil)
		return
	}

	oilField, err := server.db.DeleteOilField(ctx, input.OilFieldId)
	if err != nil {
		response.ErrorResponse(l, w, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	oilFieldResult := &models.OilFieldResult{
		OilFieldId:  oilField.OilFieldId,
		HttpAddress: oilField.HttpAddress,
		CompanyID:   oilField.CompanyID,
		Name:        oilField.Name,
		Lat:         oilField.Lat,
		Lon:         oilField.Lon,
		IsDeleted:   oilField.IsDeleted,
		CreatedTs:   oilField.CreatedTs,
		UpdatedTs:   oilField.UpdatedTs,
		IsOnline:    server.isOilFieldOnline(oilField.OilFieldId),
	}

	response.Response(l, w, oilFieldResult)
}

func (server *Server) controllersList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, _ := icontext.GetUser(ctx)
	l, _ := icontext.GetLogger(ctx)

	keys := r.URL.Query()
	oilFieldID, err := strconv.ParseInt(keys.Get("oilFieldId"), 10, 64)
	if err != nil {
		l.Errorf("%v", err)
		response.ErrorResponse(l, w, http.StatusForbidden, "Required oilFieldId", nil)
		return
	}

	oilField, err := server.db.GetOilField(ctx, oilFieldID)
	if err != nil {
		l.Errorf("%v", err)
		response.ErrorResponse(l, w, http.StatusForbidden, "Oil Field Not found", nil)
		return
	}

	if oilField.CompanyID != user.CompanyID && !user.IsSuperUser() {
		response.ErrorResponse(l, w, http.StatusForbidden, "Access denied", nil)
		return
	}

	output, err := server.db.GetControllers(ctx, oilFieldID)
	if err != nil {
		response.ErrorResponse(l, w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	response.Response(l, w, output)
}

func (server *Server) controllerData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, _ := icontext.GetUser(ctx)
	l, _ := icontext.GetLogger(ctx)

	input := models.SyncControllerDataRequest{}
	err := utils.ParseJson(r, &input)
	if err != nil {
		response.ErrorResponse(l, w, http.StatusInternalServerError, "Parse error", nil)
		return
	}

	fmt.Println("DIFF: ", input.DiffTime)

	if err = input.Validate(); err != nil {
		l.Errorf("%v", err)
		response.ErrorResponse(l, w, http.StatusForbidden, err.Error(), nil)
		return
	}

	controller, err := server.db.GetController(ctx, input.ControllerID)
	if err != nil {
		l.Errorf("%v", err)
		response.ErrorResponse(l, w, http.StatusForbidden, "Controller Not found", nil)
		return
	}

	oilField, err := server.db.GetOilField(ctx, controller.OilFieldId)
	if err != nil {
		l.Errorf("%v", err)
		response.ErrorResponse(l, w, http.StatusForbidden, "Controller Not found", nil)
		return
	}
	if !user.IsSuperUser() && oilField.CompanyID != user.CompanyID {
		response.ErrorResponse(l, w, http.StatusForbidden, "Access denied", nil)
		return
	}
	sensors, err := server.db.GetSensors(ctx, controller.ControllerID)
	if err != nil {
		l.Errorf("%v", err)
		response.ErrorResponse(l, w, http.StatusInternalServerError, "Parse error", nil)
		return
	}

	tags := make([]string, 0, 10)
	for _, sensor := range sensors {
		tags = append(tags, sensor.SensorId)
	}

	result := server.influxDB.GetMultipleTagsData(tags, input.SelectTime, input.DiffTime, input.GroupTime)
	result.Objects = sensors

	response.Response(l, w, result)
}

func (server *Server) mnemoschemesList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, _ := icontext.GetUser(ctx)
	l, _ := icontext.GetLogger(ctx)

	mnemoschemes, err := server.db.GetMnemoschemes(ctx, user.CompanyID, user.IsSuperUser())
	if err != nil {
		response.ErrorResponse(l, w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	response.Response(l, w, mnemoschemes)
}

func (server *Server) mnemoschemesSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, _ := icontext.GetUser(ctx)
	l, _ := icontext.GetLogger(ctx)

	mnemoID, _ := strconv.ParseInt(r.FormValue("mnemoId"), 10, 64)
	companyID, _ := strconv.ParseInt(r.FormValue("companyId"), 10, 64)
	l.Errorf("companyID: %d", companyID)
	l.Errorf("mnemoID: %d", mnemoID)

	model := models.MnemoResult{
		MnemoId:   mnemoID,
		CompanyId: companyID,
		Info:      r.FormValue("info"),
		Name:      r.FormValue("name"),
	}

	l.Errorf("Mnemo: %d", model.MnemoId)
	l.Errorf("Mnemo: %d", model.CompanyId)
	l.Errorf("Mnemo: %d", model.Info)
	l.Errorf("Mnemo: %d", model.Name)
	if mnemoID > 0 {
		mnemo, err := server.db.GetMnemoscheme(ctx, mnemoID)
		l.Errorf("Find mnemo ID: %v", err)
		if err != nil {
			response.ErrorResponse(l, w, http.StatusUnprocessableEntity, "Mnemo not found", nil)
			return
		}
		model.FileUrl = mnemo.FileUrl
	}

	err := model.Validate()
	if err != nil {
		l.Errorf("validate: %v", err)
		response.ErrorResponse(l, w, http.StatusUnprocessableEntity, err.Error(), nil)
		return
	}

	if user.CompanyID != companyID && !user.IsSuperUser() {
		response.ErrorResponse(l, w, http.StatusForbidden, "Access denied", nil)
		return
	}

	if upload.FileExists(r, "file") {
		if mnemoID > 0 {
			tempMnemo, err := server.db.GetMnemoscheme(ctx, mnemoID)
			if err == nil && len(tempMnemo.FileUrl) > 0 {
				l.Errorf("FileExists(919): %v", err)
				_ = upload.RemoveFile(tempMnemo.FileUrl[1:], l)
			}
		}

		path := "uploads/" + strconv.FormatInt(user.UserID, 10)
		fileData, err := upload.UploadFile(r, "file", path, l)
		if err != nil {
			l.Errorf("fileData(927): %v", err)
			fmt.Println("err: ", err.Error())
			response.ErrorResponse(l, w, 500, "Internal server error", nil)
			return
		}
		model.FileUrl = fileData.Url
	}
	//else if mnemoID == 0 {
	// 	l.Errorf("mnemoID equal 0(934): %v")
	// 	response.ErrorResponse(l, w, http.StatusUnprocessableEntity, "FILE NOT EXISTS", nil)
	// 	return
	// }

	mnemo, err := server.db.SaveMnemoschemes(ctx, model)
	if err != nil {
		l.Errorf("Save Mnemoschemes(940): %v", err)
		response.ErrorResponse(l, w, http.StatusUnprocessableEntity, err.Error(), nil)
		return
	}

	response.Response(l, w, mnemo)
}

func (server *Server) mnemoschemesData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l, _ := icontext.GetLogger(ctx)

	input := struct {
		SensorIDs []string `json:"sensorIds"`
	}{}
	err := utils.ParseJson(r, &input)
	if err != nil {
		fmt.Println(err.Error())
		response.ErrorResponse(l, w, http.StatusInternalServerError, "Parse error", nil)
		return
	}

	if len(input.SensorIDs) == 0 {
		response.ErrorResponse(l, w, http.StatusUnprocessableEntity, "Empty sensorIds", nil)
		return
	}

	latestSensorDatas := server.db.GetLatestSensorValues(ctx, server.influxDB, input.SensorIDs)
	response.Response(l, w, latestSensorDatas)
}

func (server *Server) mnemoschemesInfoSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, _ := icontext.GetUser(ctx)
	l, _ := icontext.GetLogger(ctx)

	input := struct {
		MnemoID   int64  `json:"mnemo_id"`
		CompanyID int64  `json:"company_id"`
		Info      string `json:"info"`
	}{}
	err := utils.ParseJson(r, &input)
	if err != nil {
		response.ErrorResponse(l, w, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	if user.CompanyID != input.CompanyID && !user.IsSuperUser() {
		response.ErrorResponse(l, w, http.StatusForbidden, "Access denied", nil)
		return
	}

	mnemo, err := server.db.SaveMnemoschemeInfo(ctx, input.MnemoID, input.Info)
	if err != nil {
		response.ErrorResponse(l, w, http.StatusUnprocessableEntity, err.Error(), nil)
		return
	}

	response.Response(l, w, mnemo)
}

func (server *Server) alarmsList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l, _ := icontext.GetLogger(ctx)
	user, _ := icontext.GetUser(ctx)

	alarms, err := server.db.GetAlarms(ctx, user.CompanyID, user.IsSuperUser())
	if err != nil {
		l.Errorf("%v", err)
		response.ErrorResponse(l, w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	response.Response(l, w, alarms)
}

func (server *Server) sensorsList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l, _ := icontext.GetLogger(ctx)
	user, _ := icontext.GetUser(ctx)
	input := models.SensorResult{}

	oilFields, err := server.db.GetOilFieldCheckingUser(ctx, user.CompanyID, user.IsSuperUser())

	if err != nil {
		response.ErrorResponse(l, w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	oilFieldID := make([]int64, 0, 10)

	for _, value := range oilFields {
		oilFieldID = append(oilFieldID, value.OilFieldId)
	}

	if oilFieldID == nil {
		response.ErrorResponse(l, w, http.StatusUnprocessableEntity, "oilfieldID Empty", nil)
		return
	}

	controllers, err := server.db.GetControllersArrayInputParam(ctx, oilFieldID, input.SensorId)
	if err != nil {
		l.Errorf("%v", err)
		response.ErrorResponse(l, w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	sensors := make([]*models.SensorResult, 0, 10)

	for _, value := range controllers {
		sensors = append(sensors, value.Sensors...)
	}

	response.Response(l, w, sensors)

}

func (server *Server) actionsList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l, _ := icontext.GetLogger(ctx)
	user, _ := icontext.GetUser(ctx)

	actions, err := server.db.GetActionsLogs(ctx, user.CompanyID, user.IsSuperUser())
	if err != nil {
		response.ErrorResponse(l, w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	response.Response(l, w, actions)
}

func (server *Server) markAlarmViewed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l, _ := icontext.GetLogger(ctx)
	input := struct {
		AlarmID int64 `json:"alarmId"`
	}{}
	err := utils.ParseJson(r, &input)
	if err != nil {
		fmt.Println(err.Error())
		response.ErrorResponse(l, w, http.StatusInternalServerError, "Parse error", nil)
		return
	}

	alarm, err := server.db.MarkAlarmAsViewed(ctx, input.AlarmID)
	if err != nil {
		response.ErrorResponse(l, w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	response.Response(l, w, alarm)
}

func (server *Server) files(w http.ResponseWriter, r *http.Request) {
	keys := r.URL.Query()
	filePath := keys.Get("path")
	http.ServeFile(w, r, filePath)
}

func (server *Server) isOilFieldOnline(oilFieldID int64) bool {
	if _, exists := server.MasterSocketConnectionsPool[oilFieldID]; exists {
		return true
	}

	return false
}

func (server *Server) masterSocketDisconnect(ctx context.Context, oilFieldID int64) {
	l, _ := icontext.GetLogger(ctx)
	l.Infof("Disconnect (oilFieldID: %s)", oilFieldID)
	fmt.Printf("Disconnect (oilFieldID: %s)\n", oilFieldID)
	masterConnection, exists := server.MasterSocketConnectionsPool[oilFieldID]
	if exists {
		masterConnection.Close()
		delete(server.MasterSocketConnectionsPool, oilFieldID)

		oilField, err := server.db.GetOilField(ctx, oilFieldID)
		if err != nil {
			return
		}

		oilFieldResult := &models.OilFieldResult{
			OilFieldId:  oilField.OilFieldId,
			HttpAddress: oilField.HttpAddress,
			CompanyID:   oilField.CompanyID,
			Name:        oilField.Name,
			Lat:         oilField.Lat,
			Lon:         oilField.Lon,
			IsDeleted:   oilField.IsDeleted,
			CreatedTs:   oilField.CreatedTs,
			UpdatedTs:   oilField.UpdatedTs,
			IsOnline:    server.isOilFieldOnline(oilField.OilFieldId),
		}

		users, err := server.db.GetUsers(ctx, oilField.CompanyID, true)
		if err != nil {
			server.logger.Errorf("Can't receive user contact list %s", err.Error())
		} else {
			for _, currentUser := range users {
				server.SendMessageTo(ctx, models.MessageTypeOilFieldOffline, oilFieldResult, currentUser.UserID)
			}
		}
	}
}

func (server *Server) listenOilField(ctx context.Context, oilFieldModel *models.OilField) {
	if server.isOilFieldOnline(oilFieldModel.OilFieldId) {
		return
	}

	conn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s/connectCloud?CloudID=%d", oilFieldModel.HttpAddress, oilFieldModel.OilFieldId), nil)
	if err != nil {
		//fmt.Printf("Dial failed: %s\n", err.Error())
		return
	}

	syncClient := NewSyncClient(oilFieldModel.OilFieldId, oilFieldModel.HttpAddress, conn, server, server.logger)
	syncClient.Run()
	server.MasterSocketConnectionsPool[oilFieldModel.OilFieldId] = syncClient

	oilField, err := server.db.GetOilField(ctx, oilFieldModel.OilFieldId)
	if err != nil {
		return
	}

	oilFieldResult := &models.OilFieldResult{
		OilFieldId:  oilField.OilFieldId,
		HttpAddress: oilField.HttpAddress,
		CompanyID:   oilField.CompanyID,
		Name:        oilField.Name,
		Lat:         oilField.Lat,
		Lon:         oilField.Lon,
		IsDeleted:   oilField.IsDeleted,
		CreatedTs:   oilField.CreatedTs,
		UpdatedTs:   oilField.UpdatedTs,
		IsOnline:    server.isOilFieldOnline(oilField.OilFieldId),
	}

	users, err := server.db.GetUsers(ctx, oilField.CompanyID, true)
	if err != nil {
		server.logger.Errorf("Can't receive user contact list %s", err.Error())
	} else {
		for _, currentUser := range users {
			server.SendMessageTo(ctx, models.MessageTypeOilFieldOnline, oilFieldResult, currentUser.UserID)
		}
	}
}

func (server *Server) disconnectOilField(ctx context.Context, oilFieldID int64) {
	masterConnection, exists := server.MasterSocketConnectionsPool[oilFieldID]
	if exists {
		masterConnection.Close()
		delete(server.MasterSocketConnectionsPool, oilFieldID)
	}
}

func (server *Server) CheckAlarms(alarms models.Alarms) {
	ctx := context.Background()
	guid := xid.New()
	requestID := guid.String()
	requestLogger := log.WithFields(log.Fields{"request_id": requestID})
	ctx = context.WithValue(ctx, icontext.LoggerContextKey, requestLogger)

	socketAlarms := server.db.SaveAlarms(ctx, alarms)
	if len(socketAlarms) > 0 {
		for _, alarm := range socketAlarms {
			server.SendMessageTo(ctx, models.MessageTypeAlarm, alarm, alarm.UserID)
		}
	}
}

func (server *Server) NewIncomingMessage(
	ctx context.Context,
	message *models.InputMessage,
	oilFieldId int64,
) {
	server.newIncomingMessage <- &incomingMessageWithContext{
		Context:    ctx,
		Message:    message,
		OilFieldId: oilFieldId,
	}
}

func (server *Server) SendMessageTo(ctx context.Context, messageType string, body interface{}, userID int64) {
	connections, exists := server.SocketConnectionsPool[userID]
	if !exists {
		return
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		server.logger.Errorf("Can't marshal send message body: %+v", body)
	}

	message := &models.OutputMessage{
		Type:      messageType,
		Timestamp: time.Now(),
		Body:      bodyBytes,
	}

	for _, conn := range connections {
		conn.SendMessage(message)
	}
	return
}

func (server *Server) SendMessageOilField(messageType string, body interface{}, oilFieldId int64) {
	cc, exists := server.MasterSocketConnectionsPool[oilFieldId]
	if !exists {
		return
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		server.logger.Errorf("Can't marshal send message body: %+v", body)
	}

	fmt.Println("SEND MESSAGE: ", string(bodyBytes))
	message := &models.OutputMessage{
		Type:      messageType,
		Timestamp: time.Now(),
		Body:      bodyBytes,
	}

	cc.SendMessage(message)
	fmt.Println("MESSAGE SENT!")
	return
}

func (server *Server) socketDisconnect(ctx context.Context, userID int64, connectionID string) {
	l, _ := icontext.GetLogger(ctx)
	l.Infof("Disconnect (userId: %d, connectionID: %s)", userID, connectionID)
	userConnections, exists := server.SocketConnectionsPool[userID]

	if exists {
		indexToRemove := -1
		for k, v := range userConnections {
			if v.ID == connectionID {
				indexToRemove = k
				v.Close()
			}
		}

		if indexToRemove != -1 {
			copy(userConnections[indexToRemove:], userConnections[indexToRemove+1:])
			userConnections[len(userConnections)-1] = nil
			userConnections = userConnections[:len(userConnections)-1]

			if len(userConnections) > 0 {
				server.SocketConnectionsPool[userID] = userConnections
			} else {
				delete(server.SocketConnectionsPool, userID)

				user, err := server.db.GetUser(ctx, userID)
				if err != nil {
					l.Errorf("Can't receive user %s", err.Error())
					return
				}
				userResult := models.UserResult{
					UserID:    user.UserID,
					CompanyID: user.CompanyID,
					Email:     user.Email,
					FirstName: user.FirstName,
					LastName:  user.LastName,
					Role:      user.GetResultRole(),
					IsDeleted: user.IsDeleted,
					IsOnline:  server.isUserOnline(user.UserID),
					CreatedTs: user.CreatedTs,
					UpdatedTs: user.UpdatedTs,
				}

				users, err := server.db.GetUsers(ctx, user.CompanyID, user.IsSuperUser())
				if err != nil {
					l.Errorf("Can't receive user contact list %s", err.Error())
				} else {
					for _, currentUser := range users {
						if currentUser.UserID == user.UserID {
							continue
						}
						server.SendMessageTo(ctx, models.MessageTypeUserOffline, userResult, currentUser.UserID)
					}
				}
			}
		}
	}
}

func (server *Server) isUserOnline(userID int64) bool {
	if _, exists := server.SocketConnectionsPool[userID]; exists {
		return true
	}
	return false
}
