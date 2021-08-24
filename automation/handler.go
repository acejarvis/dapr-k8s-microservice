package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type CreateRequest struct {
	Deployment map[string]interface{} `json:"deployment"`
	Connect    DCSConnectRequest      `json:"connect"`
}

type DeleteRequest struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type DCSConnectRequest struct {
	DCSName    string `json:"dcsName"`
	Credential string `json:"credential"` // DCS connect password if have one
	AK         string `json:"ak"`         // base64 encoded AK
	SK         string `json:"sk"`         // base64 encoded SK
}

func (s *Server) HandleHeathCheck(w http.ResponseWriter, r *http.Request) {
	log.Println("HandleHeathCheck")
}

func (s *Server) HandleHelloWorld(w http.ResponseWriter, r *http.Request) {
	log.Println("HandleHelloWorld")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello Dapr K8s!"))
}

func (s *Server) HandleCreate(w http.ResponseWriter, r *http.Request) {
	log.Println("HandleCreate")
	var req CreateRequest
	json.NewDecoder(r.Body).Decode(&req)
	log.Println(req)
	result, err := s.kubeClient.CreateAppDeploy(req.Deployment, &req.Connect)
	if err != nil {
		HandleInternalServerError(w, err)
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(result))
	}
}

func (s *Server) HandleDelete(w http.ResponseWriter, r *http.Request) {
	log.Println("HandleDelete")
	var req DeleteRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		HandleInternalServerError(w, err)
	}
	log.Println(req)

	result, err := s.kubeClient.DeleteAppDeploy(req.Namespace, req.Name)
	if err != nil {
		HandleNotFound(w, err)
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(result))
	}
}

func (s *Server) HandleConnect(w http.ResponseWriter, r *http.Request) {
	log.Println("HandleConnect")
	var req DCSConnectRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		HandleInternalServerError(w, err)
	}
	log.Println(req)
	result, err := s.kubeClient.ConnectRedis(&req)
	if err != nil {
		HandleInternalServerError(w, err)
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Dapr StateStore Connected, \n " + result))
	}
}

func (s *Server) HandleDisconnect(w http.ResponseWriter, r *http.Request) {
	log.Println("HandleDisconnect")
	result, err := s.kubeClient.DisconnectRedis()
	if err != nil {
		HandleNotFound(w, err)
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(result))
	}
}

func HandleInternalServerError(w http.ResponseWriter, err error) {
	log.Println(err)
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(err.Error()))
}

func HandleNotFound(w http.ResponseWriter, err error) {
	log.Println(err)
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(err.Error()))
}
