package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type AppCreateRequest struct {
	Deployment map[string]interface{} `json:"deployment"` // Kubernetes App Deployment template
	DCSConnect DCSConnectRequest      `json:"dcsConnect"`
}

type AppDeleteRequest struct {
	Namespace     string               `json:"namespace"`
	Name          string               `json:"name"`
	DCSDisconnect DCSDisconnectRequest `json:"dcsDisconnect"`
}

type DCSDisconnectRequest struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type DCSConnectRequest struct {
	DCSName    string `json:"dcsName"`    // Huaweicloud DCS name
	Credential string `json:"credential"` // base64 encoded DCS connect password, leave empty if your DCS does not have one
	AK         string `json:"ak"`         // base64 encoded AK
	SK         string `json:"sk"`         // base64 encoded SK
	Namespace  string `json:"namespace"`  // Kubernetes Namespace
	Name       string `json:"name"`       // Dapr/Kubernetes resource name
}

func (s *Server) HandleHeathCheck(w http.ResponseWriter, r *http.Request) {
	log.Println("HandleHeathCheck")
}

func (s *Server) HandleHelloWorld(w http.ResponseWriter, r *http.Request) {
	log.Println("HandleHelloWorld")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello Dapr K8s!"))
}

// Deploy App on Dapr
func (s *Server) HandleAppCreate(w http.ResponseWriter, r *http.Request) {
	log.Println("HandleAppCreate")
	var req AppCreateRequest
	json.NewDecoder(r.Body).Decode(&req)
	log.Println(req)
	result, err := s.kubeClient.CreateAppDeploy(&req)
	if err != nil {
		HandleInternalServerError(w, err)
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(result))
	}
}

// Delete App on Dapr
func (s *Server) HandleAppDelete(w http.ResponseWriter, r *http.Request) {
	log.Println("HandleAppDelete")
	var req AppDeleteRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		HandleInternalServerError(w, err)
	}
	log.Println(req)

	result, err := s.kubeClient.DeleteAppDeploy(&req)
	if err != nil {
		HandleNotFound(w, err)
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(result))
	}
}

// Connect DCS to Dapr
func (s *Server) HandleDCSConnect(w http.ResponseWriter, r *http.Request) {
	log.Println("HandleDCSConnect")
	var req DCSConnectRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		HandleInternalServerError(w, err)
	}
	log.Println(req)
	result, err := s.kubeClient.ConnectDCS(&req)
	if err != nil {
		HandleInternalServerError(w, err)
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Dapr StateStore Connected, \n " + result))
	}
}

// Disconnect DCS from Dapr
func (s *Server) HandleDCSDisconnect(w http.ResponseWriter, r *http.Request) {
	log.Println("HandleDCSDisconnect")
	var req DCSDisconnectRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		HandleInternalServerError(w, err)
	}
	log.Println(req)
	result, err := s.kubeClient.DisconnectDCS(&req)
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
