package main

import (
	"encoding/json"
	"log"
	"net/http"
)

var Servers []Server

type SmscSessionStatus struct {
	SmscName         string
	NumberOfSessions int
	Port             int
}

type App struct {
	smppServers []*Server
}

func createApp() *App {
	var servers = make([]*Server, 12)
	for i := 0; i < 4; i++ {
		log.Printf("Creating (%s) port (%d) \n", "Stc", 2000+i)
		servers[i] = NewDefaultServer("Stc", 2000+i)
		//servers[i].Handler = servers[i].DefaultHandler
		servers[i].Handler = servers[i].SecondHandler
		servers[i].Start()
	}
	for i := 4; i < 8; i++ {
		log.Printf("Creating (%s) port (%d) \n", "Mobily", 3000+i)
		servers[i] = NewDefaultServer("Mobily", 3000+i)
		//servers[i].Handler = servers[i].DefaultHandler
		servers[i].Handler = servers[i].SecondHandler
		servers[i].Start()
	}
	for i := 8; i < 12; i++ {
		log.Printf("Creating (%s) port (%d) \n", "Zain", 4000+i)
		servers[i] = NewDefaultServer("Zain", 4001+i)
		//servers[i].Handler = servers[i].DefaultHandler
		servers[i].Handler = servers[i].SecondHandler
		servers[i].Start()
	}
	return &App{
		smppServers: servers,
	}
}

func (app *App) ListPorts(w http.ResponseWriter, r *http.Request) { // <<<
	var status []SmscSessionStatus
	log.Println("From inside the ListPorts")
	log.Println(app.smppServers)
	for i := 0; i < len(app.smppServers); i++ {
		if app.smppServers[i].Conns == nil {
			log.Println("No Session Connected yet")
		}
		status = append(status, SmscSessionStatus{SmscName: app.smppServers[i].SmscName, NumberOfSessions: len(app.smppServers[i].Conns), Port: app.smppServers[i].Port})
	}
	js, _ := json.Marshal(status)
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func (app *App) FreezeSmsc(w http.ResponseWriter, r *http.Request) { // <<<
	smsc := r.FormValue("smsc")
	if smsc == "" {
		w.WriteHeader(400)
		w.Write([]byte("smsc param is required"))
		return
	}
	for i := 0; i < len(app.smppServers); i++ {
		if app.smppServers[i].SmscName == smsc {
			app.smppServers[i].Freeze()
		}
	}
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (app *App) UnFreezeSmsc(w http.ResponseWriter, r *http.Request) { // <<<
	smsc := r.FormValue("smsc")
	if smsc == "" {
		w.WriteHeader(400)
		w.Write([]byte("smsc param is required"))
		return
	}
	for i := 0; i < len(app.smppServers); i++ {
		if app.smppServers[i].SmscName == smsc {
			app.smppServers[i].UnFreeze()
		}
	}
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (app *App) ListCurrentSenders(w http.ResponseWriter, r *http.Request) { // <<< ## TODO
	port := r.FormValue("port")
	sess := r.FormValue("session")
	if port == "" || sess == "" {
		w.WriteHeader(400)
		w.Write([]byte("smsc param is required"))
		return
	}
	// This should be implemented to notify the ui the current traffic routed to this belongs to which smsc
	// We should just detect from simple fild i.e. sender
	w.WriteHeader(501)
	w.Write([]byte("Not implemented yet"))
}

func (app *App) UnbindAllSessionsForSmcs(w http.ResponseWriter, r *http.Request) { // <<<
	smsc := r.FormValue("smsc")
	if smsc == "" {
		w.WriteHeader(400)
		w.Write([]byte("smsc param is required"))
		return
	}
	for i := 0; i < len(app.smppServers); i++ {
		if app.smppServers[i].SmscName == smsc {
			app.smppServers[i].UnBind()
		}
	}
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

/*
func (app *App) CloseSession(w http.ResponseWriter, r *http.Request) { // <<<
	port := r.FormValue("port")
	sess := r.FormValue("session")
}
*/

func ctrlSessions(w http.ResponseWriter, r *http.Request) { // <<< ## TODO
	port := r.FormValue("port")
	sess := r.FormValue("sesson")
	if port == "" || sess == "" {
		w.WriteHeader(400)
		w.Write([]byte("smsc param is required"))
		return
	}
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func main() {
	done := make(chan bool, 1)
	log.Println("Starting new app bool")
	testSmscApp := createApp()
	http.HandleFunc("/listports", testSmscApp.ListPorts)
	http.HandleFunc("/freezesmsc", testSmscApp.FreezeSmsc)
	http.HandleFunc("/unfreezesmsc", testSmscApp.UnFreezeSmsc)
	http.HandleFunc("/listSenders", testSmscApp.ListCurrentSenders)
	http.HandleFunc("/unbindall", testSmscApp.UnbindAllSessionsForSmcs)
	log.Println("Start listening and serve for http")
	http.ListenAndServe(":10000", nil)

	<-done

}
